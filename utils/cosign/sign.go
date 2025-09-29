// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package cosign

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/sigstore/cosign/v2/pkg/cosign"
	"github.com/sigstore/cosign/v2/pkg/oci/mutate"
	ociremote "github.com/sigstore/cosign/v2/pkg/oci/remote"
	"github.com/sigstore/cosign/v2/pkg/oci/static"
	v1 "github.com/sigstore/protobuf-specs/gen/pb-go/trustroot/v1"
	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/sign"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
)

const (
	DefaultFulcioURL       = "https://fulcio.sigstage.dev"
	DefaultRekorURL        = "https://rekor.sigstage.dev"
	DefaultTimestampURL    = "https://timestamp.sigstage.dev/api/v1/timestamp"
	DefaultOIDCProviderURL = "https://oauth2.sigstage.dev/auth"
	DefaultOIDCClientID    = "sigstore"

	DefaultFulcioTimeout             = 30 * time.Second
	DefaultTimestampAuthorityTimeout = 30 * time.Second
	DefaultRekorTimeout              = 90 * time.Second
)

// SetOrDefault returns the value if it's not empty, otherwise returns the default value.
func SetOrDefault(value string, defaultValue string) string {
	if value == "" {
		value = defaultValue
	}

	return value
}

// SignBlobOIDCOptions contains options for OIDC-based blob signing.
type SignBlobOIDCOptions struct {
	Payload         []byte
	IDToken         string
	FulcioURL       string
	RekorURL        string
	TimestampURL    string
	OIDCProviderURL string
}

// SignBlobOIDCResult contains the result of OIDC blob signing.
type SignBlobOIDCResult struct {
	Signature string
	PublicKey string
}

// SignBlobWithOIDC signs a blob using OIDC authentication.
func SignBlobWithOIDC(_ context.Context, opts *SignBlobOIDCOptions) (*SignBlobOIDCResult, error) {
	// Load signing options.
	var signOpts sign.BundleOptions
	{
		// Define config to use for signing.
		signingConfig, err := root.NewSigningConfig(
			root.SigningConfigMediaType02,
			// Fulcio URLs
			[]root.Service{
				{
					URL:                 setOrDefault(opts.FulcioURL, DefaultFulcioURL),
					MajorAPIVersion:     1,
					ValidityPeriodStart: time.Now().Add(-time.Hour),
					ValidityPeriodEnd:   time.Now().Add(time.Hour),
				},
			},
			// OIDC Provider URLs
			// Usage and requirements: https://docs.sigstore.dev/certificate_authority/oidc-in-fulcio/
			[]root.Service{
				{
					URL:                 setOrDefault(opts.OIDCProviderURL, DefaultOIDCProviderURL),
					MajorAPIVersion:     1,
					ValidityPeriodStart: time.Now().Add(-time.Hour),
					ValidityPeriodEnd:   time.Now().Add(time.Hour),
				},
			},
			// Rekor URLs
			[]root.Service{
				{
					URL:                 setOrDefault(opts.RekorURL, DefaultRekorURL),
					MajorAPIVersion:     1,
					ValidityPeriodStart: time.Now().Add(-time.Hour),
					ValidityPeriodEnd:   time.Now().Add(time.Hour),
				},
			},
			root.ServiceConfiguration{
				Selector: v1.ServiceSelector_ANY,
			},
			[]root.Service{
				{
					URL:                 setOrDefault(opts.TimestampURL, DefaultTimestampURL),
					MajorAPIVersion:     1,
					ValidityPeriodStart: time.Now().Add(-time.Hour),
					ValidityPeriodEnd:   time.Now().Add(time.Hour),
				},
			},
			root.ServiceConfiguration{
				Selector: v1.ServiceSelector_ANY,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create signing config: %w", err)
		}

		// Use fulcio to sign.
		fulcioURL, err := root.SelectService(signingConfig.FulcioCertificateAuthorityURLs(), []uint32{1}, time.Now())
		if err != nil {
			return nil, fmt.Errorf("failed to select fulcio URL: %w", err)
		}

		fulcioOpts := &sign.FulcioOptions{
			BaseURL: fulcioURL,
			Timeout: DefaultFulcioTimeout,
			Retries: 1,
		}
		signOpts.CertificateProvider = sign.NewFulcio(fulcioOpts)
		signOpts.CertificateProviderOptions = &sign.CertificateProviderOptions{
			IDToken: opts.IDToken,
		}

		// Use timestamp authortiy to sign.
		tsaURLs, err := root.SelectServices(signingConfig.TimestampAuthorityURLs(),
			signingConfig.TimestampAuthorityURLsConfig(), []uint32{1}, time.Now())
		if err != nil {
			return nil, fmt.Errorf("failed to select timestamp authority URL: %w", err)
		}

		for _, tsaURL := range tsaURLs {
			tsaOpts := &sign.TimestampAuthorityOptions{
				URL:     tsaURL,
				Timeout: DefaultTimestampAuthorityTimeout,
				Retries: 1,
			}
			signOpts.TimestampAuthorities = append(signOpts.TimestampAuthorities, sign.NewTimestampAuthority(tsaOpts))
		}

		// Use rekor to sign.
		rekorURLs, err := root.SelectServices(signingConfig.RekorLogURLs(),
			signingConfig.RekorLogURLsConfig(), []uint32{1}, time.Now())
		if err != nil {
			return nil, fmt.Errorf("failed to select rekor URL: %w", err)
		}

		for _, rekorURL := range rekorURLs {
			rekorOpts := &sign.RekorOptions{
				BaseURL: rekorURL,
				Timeout: DefaultRekorTimeout,
				Retries: 1,
			}
			signOpts.TransparencyLogs = append(signOpts.TransparencyLogs, sign.NewRekor(rekorOpts))
		}
	}

	// Generate an ephemeral keypair for signing.
	signKeypair, err := sign.NewEphemeralKeypair(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create ephemeral keypair: %w", err)
	}

	// Sign the record JSON data.
	sigBundle, err := sign.Bundle(&sign.PlainData{Data: opts.Payload}, signKeypair, signOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to sign record: %w", err)
	}

	publicKeyPEM, err := signKeypair.GetPublicKeyPem()
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %w", err)
	}

	return &SignBlobOIDCResult{
		Signature: base64.StdEncoding.EncodeToString(sigBundle.GetMessageSignature().GetSignature()),
		PublicKey: publicKeyPEM,
	}, nil
}

