// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package pull provides the CLI command for pulling agents from the Agent Hub.
package pull

import (
	"errors"
	"fmt"
	"os"

	hubClient "github.com/agntcy/dir/hub/client/hub"
	hubOptions "github.com/agntcy/dir/hub/cmd/options"
	service "github.com/agntcy/dir/hub/service"
	authUtils "github.com/agntcy/dir/hub/utils/auth"
	"github.com/spf13/cobra"
)

// NewCommand creates the "pull" command for the Agent Hub CLI.
// It pulls a record from the hub by digest or repository:version and prints the result.
// Returns the configured *cobra.Command.
func NewCommand(hubOpts *hubOptions.HubOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pull <agent_ref>",
		Short: "Pull a record from Agent Hub",
		Long: `Pull a record from the Agent Hub.

Parameters:
  <agent_ref>    Agent reference in one of the following formats:
                - sha256:<hash>    : Pull by digest
                - <repo>:<version> : Pull by repository and version

Authentication:
  API key authentication can be provided via:
  1. API key file: --apikey-file (JSON file with API key credentials)
  2. Environment variables: DIRCTL_CLIENT_ID and DIRCTL_CLIENT_SECRET
  3. Session file created via 'dirctl hub login'

  API key file takes precedence over environment variables, which take precedence over session file.

Examples:
  # Pull agent by digest
  dirctl hub pull sha256:1234567890abcdef...

  # Pull agent by repository name and version
  dirctl hub pull repo-name:v1.0.0

  # Pull using API key file (JSON format)
  # File content example:
  # {
  #   "client_id": "your-client-id",
  #   "secret": "your-secret"
  # }
  dirctl hub pull repo-name:v1.0.0 --apikey-file /path/to/apikey.json

  # Pull using API key authentication via environment variables
  export DIRCTL_CLIENT_ID=your_client_id
  export DIRCTL_CLIENT_SECRET=your_secret
  dirctl hub pull repo-name:v1.0.0

  # Pull using session file (after login)
  dirctl hub login
  dirctl hub pull repo-name:v1.0.0`,
	}

	opts := hubOptions.NewHubPullOptions(hubOpts)

	// API key authentication flags
	var apikeyFile string

	cmd.Flags().StringVar(&apikeyFile, "apikey-file", "", `Path to a JSON file containing API key credentials (format: {"client_id": "...", "secret": "..."})`)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("agent id is the only required argument")
		}

		cmd.SetOut(os.Stdout)
		cmd.SetErr(os.Stderr)

		// Authenticate using either API key file or session file
		currentSession, err := authUtils.GetOrCreateSession(cmd, opts.ServerAddress, "", "", apikeyFile, false)
		if err != nil {
			return fmt.Errorf("failed to get or create session: %w", err)
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

		fmt.Fprintf(cmd.OutOrStdout(), "%s\n", string(prettyModel))

		return nil
	}

	return cmd
}
