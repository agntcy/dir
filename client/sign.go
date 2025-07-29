// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
	"github.com/agntcy/dir/utils/cosign"
	v1 "github.com/sigstore/protobuf-specs/gen/pb-go/trustroot/v1"
	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/sign"
	"google.golang.org/protobuf/encoding/protojson"
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

type SignOpts struct {
	FulcioURL       string
	RekorURL        string
	TimestampURL    string
	OIDCProviderURL string
	OIDCClientID    string
	OIDCToken       string
	Key             string
}

// SignOIDC signs the record using keyless OIDC service-based signing.
// The OIDC ID Token must be provided by the caller.
// An ephemeral keypair is generated for signing.
func (c *Client) SignWithOIDC(ctx context.Context, req *signv1.SignRequest) (*signv1.SignResponse, error) {
	// Validate request.
	if req.GetRecord() == nil {
		return nil, errors.New("record must be set")
	}

	oidcSigner := req.GetProvider().GetOidc()

	// Load signing options.
	var signOpts sign.BundleOptions
	{
		// Define config to use for signing.
		signingConfig, err := root.NewSigningConfig(
			root.SigningConfigMediaType02,
			// Fulcio URLs
			[]root.Service{
				{
					URL:                 setOrDefault(oidcSigner.GetOptions().GetFulcioUrl(), DefaultFulcioURL),
					MajorAPIVersion:     1,
					ValidityPeriodStart: time.Now().Add(-time.Hour),
					ValidityPeriodEnd:   time.Now().Add(time.Hour),
				},
			},
			// OIDC Provider URLs
			// Usage and requirements: https://docs.sigstore.dev/certificate_authority/oidc-in-fulcio/
			[]root.Service{
				{
					URL:                 setOrDefault(oidcSigner.GetOptions().GetOidcProviderUrl(), DefaultOIDCProviderURL),
					MajorAPIVersion:     1,
					ValidityPeriodStart: time.Now().Add(-time.Hour),
					ValidityPeriodEnd:   time.Now().Add(time.Hour),
				},
			},
			// Rekor URLs
			[]root.Service{
				{
					URL:                 setOrDefault(oidcSigner.GetOptions().GetRekorUrl(), DefaultRekorURL),
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
					URL:                 setOrDefault(oidcSigner.GetOptions().GetTimestampUrl(), DefaultTimestampURL),
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

		// Use fulcio to sign the agent.
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
			IDToken: oidcSigner.GetIdToken(),
		}

		// Use timestamp authortiy to sign the agent.
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

		// Use rekor to sign the agent.
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

	signature, err := c.sign(ctx, req.GetRecord(), signKeypair, signOpts)
	if err != nil {
		return nil, err
	}

	return &signv1.SignResponse{
		Signature: signature,
	}, nil
}

func (c *Client) SignWithKey(ctx context.Context, req *signv1.SignRequest) (*signv1.SignResponse, error) {
	keySigner := req.GetProvider().GetKey()

	// Generate a keypair from the provided private key bytes.
	// The keypair hint is derived from the public key and will be used for verification.
	signKeypair, err := cosign.LoadKeypair(keySigner.GetPrivateKey(), keySigner.GetPassword())
	if err != nil {
		return nil, fmt.Errorf("failed to create keypair: %w", err)
	}

	signature, err := c.sign(ctx, req.GetRecord(), signKeypair, sign.BundleOptions{})
	if err != nil {
		return nil, err
	}

	return &signv1.SignResponse{
		Signature: signature,
	}, nil
}

func (c *Client) sign(_ context.Context, record *corev1.Record, signKeypair sign.Keypair, signOpts sign.BundleOptions) (*signv1.Signature, error) {
	// Convert the record to JSON.
	recordJSON, err := json.Marshal(record)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal record: %w", err)
	}

	// Sign the record JSON data.
	sigBundle, err := sign.Bundle(&sign.PlainData{Data: recordJSON}, signKeypair, signOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to sign record: %w", err)
	}

	certData := sigBundle.GetVerificationMaterial()
	sigData := sigBundle.GetMessageSignature()

	// Extract data from the signature bundle.
	sigBundleJSON, err := protojson.Marshal(sigBundle)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal bundle: %w", err)
	}

	// Update the agent with the signature details.
	signature := &signv1.Signature{
		Algorithm:     sigData.GetMessageDigest().GetAlgorithm().String(),
		Signature:     base64.StdEncoding.EncodeToString(sigData.GetSignature()),
		Certificate:   base64.StdEncoding.EncodeToString(certData.GetCertificate().GetRawBytes()),
		ContentType:   sigBundle.GetMediaType(),
		ContentBundle: base64.StdEncoding.EncodeToString(sigBundleJSON),
		SignedAt:      time.Now().Format(time.RFC3339),
	}

	return signature, nil
}

func setOrDefault(value string, defaultValue string) string {
	if value == "" {
		value = defaultValue
	}

	return value
}
