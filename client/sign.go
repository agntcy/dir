package client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	v1 "github.com/sigstore/protobuf-specs/gen/pb-go/trustroot/v1"
	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/sign"
	"google.golang.org/protobuf/encoding/protojson"
)

// SignRequest represents the request to sign an agent.
type SignRequest struct {
	Agent       *coretypes.Agent
	OIDCIDToken string // OIDC flow against Sigstore Fulcio performed/passed by the caller
}

func (r *SignRequest) Validate() error {
	if r.Agent == nil {
		return fmt.Errorf("agent is required")
	}

	return nil
}

// Sign takes in the signature request details and returns the signed agent.
func (c *Client) Sign(ctx context.Context, req *SignRequest) (*coretypes.Agent, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid sign request: %w", err)
	}

	// Reset the signature field in the agent.
	agent := req.Agent
	agent.Signature = nil

	// Convert the agent to JSON.
	agentData, err := json.Marshal(agent)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal agent: %w", err)
	}

	// Define config to use for signing.
	signingConfig, err := root.NewSigningConfig(
		root.SigningConfigMediaType02,
		// Fulcio URLs
		[]root.Service{
			{
				URL:                 "https://fulcio.sigstage.dev",
				MajorAPIVersion:     1,
				ValidityPeriodStart: time.Now().Add(-time.Hour),
				ValidityPeriodEnd:   time.Now().Add(time.Hour),
			},
		},
		// OIDC Provider URLs
		[]root.Service{
			{
				URL:                 "https://oauth2.sigstage.dev/auth",
				MajorAPIVersion:     1,
				ValidityPeriodStart: time.Now().Add(-time.Hour),
				ValidityPeriodEnd:   time.Now().Add(time.Hour),
			},
		},
		// Rekor URLs
		[]root.Service{
			{
				URL:                 "https://rekor.sigstage.dev",
				MajorAPIVersion:     1,
				ValidityPeriodStart: time.Now().Add(-time.Hour),
				ValidityPeriodEnd:   time.Now().Add(time.Hour),
			},
		},
		root.ServiceConfiguration{
			Selector: v1.ServiceSelector_ANY,
		},
		[]root.Service{},
		root.ServiceConfiguration{
			Selector: v1.ServiceSelector_ANY,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create signing config: %w", err)
	}

	// Define signing options from the config.
	opts := sign.BundleOptions{}

	// Use fulcio to sign the agent.
	fulcioURL, err := root.SelectService(signingConfig.FulcioCertificateAuthorityURLs(), []uint32{1}, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to select fulcio URL: %w", err)
	}
	fulcioOpts := &sign.FulcioOptions{
		BaseURL: fulcioURL,
		Timeout: time.Duration(30 * time.Second),
		Retries: 1,
	}
	opts.CertificateProvider = sign.NewFulcio(fulcioOpts)
	opts.CertificateProviderOptions = &sign.CertificateProviderOptions{
		IDToken: req.OIDCIDToken,
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
			Timeout: time.Duration(90 * time.Second),
			Retries: 1,
		}
		opts.TransparencyLogs = append(opts.TransparencyLogs, sign.NewRekor(rekorOpts))
	}

	// Generate a keypair for signing.
	keypair, err := sign.NewEphemeralKeypair(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create ephemeral keypair: %w", err)
	}

	// Sign the data.
	bundle, err := sign.Bundle(&sign.PlainData{Data: agentData}, keypair, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to sign agent: %w", err)
	}

	// Extract data from the bundle.
	bundleJSON, err := protojson.Marshal(bundle)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal bundle: %w", err)
	}

	// Update the agent with the signature.
	agent.Signature = &coretypes.Signature{
		Signature: string(bundleJSON),
	}

	return agent, nil
}
