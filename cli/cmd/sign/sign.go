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

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/cli/presenter"
	agentUtils "github.com/agntcy/dir/cli/util/agent"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
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
	// Get the client from the context
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	// Load into an Agent struct
	agent := &coretypes.Agent{}
	if _, err := agent.LoadFromReader(source); err != nil {
		return fmt.Errorf("failed to load agent: %w", err)
	}

	var err error

	var agentSigned *coretypes.Agent

	//nolint:nestif
	if opts.Key != "" {
		// Load the key from file
		rawKey, err := os.ReadFile(filepath.Clean(opts.Key))
		if err != nil {
			return fmt.Errorf("failed to read key file: %w", err)
		}

		// Sign the agent using the provided key
		agentSigned, err = c.SignWithKey(cmd.Context(), rawKey, agent)
		if err != nil {
			return fmt.Errorf("failed to sign agent with key: %w", err)
		}
	} else {
		// Retrieve the token from the OIDC provider
		token, err := oauthflow.OIDConnect(opts.OIDCProviderURL, opts.OIDCClientID, "", "", oauthflow.DefaultIDTokenGetter)
		if err != nil {
			return fmt.Errorf("failed to get OIDC token: %w", err)
		}

		// Sign the agent using the OIDC provider
		agentSigned, err = c.SignOIDC(cmd.Context(), agent, token.RawString, opts.SignOpts)
		if err != nil {
			return fmt.Errorf("failed to sign agent: %w", err)
		}
	}

	// Print signed agent
	signedAgentJSON, err := json.MarshalIndent(agentSigned, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal agent: %w", err)
	}

	presenter.Print(cmd, string(signedAgentJSON))

	return nil
}
