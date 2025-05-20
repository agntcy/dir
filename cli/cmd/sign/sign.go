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
	"github.com/sigstore/sigstore/pkg/oauthflow"
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

	// Load into an Agent struct
	agent := &coretypes.Agent{}
	_, err := agent.LoadFromFile(pathToAgent)
	if err != nil {
		return fmt.Errorf("failed to load agent from file: %w", err)
	}

	// Retreive the token from the OIDC provider
	tok, err := oauthflow.OIDConnect(client.DefaultOIDCProviderURL, client.DefaultOIDCClientID, "", "", oauthflow.DefaultIDTokenGetter)
	if err != nil {
		return fmt.Errorf("failed to get OIDC token: %w", err)
	}

	// Sign the agent using the OIDC provider
	agentSigned, err := c.SignOIDC(cmd.Context(), agent, tok.RawString)
	if err != nil {
		return fmt.Errorf("failed to sign agent: %w", err)
	}

	// Print agent
	signedAgentRaw, err := json.MarshalIndent(agentSigned, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal agent: %w", err)
	}
	presenter.Print(cmd, string(signedAgentRaw))

	return nil
}
