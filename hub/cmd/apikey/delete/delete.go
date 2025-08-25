// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package delete

import (
	"errors"
	"fmt"

	"github.com/agntcy/dir/hub/auth"
	hubClient "github.com/agntcy/dir/hub/client/hub"
	"github.com/agntcy/dir/hub/cmd/apikey/options"
	hubOptions "github.com/agntcy/dir/hub/cmd/options"
	service "github.com/agntcy/dir/hub/service"
	"github.com/agntcy/dir/hub/sessionstore"
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
	opts := options.NewApiKeyDeleteOptions(hubOpts)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runCommand(cmd, args, opts)
	}

	return cmd
}

func runCommand(cmd *cobra.Command, args []string, opts *options.ApiKeyDeleteOptions) error {
	if len(args) != 1 {
		return errors.New("API Key Client ID is the only required argument")
	}

	// Retreive the API key Client ID to delete from the command arguments
	clientId := args[0]
	fmt.Printf("Deleting API key with Client ID: %s\n", clientId)

	// Retrieve session from context
	ctxSession := cmd.Context().Value(sessionstore.SessionContextKey)
	currentSession, ok := ctxSession.(*sessionstore.HubSession)
	if !ok || currentSession == nil {
		return errors.New("could not get current hub session")
	}
	if !auth.HasLoginCreds(currentSession) && auth.HasApiKey(currentSession) {
		fmt.Println("User is authenticated with API key, using it to get credentials...")
		if err := auth.RefreshApiKeyAccessToken(cmd.Context(), currentSession, opts.ServerAddress); err != nil {
			return fmt.Errorf("failed to refresh API key access token: %w", err)
		}
	}

	if !auth.HasLoginCreds(currentSession) && !auth.HasApiKey(currentSession) {
		return errors.New("you need to be logged in to push to the hub\nuse `dirctl hub login` command to login")
	}

	hc, err := hubClient.New(currentSession.HubBackendAddress)
	if err != nil {
		return fmt.Errorf("failed to create hub client: %w", err)
	}

	_, err = service.DeleteAPIKey(cmd.Context(), hc, clientId, currentSession)
	if err != nil {
		return fmt.Errorf("failed to delete API key: %w", err)
	}
	fmt.Printf("API Key '%s' deleted successfully\n", clientId)

	return nil
}
