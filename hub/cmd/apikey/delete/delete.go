// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package delete //nolint:predeclared

import (
	"errors"
	"fmt"

	hubClient "github.com/agntcy/dir/hub/client/hub"
	"github.com/agntcy/dir/hub/cmd/apikey/options"
	hubOptions "github.com/agntcy/dir/hub/cmd/options"
	service "github.com/agntcy/dir/hub/service"
	"github.com/agntcy/dir/hub/sessionstore"
	authUtils "github.com/agntcy/dir/hub/utils/auth"
	"github.com/spf13/cobra"
)

// NewCommand creates the "delete" command for the Agent apikey Hub CLI.
func NewCommand(hubOpts *hubOptions.HubOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <client_id>",
		Short: "delete an API key for hub action",
		Long: `delete an API key for hub action.

Parameters:
  <client_id>    Client ID of the API key to delete

Examples:
  # Delete API key
  dirctl hub apikey delete f1a3c901-deba-40bc-8a7c-9f9b0a246b80`,
	}
	opts := options.NewAPIKeyDeleteOptions(hubOpts, cmd)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runCommand(cmd, args, opts)
	}

	return cmd
}

func runCommand(cmd *cobra.Command, args []string, opts *options.APIKeyDeleteOptions) error {
	if len(args) != 1 {
		return errors.New("API Key Client ID is the only required argument")
	}

	// Retrieve the API key Client ID to delete from the command arguments
	clientID := args[0]
	if !opts.JSONOutput {
		fmt.Fprintf(cmd.OutOrStdout(), "Deleting API key with Client ID: %s\n", clientID)
	}

	// Retrieve session from context
	ctxSession := cmd.Context().Value(sessionstore.SessionContextKey)

	currentSession, ok := ctxSession.(*sessionstore.HubSession)
	if !ok || currentSession == nil {
		return errors.New("could not get current hub session")
	}

	// Check for credentials
	if err := authUtils.CheckForCreds(cmd, currentSession, opts.ServerAddress, opts.JSONOutput); err != nil {
		// this error need to be return without modification in order to be displayed
		return err //nolint:wrapcheck
	}

	hc, err := hubClient.New(currentSession.HubBackendAddress)
	if err != nil {
		return fmt.Errorf("failed to create hub client: %w", err)
	}

	_, err = service.DeleteAPIKey(cmd.Context(), hc, clientID, currentSession)
	if err != nil {
		return fmt.Errorf("failed to delete API key: %w", err)
	}

	if !opts.JSONOutput {
		fmt.Fprintf(cmd.OutOrStdout(), "API Key '%s' deleted successfully\n", clientID)
	}

	return nil
}
