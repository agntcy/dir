// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:mnd,wsl
package client

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
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
)

type SignOpts struct {
	FulcioURL       string
	RekorURL        string
	TimestampURL    string
	OIDCProviderURL string
	OIDCClientID    string
	Key             string
}

// SignOIDC signs the agent using keyless OIDC service-based signing.
// The OIDC ID Token must be provided by the caller.
// An ephemeral keypair is generated for signing.
func (c *Client) SignOIDC(ctx context.Context, agent *coretypes.Agent, idToken string, options SignOpts) (*coretypes.Agent, error) {
	// Validate request.
	if agent == nil {
		return nil, errors.New("agent must be set")
	}

	// Load signing options.
	var signOpts sign.BundleOptions
	{
		// Define config to use for signing.
		signingConfig, err := root.NewSigningConfig(
			root.SigningConfigMediaType02,
			// Fulcio URLs
			[]root.Service{
				{
					URL:                 options.FulcioURL,
					MajorAPIVersion:     1,
					ValidityPeriodStart: time.Now().Add(-time.Hour),
					ValidityPeriodEnd:   time.Now().Add(time.Hour),
				},
			},
			// OIDC Provider URLs
			// Usage and requirements: https://docs.sigstore.dev/certificate_authority/oidc-in-fulcio/
			[]root.Service{
				{
					URL:                 options.OIDCProviderURL,
					MajorAPIVersion:     1,
					ValidityPeriodStart: time.Now().Add(-time.Hour),
					ValidityPeriodEnd:   time.Now().Add(time.Hour),
				},
			},
			// Rekor URLs
			[]root.Service{
				{
					URL:                 options.RekorURL,
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
					URL:                 options.TimestampURL,
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
			Timeout: 30 * time.Second,
			Retries: 1,
		}
		signOpts.CertificateProvider = sign.NewFulcio(fulcioOpts)
		signOpts.CertificateProviderOptions = &sign.CertificateProviderOptions{
			IDToken: idToken,
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
				Timeout: 30 * time.Second,
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
				Timeout: 90 * time.Second,
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

	return c.Sign(ctx, agent, signKeypair, signOpts)
}

func (c *Client) SignWithKey(ctx context.Context, privKey []byte, agent *coretypes.Agent) (*coretypes.Agent, error) {
	if len(privKey) == 0 {
		return nil, errors.New("key must not be empty")
	}

	// Generate an ephemeral keypair using the provided key as a hint.
	// This allows the hint to be used later for verification.
	signKeypair, err := sign.NewEphemeralKeypair(&sign.EphemeralKeypairOptions{
		Hint: privKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create ephemeral keypair: %w", err)
	}

	return c.Sign(ctx, agent, signKeypair, sign.BundleOptions{})
}

func (c *Client) Sign(_ context.Context, agent *coretypes.Agent, signKeypair *sign.EphemeralKeypair, signOpts sign.BundleOptions) (*coretypes.Agent, error) {
	// Reset the signature field in the agent.
	// This is required as the agent may have been signed before,
	// but also because this ensures signing idempotency.
	agent.Signature = nil

	// Convert the agent to JSON.
	agentJSON, err := json.Marshal(agent)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal agent: %w", err)
	}

	// Sign the agent JSON data.
	sigBundle, err := sign.Bundle(&sign.PlainData{Data: agentJSON}, signKeypair, signOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to sign agent: %w", err)
	}
	certData := sigBundle.GetVerificationMaterial()
	sigData := sigBundle.GetMessageSignature()

	// Extract data from the signature bundle.
	sigBundleJSON, err := protojson.Marshal(sigBundle)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal bundle: %w", err)
	}

	// Update the agent with the signature details.
	agent.Signature = &coretypes.Signature{
		Algorithm:     sigData.GetMessageDigest().GetAlgorithm().String(),
		Signature:     base64.StdEncoding.EncodeToString(sigData.GetSignature()),
		Certificate:   base64.StdEncoding.EncodeToString(certData.GetCertificate().GetRawBytes()),
		ContentType:   sigBundle.GetMediaType(),
		ContentBundle: base64.StdEncoding.EncodeToString(sigBundleJSON),
		SignedAt:      time.Now().Format(time.RFC3339),
	}

	return agent, nil
}
