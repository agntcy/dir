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

Authentication:
  API key authentication can be provided via:
  1. Command flags: --client-id and --secret
  2. Environment variables: DIRCTL_CLIENT_ID and DIRCTL_CLIENT_SECRET
  3. Session file created via 'dirctl hub login'

  Command flags take precedence over environment variables, which take precedence over session file.

Examples:
  # Pull agent by digest
  dirctl hub pull sha256:1234567890abcdef...

  # Pull agent by repository and version
  dirctl hub pull owner/repo-name:v1.0.0

  # Pull using API key authentication via flags
  dirctl hub pull owner/repo-name:v1.0.0 --client-id YOUR_CLIENT_ID --secret YOUR_SECRET

  # Pull using API key authentication via environment variables
  export DIRCTL_CLIENT_ID=your_client_id
  export DIRCTL_CLIENT_SECRET=your_secret
  dirctl hub pull owner/repo-name:v1.0.0

  # Pull using session file (after login)
  dirctl hub login
  dirctl hub pull owner/repo-name:v1.0.0`,
	}

	opts := hubOptions.NewHubPullOptions(hubOpts)

	// API key authentication flags
	var clientID, secret string

	cmd.Flags().StringVar(&clientID, "client-id", "", "API key client ID for authentication")
	cmd.Flags().StringVar(&secret, "secret", "", "API key secret for authentication")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("agent id is the only required argument")
		}

		cmd.SetOut(os.Stdout)
		cmd.SetErr(os.Stderr)

		// Authenticate using either API key or session file
		currentSession, err := authUtils.GetOrCreateSession(cmd, opts.ServerAddress, clientID, secret, false)
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
