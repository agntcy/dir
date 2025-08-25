// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package pull provides the CLI command for pulling agents from the Agent Hub.
package pull

import (
	"errors"
	"fmt"
	"os"

	"github.com/agntcy/dir/hub/auth"
	hubClient "github.com/agntcy/dir/hub/client/hub"
	hubOptions "github.com/agntcy/dir/hub/cmd/options"
	service "github.com/agntcy/dir/hub/service"
	"github.com/agntcy/dir/hub/sessionstore"
	"github.com/spf13/cobra"
)

// NewCommand creates the "pull" command for the Agent Hub CLI.
// It pulls an agent from the hub by digest or repository:version and prints the result.
// Returns the configured *cobra.Command.
func NewCommand(hubOpts *hubOptions.HubOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pull <agent_ref>",
		Short: "Pull an agent from Agent Hub",
		Long: `Pull an agent from the Agent Hub.

Parameters:
  <agent_ref>    Agent reference in one of the following formats:
                - sha256:<hash>    : Pull by digest
                - <repo>:<version> : Pull by repository and version

Examples:
  # Pull agent by digest
  dirctl hub pull sha256:1234567890abcdef...

  # Pull agent by repository and version
  dirctl hub pull owner/repo-name:v1.0.0`,
	}

	opts := hubOptions.NewHubPullOptions(hubOpts)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("agent id is the only required argument")
		}

		// Retrieve session from context
		ctxSession := cmd.Context().Value(sessionstore.SessionContextKey)
		currentSession, ok := ctxSession.(*sessionstore.HubSession)
		if !ok || currentSession == nil {
			errors.New("could not get current hub session")
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

		agentID, err := service.ParseAgentID(args[0])
		if err != nil {
			return fmt.Errorf("invalid agent id: %w", err)
		}

		prettyModel, err := service.PullAgent(cmd.Context(), hc, agentID, currentSession)
		if err != nil {
			return fmt.Errorf("failed to pull agent: %w", err)
		}
		fmt.Fprintf(os.Stdout, "%s\n", string(prettyModel))

		return nil
	}

	return cmd
}
