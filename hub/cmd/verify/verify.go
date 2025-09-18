// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package verify

import (
	"bytes"
	"crypto"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"github.com/agntcy/dir/cli/presenter"
	agentUtils "github.com/agntcy/dir/hub/utils/agent"
	"github.com/agntcy/dir/utils/cosign"
	corev1alpha1 "github.com/agntcy/dirhub/backport/api/core/v1alpha1"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "verify",
	Short: "Verify agent model signature against identity-based signing",
	Long: `This command verifies the agent data model signature against
identity-based signing process. 

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
		source, err := agentUtils.GetReader(path, opts.FromStdin)
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

	// Extract agent signature
	if agent.Signature == nil {
		return errors.New("agent does not contain a signature")
	}
	pubKey := agent.Signature.Certificate
	sig := agent.Signature.Signature

	// Verify agent
	agent.Signature = nil
	agentDigest, err := agentUtils.GetDigest(agent)
	if err != nil {
		return fmt.Errorf("failed to marshal agent: %w", err)
	}
	signingData, err := cosign.GeneratePayload(agentDigest.String())
	if err != nil {
		return fmt.Errorf("failed to generate signing payload: %w", err)
	}

	if err := verifySignature(string(signingData), []byte(sig), []byte(pubKey)); err != nil {
		return fmt.Errorf("failed to verify agent signature: %w", err)
	}

	// Print success message
	presenter.Print(cmd, "Agent signature verified successfully!")

	return nil
}

// verifySignature checks if the signature of agent can be verified
// using publicKey.
// signature is expected to be base64 encoded.
// publicKey is expected to be PEM encoded.
func verifySignature(digest string, sig, publicKey []byte) error {
	sigRaw := base64.NewDecoder(base64.StdEncoding, bytes.NewReader(sig))
	pubKeyRaw, err := cryptoutils.UnmarshalPEMToPublicKey(publicKey)
	if err != nil {
		return fmt.Errorf("unable to parse public key: %w", err)
	}
	if err := cryptoutils.ValidatePubKey(pubKeyRaw); err != nil {
		return fmt.Errorf("unable to validate public key: %w", err)
	}

	verifier, err := signature.LoadVerifier(pubKeyRaw, crypto.SHA256)
	if err != nil {
		return fmt.Errorf("unable to load verifier: %w", err)
	}

	if err := verifier.VerifySignature(sigRaw, bytes.NewReader([]byte(digest))); err != nil {
		return fmt.Errorf("unable to verify signature: %w", err)
	}

	return nil
}
