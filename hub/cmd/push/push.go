// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package push provides the CLI command for pushing models to the Agent Hub.
package push

import (
	"errors"
	"fmt"

	"github.com/agntcy/dir/cli/util/agent"
	hubClient "github.com/agntcy/dir/hub/client/hub"
	hubOptions "github.com/agntcy/dir/hub/cmd/options"
	"github.com/agntcy/dir/hub/service"
	"github.com/agntcy/dir/hub/sessionstore"
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

Examples:
  # Push model to a repository by name
  dirctl hub push owner/repo-name model.json

  # Push model to a repository by ID
  dirctl hub push 123e4567-e89b-12d3-a456-426614174000 model.json

  # Push model from stdin
  dirctl hub push owner/repo-name --stdin < model.json`,
	}

	opts := hubOptions.NewHubPushOptions(hubOpts, cmd)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		// Retrieve session from context
		ctxSession := cmd.Context().Value(sessionstore.SessionContextKey)

		currentSession, ok := ctxSession.(*sessionstore.HubSession)
		if !ok || currentSession == nil {
			return errors.New("could not get current hub session")
		}

		// Check for credentials
		if err := authUtils.CheckForCreds(cmd, currentSession, opts.ServerAddress, false); err != nil {
			// this error need to be return without modification in order to be displayed
			return err //nolint:wrapcheck
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
