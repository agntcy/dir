// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:dupword
package apikey

import (
	"github.com/agntcy/dir/hub/cmd/apikey/create"
	"github.com/agntcy/dir/hub/cmd/apikey/delete"
	"github.com/agntcy/dir/hub/cmd/apikey/list"
	"github.com/agntcy/dir/hub/cmd/options"
	"github.com/spf13/cobra"
)

// NewCommand creates all apikey commands for the Agent Hub CLI.
// Returns the configured *cobra.Command.
func NewCommand(hubOpts *options.HubOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apikey",
		Short: "Manage the Agent Hub API keys",

		TraverseChildren: true,
	}

	cmd.AddCommand(
		create.NewCommand(hubOpts),
		delete.NewCommand(hubOpts),
		list.NewCommand(hubOpts),
	)

	return cmd
}
