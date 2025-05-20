// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package verify

import (
	"errors"
	"fmt"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "verify",
	Short: "Verify agent model signature using OIDC-based identity",
	Long: `This command verifies the agent data model signature using OIDC. 

Usage examples:

	dirctl verify agent.json

`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("path to agent model is a required argument")
		}

		return runCommand(cmd, args[0])
	},
}

func runCommand(cmd *cobra.Command, pathToAgent string) error {
	// Get the client from the context.
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	// Load into an Agent struct
	agent := &coretypes.Agent{}
	_, err := agent.LoadFromFile(pathToAgent)
	if err != nil {
		return fmt.Errorf("failed to load agent from file: %w", err)
	}

	// Verify the agent using the OIDC provider
	err = c.VerifyOIDC(cmd.Context(), "https://github.com/login/oauth", "rpolic@cisco.com", agent)
	if err != nil {
		return fmt.Errorf("failed to verify agent: %w", err)
	}

	// Print success message
	presenter.Print(cmd, "Agent successfully verified")

	return nil
}
