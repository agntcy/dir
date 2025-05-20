// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:mnd,wsl
package client

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/sigstore/sigstore-go/pkg/bundle"
	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/tuf"
	"github.com/sigstore/sigstore-go/pkg/util"
	"github.com/sigstore/sigstore-go/pkg/verify"
	"github.com/theupdateframework/go-tuf/v2/metadata/fetcher"
)

// Verify verifies the signature of the agent using OIDC.
func (c *Client) VerifyOIDC(_ context.Context, expectedIssuer, expectedSigner string, agent *coretypes.Agent) error {
	// Validate request.
	if agent == nil {
		return errors.New("agent must be set")
	}
	if agent.GetSignature() == nil {
		return errors.New("agent has no signature")
	}

	// Extract signature data from the agent.
	sigBundleRawJSON, err := base64.StdEncoding.DecodeString(agent.GetSignature().GetContentBundle())
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	sigBundle := &bundle.Bundle{}
	if err := sigBundle.UnmarshalJSON(sigBundleRawJSON); err != nil {
		return fmt.Errorf("failed to unmarshal signature bundle: %w", err)
	}

	// Get agent JSON data without the signature.
	// We need to remove the signature from the agent before verifying.
	agentSignature := agent.GetSignature()
	agent.Signature = nil

	agentJSON, err := json.Marshal(agent)
	if err != nil {
		return fmt.Errorf("failed to marshal agent: %w", err)
	}

	agent.Signature = agentSignature

	// Load identity verification options.
	var identityPolicy verify.PolicyOption
	{
		// Create OIDC identity matcher for verification.
		certID, err := verify.NewShortCertificateIdentity("", expectedIssuer, "", expectedSigner)
		if err != nil {
			return fmt.Errorf("failed to create certificate identity: %w", err)
		}

		identityPolicy = verify.WithCertificateIdentity(certID)
	}

	// Load trusted root material.
	var trustedMaterial root.TrustedMaterialCollection
	{
		// Get staging TUF trusted root.
		// TODO: allow switching between TUF environments.
		fetcher := fetcher.NewDefaultFetcher()
		fetcher.SetHTTPUserAgent(util.ConstructUserAgent())
		tufOptions := &tuf.Options{
			Root:              tuf.StagingRoot(),
			RepositoryBaseURL: tuf.StagingMirror,
			Fetcher:           fetcher,
			DisableLocalCache: true, // read-only mode; prevent from pulling root CA to local dir
		}
		tufClient, err := tuf.New(tufOptions)
		if err != nil {
			return fmt.Errorf("failed to create TUF client: %w", err)
		}

		trustedRoot, err := root.GetTrustedRoot(tufClient)
		if err != nil {
			return fmt.Errorf("failed to get trusted root: %w", err)
		}

		trustedMaterial = append(trustedMaterial, trustedRoot)
	}

	// Create verifier session.
	sev, err := verify.NewVerifier(trustedMaterial,
		verify.WithSignedCertificateTimestamps(1),
		verify.WithObserverTimestamps(1),
		verify.WithTransparencyLog(1),
	)
	if err != nil {
		return fmt.Errorf("failed to create verifier: %w", err)
	}

	// Run verification
	_, err = sev.Verify(sigBundle, verify.NewPolicy(verify.WithArtifact(bytes.NewReader(agentJSON)), identityPolicy))
	if err != nil {
		return fmt.Errorf("failed to verify signature: %w", err)
	}

	// Verify the signature.
	return nil
}
