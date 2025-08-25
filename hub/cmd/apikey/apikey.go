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

// NewCommand creates the "pull" command for the Agent Hub CLI.
// It pulls an agent from the hub by digest or repository:version and prints the result.
// Returns the configured *cobra.Command.
func NewCommand(hubOpts *options.HubOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apikey",
		Short: "Manage the Agent Hub",

		TraverseChildren: true,
	}

	cmd.AddCommand(
		create.NewCommand(hubOpts),
		delete.NewCommand(hubOpts),
		list.NewCommand(hubOpts),
	)

	return cmd
}
