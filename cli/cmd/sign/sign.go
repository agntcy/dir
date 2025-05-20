// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sign

import (
	"encoding/json"
	"errors"
	"fmt"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/agntcy/dir/client"
	"github.com/agntcy/dir/hub/config"
	"github.com/agntcy/dir/hub/sessionstore"
	"github.com/agntcy/dir/hub/utils/file"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "sign",
	Short: "Sign agent model using OIDC-based identity",
	Long: `This command signs the agent data model using identity retrieved from OIDC. 
This attaches the signature to make the model verifable across the network.

Usage examples:

	dirctl sign agent.json

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

	// Get the OIDC token for session.
	// TODO: we need an auth flow here to get IDToken from Fulcio.
	// TODO: this is already implemented in the hub via sessionstore.
	sessionStore := sessionstore.NewFileSessionStore(file.GetSessionFilePath())
	currentSession, err := sessionStore.GetHubSession(config.DefaultHubAddress)
	if err != nil {
		return fmt.Errorf("failed to get hub session: %w", err)
	}
	if currentSession == nil || currentSession.Tokens == nil {
		return errors.New("no session found, please login to hub first")
	}
	oidcSession, ok := currentSession.Tokens[currentSession.CurrentTenant]
	if currentSession == nil {
		return errors.New("no session found, please login to hub first")
	}

	// Load into an Agent struct
	agent := &coretypes.Agent{}
	_, err = agent.LoadFromFile(pathToAgent)
	if err != nil {
		return fmt.Errorf("failed to load agent from file: %w", err)
	}

	// Sign the agent
	agentSigned, err := c.Sign(cmd.Context(), &client.SignRequest{
		Agent:       agent,
		OIDCIDToken: oidcSession.IDToken,
	})
	if err != nil {
		return fmt.Errorf("failed to sign agent: %w", err)
	}

	// Print agent
	agentJSON, err := json.MarshalIndent(agentSigned, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal agent: %w", err)
	}
	presenter.Print(cmd, agentJSON)

	return nil
}