// SignBlobKeyOptions contains options for key-based blob signing.
type SignBlobKeyOptions struct {
	Payload    []byte
	PrivateKey []byte
	Password   []byte
}

// SignBlobKeyResult contains the result of key-based blob signing.
type SignBlobKeyResult struct {
	Signature string
	PublicKey string
}

// SignBlobWithKey signs a blob using a private key.
func SignBlobWithKey(_ context.Context, opts *SignBlobKeyOptions) (*SignBlobKeyResult, error) {
	payload := bytes.NewReader(opts.Payload)

	sv, err := cosign.LoadPrivateKey(opts.PrivateKey, opts.Password)
	if err != nil {
		return nil, fmt.Errorf("loading private key: %w", err)
	}

	sig, err := sv.SignMessage(payload)
	if err != nil {
		return nil, fmt.Errorf("signing blob: %w", err)
	}

	pubKey, err := sv.PublicKey()
	if err != nil {
		return nil, fmt.Errorf("getting public key: %w", err)
	}

	publicKeyPEM, err := cryptoutils.MarshalPublicKeyToPEM(pubKey)
	if err != nil {
		return nil, fmt.Errorf("getting public key: %w", err)
	}

	return &SignBlobKeyResult{
		Signature: base64.StdEncoding.EncodeToString(sig),
		PublicKey: string(publicKeyPEM),
	}, nil
}

// AttachSignatureOptions contains options for attaching signatures to OCI images.
type AttachSignatureOptions struct {
	ImageRef  string
	Signature string
	Payload   string
	Username  string
	Password  string
}

// AttachSignature attaches a signature to an OCI image using cosign.
func AttachSignature(_ context.Context, opts *AttachSignatureOptions) error {
	ref, err := name.ParseReference(opts.ImageRef)
	if err != nil {
		return fmt.Errorf("failed to parse image reference: %w", err)
	}

	sig, err := static.NewSignature([]byte(opts.Payload), opts.Signature)
	if err != nil {
		return fmt.Errorf("failed to create static signature: %w", err)
	}

	// Remote options for authentication
	remoteOpts := []ociremote.Option{}
	if opts.Username != "" && opts.Password != "" {
		remoteOpts = append(remoteOpts, ociremote.WithRemoteOptions(
			remote.WithAuth(
				&authn.Basic{
					Username: opts.Username,
					Password: opts.Password,
				},
			),
		))
	}

	se, err := ociremote.SignedEntity(ref, remoteOpts...)
	if err != nil {
		return fmt.Errorf("failed to create signed entity: %w", err)
	}

	// Attach the signature to the entity.
	newSE, err := mutate.AttachSignatureToEntity(se, sig)
	if err != nil {
		return fmt.Errorf("failed to attach signature to entity: %w", err)
	}

	digest, err := ociremote.ResolveDigest(ref, remoteOpts...)
	if err != nil {
		return fmt.Errorf("resolving digest: %w", err)
	}

	err = ociremote.WriteSignaturesExperimentalOCI(digest, newSE, remoteOpts...)
	if err != nil {
		return fmt.Errorf("failed to write signatures: %w", err)
	}

	return nil
}

// GenerateKeyPairOptions contains options for generating cosign key pairs.
type GenerateKeyPairOptions struct {
	Directory string
	Password  string
}

// GenerateKeyPair generates a cosign key pair in the specified directory.
func GenerateKeyPair(ctx context.Context, opts *GenerateKeyPairOptions) error {
	cmd := exec.CommandContext(ctx, "cosign", "generate-key-pair")

	if opts.Directory != "" {
		cmd.Dir = opts.Directory
	}

	if opts.Password != "" {
		cmd.Env = append(os.Environ(), "COSIGN_PASSWORD="+opts.Password)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("cosign generate-key-pair failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

func setOrDefault(value string, defaultValue string) string {
	if value == "" {
		value = defaultValue
	}

	return value
}
