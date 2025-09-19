// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:revive
package verify

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/agntcy/dir/utils/cosign"
	corev1alpha1 "github.com/agntcy/dirhub/backport/api/core/v1alpha1"
	"github.com/sigstore/sigstore-go/pkg/bundle"
	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/tuf"
	"github.com/sigstore/sigstore-go/pkg/util"
	"github.com/sigstore/sigstore-go/pkg/verify"
	"github.com/spf13/cobra"
	"github.com/theupdateframework/go-tuf/v2/metadata/fetcher"
)

var Command = &cobra.Command{
	Use:   "verify",
	Short: "Verify agent model signature against identity-based OIDC signing",
	Long: `This command verifies the agent data model signature against
identity-based OIDC signing process. 
It relies on Sigstore Rekor for signature verification.

Usage examples:

1. Verify an agent model from file:

	dirctl verify agent.json

2. Verify an agent model from standard input:

	dirctl pull <digest> | dirctl verify --stdin

`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var path string
		if len(args) > 1 {
			return errors.New("only one file path is allowed")
		} else if len(args) == 1 {
			path = args[0]
		}

		// get source
		source, err := GetCmdReader(path, opts.FromStdin)
		if err != nil {
			return err //nolint:wrapcheck
		}

		return runCommand(cmd, source)
	},
}

// nolint:mnd
func runCommand(cmd *cobra.Command, source io.ReadCloser) error {
	// Load into an Agent struct
	agent := &corev1alpha1.Agent{}
	if _, err := agent.LoadFromReader(source); err != nil {
		return fmt.Errorf("failed to load agent: %w", err)
	}

	//nolint:nestif
	if opts.Key != "" {
		// Load the public key from file
		rawPubKey, err := os.ReadFile(filepath.Clean(opts.Key))
		if err != nil {
			return fmt.Errorf("failed to read key file: %w", err)
		}

		// Verify the agent using the provided key
		err = VerifyWithKey(cmd.Context(), rawPubKey, agent)
		if err != nil {
			return fmt.Errorf("failed to verify agent: %w", err)
		}
	} else {
		// Verify the agent using the OIDC provider
		err := VerifyOIDC(cmd.Context(), opts.OIDCIssuer, opts.OIDCIdentity, agent)
		if err != nil {
			return fmt.Errorf("failed to verify agent: %w", err)
		}
	}

	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Signature verification successful")

	return nil
}

// Verify verifies the signature of the agent using OIDC.
func VerifyOIDC(_ context.Context, expectedIssuer, expectedSigner string, agent *corev1alpha1.Agent) error {
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

func VerifyWithKey(_ context.Context, key []byte, agent *corev1alpha1.Agent) error {
	// Validate request.
	if len(key) == 0 {
		return errors.New("key must not be empty")
	}

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

	// Get the public key from the signature bundle and compare it with the provided key.
	sigBundleVerificationMaterial := sigBundle.VerificationMaterial
	if sigBundleVerificationMaterial == nil {
		return errors.New("signature bundle has no verification material")
	}

	pubKey := sigBundleVerificationMaterial.GetPublicKey()
	if pubKey == nil {
		return errors.New("signature bundle verification material has no public key")
	}

	// Decode the PEM-encoded public key and generate the expected hint.
	p, _ := pem.Decode(key)
	if p == nil {
		return errors.New("failed to decode PEM block containing public key")
	}

	if p.Type != "PUBLIC KEY" {
		return fmt.Errorf("unexpected PEM type: %s", p.Type)
	}

	expectedHint := string(cosign.GenerateHintFromPublicKey(p.Bytes))

	if pubKey.GetHint() != expectedHint {
		return fmt.Errorf("public key hint mismatch: expected %s, got %s", expectedHint, pubKey.GetHint())
	}

	return nil
}

func GetCmdReader(fpath string, fromStdin bool) (io.ReadCloser, error) {
	if fpath == "" && !fromStdin {
		return nil, errors.New("if no path defined --stdin flag must be set")
	}

	if fpath != "" {
		file, err := os.Open(fpath)
		if err != nil {
			return nil, fmt.Errorf("could not open file %s: %w", fpath, err)
		}

		return file, nil
	}

	return os.Stdin, nil
}
