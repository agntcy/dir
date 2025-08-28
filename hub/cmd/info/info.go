// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package info provides the CLI commands for getting information about users and organizations.
// The info command has subcommands for getting user and organization information.
package info

import (
	"github.com/agntcy/dir/hub/cmd/info/org"
	"github.com/agntcy/dir/hub/cmd/info/user"
	"github.com/agntcy/dir/hub/cmd/options"
	"github.com/spf13/cobra"
)

// NewCommand creates the "info" command for the Agent Hub CLI.
// It provides subcommands for getting information about users and organizations.
// Returns the configured *cobra.Command.
func NewCommand(hubOpts *options.HubOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info",
		Short: "Get information about users and organizations",
		Long:  "Get information about users and organizations including details and metadata",
	}

	cmd.AddCommand(org.NewCommand(hubOpts))
	cmd.AddCommand(user.NewCommand(hubOpts))

	return cmd
}
