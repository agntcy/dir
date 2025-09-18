// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sign

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	agentUtils "github.com/agntcy/dir/hub/utils/agent"
	"github.com/agntcy/dir/utils/cosign"
	corev1alpha1 "github.com/agntcy/dirhub/backport/api/core/v1alpha1"
	"github.com/sigstore/sigstore/pkg/oauthflow"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "sign",
	Short: "Sign agent model using identity-based OIDC signing",
	Long: `This command signs the agent data model using identity-based signing.
It uses a short-lived signing certificate issued by Sigstore Fulcio
along with a local ephemeral signing key and OIDC identity.

Verification data is attached to the signed agent model,
and the transparency log is pushed to Sigstore Rekor.

This command opens a browser window to authenticate the user 
with the default OIDC provider.

Usage examples:

1. Sign an agent model from file:

	dirctl sign agent.json

2. Sign an agent model from standard input:

	cat agent.json | dirctl sign --stdin

`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var fpath string
		if len(args) > 1 {
			return errors.New("only one file path is allowed")
		} else if len(args) == 1 {
			fpath = args[0]
		}

		// get source
		source, err := agentUtils.GetReader(fpath, opts.FromStdin)
		if err != nil {
			return err //nolint:wrapcheck
		}

		return runCommand(cmd, source)
	},
}

func runCommand(cmd *cobra.Command, source io.ReadCloser) error {
	// Load data from source
	agent := &corev1alpha1.Agent{}
	if _, err := agent.LoadFromReader(source); err != nil {
		return fmt.Errorf("failed to read data: %w", err)
	}
	defer source.Close() //nolint:errcheck

	// Get data to sign, drop the existing signature if any
	agent.Signature = nil
	agentDigest, err := agentUtils.GetDigest(agent)
	if err != nil {
		return fmt.Errorf("failed to marshal agent: %w", err)
	}
	signingData := agentDigest.String()

	//nolint:nestif,gocritic
	if opts.Key != "" {
		// Load the key from file
		rawKey, err := os.ReadFile(filepath.Clean(opts.Key))
		if err != nil {
			return fmt.Errorf("failed to read key file: %w", err)
		}

		// Read password from environment variable
		pw, err := cosign.ReadPrivateKeyPassword()()
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}

		// Sign the agent using the provided key
		signature, err := cosign.SignBlobWithKey(cmd.Context(), &cosign.SignBlobKeyOptions{
			Payload:    []byte(signingData),
			PrivateKey: rawKey,
			Password:   pw,
		})
		if err != nil {
			return fmt.Errorf("failed to sign agent with key: %w", err)
		}

		// Set agent signature
		setAgentSignature(agent, signature.Signature, signature.PublicKey)
	} else if opts.OIDCToken != "" {
		// Sign the agent using the OIDC provider
		signature, err := cosign.SignBlobWithOIDC(cmd.Context(), &cosign.SignBlobOIDCOptions{
			Payload:         []byte(signingData),
			IDToken:         opts.OIDCToken,
			FulcioURL:       opts.FulcioURL,
			RekorURL:        opts.RekorURL,
			TimestampURL:    opts.TimestampURL,
			OIDCProviderURL: opts.OIDCProviderURL,
		})
		if err != nil {
			return fmt.Errorf("failed to sign agent: %w", err)
		}

		// Set agent signature
		setAgentSignature(agent, signature.Signature, signature.PublicKey)
	} else {
		// Retrieve the token from the OIDC provider
		token, err := oauthflow.OIDConnect(opts.OIDCProviderURL, opts.OIDCClientID, "", "", oauthflow.DefaultIDTokenGetter)
		if err != nil {
			return fmt.Errorf("failed to get OIDC token: %w", err)
		}

		// Sign the agent using the OIDC provider
		signature, err := cosign.SignBlobWithOIDC(cmd.Context(), &cosign.SignBlobOIDCOptions{
			Payload:         []byte(signingData),
			IDToken:         token.RawString,
			FulcioURL:       opts.FulcioURL,
			RekorURL:        opts.RekorURL,
			TimestampURL:    opts.TimestampURL,
			OIDCProviderURL: opts.OIDCProviderURL,
		})
		if err != nil {
			return fmt.Errorf("failed to sign agent: %w", err)
		}

		// Set agent signature
		setAgentSignature(agent, signature.Signature, signature.PublicKey)
	}

	// Print signed agent
	signedAgentJSON, err := json.MarshalIndent(agent, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal agent: %w", err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", string(signedAgentJSON))

	return nil
}

func setAgentSignature(agent *corev1alpha1.Agent, signature, certificate string) {
	agent.Signature = &corev1alpha1.Signature{
		Signature:   signature,
		Certificate: certificate,
	}
}
