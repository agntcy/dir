// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package cosign

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"time"

	signv1 "github.com/agntcy/dir/api/sign/v1"
	"github.com/sigstore/cosign/v3/pkg/cosign"
	v1 "github.com/sigstore/protobuf-specs/gen/pb-go/trustroot/v1"
	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/sign"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	DefaultFulcioTimeout             = 30 * time.Second
	DefaultTimestampAuthorityTimeout = 30 * time.Second
	DefaultRekorTimeout              = 90 * time.Second
)

// SignBlobWithOIDC signs a blob using OIDC authentication.
func SignBlobWithOIDC(ctx context.Context, payload []byte, req *signv1.SignWithOIDC) (*signv1.Signature, *signv1.PublicKey, error) {
	signingTime := time.Now()

	// Get signing options from configuration
	signOpts, err := getOIDCSigningOptions(ctx, req, signingTime)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get signing options: %w", err)
	}

	// Generate an ephemeral keypair for signing.
	signKeypair, err := sign.NewEphemeralKeypair(nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create ephemeral keypair: %w", err)
	}

	// Get the public key in PEM format to return to the client.
	publicKeyPEM, err := signKeypair.GetPublicKeyPem()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get public key: %w", err)
	}

	// Sign the payload
	sigBundle, err := sign.Bundle(&sign.PlainData{Data: payload}, signKeypair, signOpts)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to sign record: %w", err)
	}

	// Marshal bundle to JSON using protobuf
	sigBundleJSON, err := protojson.Marshal(sigBundle)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal bundle to JSON: %w", err)
	}

	return &signv1.Signature{
			Signature:     base64.StdEncoding.EncodeToString(sigBundle.GetMessageSignature().GetSignature()),
			Certificate:   base64.StdEncoding.EncodeToString(sigBundle.GetVerificationMaterial().GetCertificate().GetRawBytes()),
			ContentType:   sigBundle.GetMediaType(),
			ContentBundle: string(sigBundleJSON),
			SignedAt:      signingTime.UTC().Format(time.RFC3339),
		}, &signv1.PublicKey{
			Key: publicKeyPEM,
		}, nil
}

// SignBlobWithKey signs a blob using a private key.
func SignBlobWithKey(_ context.Context, payload []byte, req *signv1.SignWithKey) (*signv1.Signature, *signv1.PublicKey, error) {
	// Load private key
	sv, err := cosign.LoadPrivateKey(req.GetPrivateKey(), req.GetPassword(), nil)
	if err != nil {
		return nil, nil, fmt.Errorf("loading private key: %w", err)
	}

	// Sign the message
	sig, err := sv.SignMessage(bytes.NewReader(payload))
	if err != nil {
		return nil, nil, fmt.Errorf("signing blob: %w", err)
	}

	// Get public key
	pubKey, err := sv.PublicKey()
	if err != nil {
		return nil, nil, fmt.Errorf("getting public key: %w", err)
	}

	publicKeyPEM, err := cryptoutils.MarshalPublicKeyToPEM(pubKey)
	if err != nil {
		return nil, nil, fmt.Errorf("getting public key: %w", err)
	}

	return &signv1.Signature{
			SignedAt:  time.Now().UTC().Format(time.RFC3339),
			Signature: base64.StdEncoding.EncodeToString(sig),
			Algorithm: detectKeyAlgorithm(string(publicKeyPEM)),
		}, &signv1.PublicKey{
			Key: string(publicKeyPEM),
		}, nil
}

// getOIDCSigningOptions returns bundle options configured from SignOptionsOIDC.
func getOIDCSigningOptions(ctx context.Context, req *signv1.SignWithOIDC, signatureTime time.Time) (sign.BundleOptions, error) {
	opts := req.GetOptions().GetDefaultOptions()

	// Construct signing config from request
	signingConfig, err := root.NewSigningConfig(
		root.SigningConfigMediaType02,
		// Fulcio URLs
		[]root.Service{
			{
				URL:                 opts.GetFulcioUrl(),
				MajorAPIVersion:     1,
				ValidityPeriodStart: signatureTime.Add(-time.Hour),
				ValidityPeriodEnd:   signatureTime.Add(time.Hour),
			},
		},
		// OIDC Provider URLs
		[]root.Service{
			{
				URL:                 opts.GetOidcProviderUrl(),
				MajorAPIVersion:     1,
				ValidityPeriodStart: signatureTime.Add(-time.Hour),
				ValidityPeriodEnd:   signatureTime.Add(time.Hour),
			},
		},
		// Rekor URLs
		[]root.Service{
			{
				URL:                 opts.GetRekorUrl(),
				MajorAPIVersion:     1,
				ValidityPeriodStart: signatureTime.Add(-time.Hour),
				ValidityPeriodEnd:   signatureTime.Add(time.Hour),
			},
		},
		root.ServiceConfiguration{
			Selector: v1.ServiceSelector_ANY,
		},
		// TSA URLs
		[]root.Service{
			{
				URL:                 opts.GetTimestampUrl(),
				MajorAPIVersion:     1,
				ValidityPeriodStart: signatureTime.Add(-time.Hour),
				ValidityPeriodEnd:   signatureTime.Add(time.Hour),
			},
		},
		root.ServiceConfiguration{
			Selector: v1.ServiceSelector_ANY,
		},
	)
	if err != nil {
		return sign.BundleOptions{}, fmt.Errorf("failed to get signing config: %w", err)
	}

	// Get signing options based on the signing config
	signOpts := sign.BundleOptions{
		Context: ctx,
		CertificateProviderOptions: &sign.CertificateProviderOptions{
			IDToken: req.GetIdToken(),
		},
	}
	{
		// Configure Fulcio certificate provider
		fulcioURL, err := root.SelectService(signingConfig.FulcioCertificateAuthorityURLs(), []uint32{1}, signatureTime)
		if err != nil {
			return sign.BundleOptions{}, fmt.Errorf("failed to select fulcio URL: %w", err)
		}

		signOpts.CertificateProvider = sign.NewFulcio(&sign.FulcioOptions{
			BaseURL: fulcioURL.URL,
			Timeout: DefaultFulcioTimeout,
			Retries: 1,
		})

		// Configure timestamp authorities
		tsaURLs, err := root.SelectServices(signingConfig.TimestampAuthorityURLs(),
			signingConfig.TimestampAuthorityURLsConfig(), []uint32{1}, signatureTime)
		if err != nil {
			return sign.BundleOptions{}, fmt.Errorf("failed to select timestamp authority URL: %w", err)
		}

		for _, tsaURL := range tsaURLs {
			signOpts.TimestampAuthorities = append(signOpts.TimestampAuthorities,
				sign.NewTimestampAuthority(&sign.TimestampAuthorityOptions{
					URL:     tsaURL.URL,
					Timeout: DefaultTimestampAuthorityTimeout,
					Retries: 1,
				}))
		}

		// Configure Rekor transparency logs (unless skip_tlog is set)
		if !opts.GetSkipTlog() {
			rekorURLs, err := root.SelectServices(signingConfig.RekorLogURLs(),
				signingConfig.RekorLogURLsConfig(), []uint32{1}, signatureTime)
			if err != nil {
				return sign.BundleOptions{}, fmt.Errorf("failed to select rekor URL: %w", err)
			}

			for _, rekorURL := range rekorURLs {
				signOpts.TransparencyLogs = append(signOpts.TransparencyLogs,
					sign.NewRekor(&sign.RekorOptions{
						BaseURL: rekorURL.URL,
						Timeout: DefaultRekorTimeout,
						Retries: 1,
					}))
			}
		}
	}

	return signOpts, nil
}
