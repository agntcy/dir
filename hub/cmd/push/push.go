// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package push provides the CLI command for pushing models to the Agent Hub.
package push

import (
	"errors"
	"fmt"
	"os"

	"github.com/agntcy/dir/cli/util/agent"
	hubClient "github.com/agntcy/dir/hub/client/hub"
	hubOptions "github.com/agntcy/dir/hub/cmd/options"
	"github.com/agntcy/dir/hub/service"
	authUtils "github.com/agntcy/dir/hub/utils/auth"
	"github.com/spf13/cobra"
)

// NewCommand creates the "push" command for the Agent Hub CLI.
// It pushes a model to the hub by repository name or ID, reading the model from a file or stdin.
// Returns the configured *cobra.Command.
func NewCommand(hubOpts *hubOptions.HubOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push <repository> {<model.json> | --stdin} ",
		Short: "Push model to Agent Hub",
		Long: `Push a model to the Agent Hub.

Parameters:
  <repository>    Repository name in the format of '<owner>/<name>'
  <model.json>    Path to the model file (optional)
  --stdin         Read model from standard input (optional)

Authentication:
  API key authentication can be provided via:
  1. Command flags: --client-id and --secret
  2. Environment variables: DIRCTL_CLIENT_ID and DIRCTL_CLIENT_SECRET
  3. Session file created via 'dirctl hub login'

  Command flags take precedence over environment variables, which take precedence over session file.

Examples:
  # Push model to a repository by name
  dirctl hub push owner/repo-name model.json

  # Push model to a repository by ID
  dirctl hub push 123e4567-e89b-12d3-a456-426614174000 model.json

  # Push model from stdin
  dirctl hub push owner/repo-name --stdin < model.json

  # Push using API key authentication via flags
  dirctl hub push owner/repo-name model.json --client-id API_KEY_CLIENT_ID --secret API_KEY_SECRET

  # Push using API key authentication via environment variables
  export DIRCTL_CLIENT_ID=your_client_id
  export DIRCTL_CLIENT_SECRET=your_secret
  dirctl hub push owner/repo-name model.json

  # Push using session file (after login)
  dirctl hub login
  dirctl hub push owner/repo-name model.json`,
	}

	opts := hubOptions.NewHubPushOptions(hubOpts, cmd)

	// API key authentication flags
	var clientID, secret string

	cmd.Flags().StringVar(&clientID, "client-id", "", "API key client ID for authentication")
	cmd.Flags().StringVar(&secret, "secret", "", "API key secret for authentication")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
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

		if len(args) > 2 { //nolint:mnd
			return errors.New("the following arguments could be given: <repository>:<version> [model.json]")
		}

		fpath := ""
		if len(args) == 2 { //nolint:mnd
			fpath = args[1]
		}

		reader, err := agent.GetReader(fpath, opts.FromStdIn)
		if err != nil {
			return fmt.Errorf("failed to get reader: %w", err)
		}

		agentBytes, err := agent.GetAgentBytes(reader)
		if err != nil {
			return fmt.Errorf("failed to get agent bytes: %w", err)
		}

		// TODO: Push based on repoName and version misleading
		repository := service.ParseRepoTagID(args[0])

		resp, err := service.PushAgent(cmd.Context(), hc, agentBytes, repository, currentSession)
		if err != nil {
			return fmt.Errorf("failed to push agent: %w", err)
		}

		fmt.Fprintln(cmd.OutOrStdout(), resp.GetId().GetDigest())

		return nil
	}

	return cmd
}
