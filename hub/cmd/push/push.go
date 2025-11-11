// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package push provides the CLI command for pushing records to the Agent Hub.
package push

import (
	"errors"
	"fmt"
	"io"
	"os"

	hubClient "github.com/agntcy/dir/hub/client/hub"
	hubOptions "github.com/agntcy/dir/hub/cmd/options"
	"github.com/agntcy/dir/hub/service"
	authUtils "github.com/agntcy/dir/hub/utils/auth"
	"github.com/spf13/cobra"
)

// NewCommand creates the "push" command for the Agent Hub CLI.
// It pushes a record to the hub by organization name or ID, reading the record from a file or stdin.
// Returns the configured *cobra.Command.
func NewCommand(hubOpts *hubOptions.HubOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push <organization> {<record.json> | --stdin} ",
		Short: "Push record to Agent Hub",
		Long: `Push a record to the Agent Hub.

Parameters:
  <organization>  Organization name or organization ID
  <record.json>   Path to the record file (optional)
  --stdin         Read record from standard input (optional)

Authentication:
  API key authentication can be provided via:
  1. API key file: --apikey-file (JSON file with API key credentials)
  2. Environment variables: DIRCTL_CLIENT_ID and DIRCTL_CLIENT_SECRET
  3. Session file created via 'dirctl hub login'

  API key file takes precedence over environment variables, which take precedence over session file.

Examples:
  # Push record to an organization by name
  dirctl hub push your-org record.json

  # Push record to an organization by ID
  dirctl hub push 123e4567-e89b-12d3-a456-426614174000 record.json

  # Push record from stdin
  dirctl hub push your-org --stdin < record.json

  # Push using API key file (JSON format)
  # File content example:
  # {
  #   "client_id": "your-client-id",
  #   "secret": "your-secret"
  # }
  dirctl hub push your-org record.json --apikey-file /path/to/apikey.json

  # Push using API key authentication via environment variables
  export DIRCTL_CLIENT_ID=your_client_id
  export DIRCTL_CLIENT_SECRET=your_secret
  dirctl hub push your-org record.json

  # Push using session file (after login)
  dirctl hub login
  dirctl hub push your-org record.json`,
	}

	opts := hubOptions.NewHubPushOptions(hubOpts, cmd)

	// API key authentication flags
	var apikeyFile string

	cmd.Flags().StringVar(&apikeyFile, "apikey-file", "", `Path to a JSON file containing API key credentials (format: {"client_id": "...", "secret": "..."})`)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		cmd.SetOut(os.Stdout)
		cmd.SetErr(os.Stderr)

		// Authenticate using either API key file or session file
		currentSession, err := authUtils.GetOrCreateSession(cmd, opts.ServerAddress, apikeyFile, false)
		if err != nil {
			return fmt.Errorf("failed to get or create session: %w", err)
		}

		hc, err := hubClient.New(currentSession.HubBackendAddress)
		if err != nil {
			return fmt.Errorf("failed to create hub client: %w", err)
		}

		if len(args) == 0 {
			return errors.New("organization argument is required")
		}

		if len(args) > 2 { //nolint:mnd
			return errors.New("the following arguments can be given: <organization> [record.json]")
		}

		organization := args[0]

		fpath := ""
		if len(args) == 2 { //nolint:mnd
			fpath = args[1]
		}

		reader, err := getReader(fpath, opts.FromStdIn)
		if err != nil {
			return fmt.Errorf("failed to get reader: %w", err)
		}

		recordBytes, err := io.ReadAll(reader)
		if err != nil {
			return fmt.Errorf("failed to read data: %w", err)
		}

		resp, err := service.PushRecord(cmd.Context(), hc, organization, recordBytes, currentSession)
		if err != nil {
			return fmt.Errorf("failed to push record: %w", err)
		}

		fmt.Fprintln(cmd.OutOrStdout(), resp.GetCid())

		return nil
	}

	return cmd
}

func getReader(fpath string, fromStdin bool) (io.ReadCloser, error) {
	if fpath == "" && !fromStdin {
		return nil, errors.New("if no path defined --stdin flag must be set")
	}

	if fpath != "" {
		file, err := os.Open(fpath)
		if err != nil {
			return nil, fmt.Errorf("could not open file %s: %w", fpath, err)
		}

		return file, nil
	}

	return io.NopCloser(os.Stdin), nil
}
