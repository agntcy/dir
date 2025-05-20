// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package verify

import (
	"errors"
	"fmt"
	"io"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/cli/presenter"
	agentUtils "github.com/agntcy/dir/cli/util/agent"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/spf13/cobra"
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

	// Verify the agent using the OIDC provider
	err := c.VerifyOIDC(cmd.Context(), opts.OIDCIssuer, opts.OIDCIdentity, agent)
	if err != nil {
		return fmt.Errorf("failed to verify agent: %w", err)
	}

	// Print success message
	presenter.Print(cmd, "Verification OK!")

	return nil
}
