// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sync

import "github.com/agntcy/dir/cli/presenter"

var opts = &options{}

type options struct {
	Limit  uint32
	Offset uint32
	CIDs   []string
	Stdin  bool
}

//nolint:mnd
func init() {
	// Add flags for list command
	listFlags := listCmd.Flags()
	listFlags.Uint32Var(&opts.Limit, "limit", 100, "Maximum number of sync operations to return (default: 100)")
	listFlags.Uint32Var(&opts.Offset, "offset", 0, "Number of sync operations to skip (for pagination)")

	// Add flags for create command
	createFlags := createCmd.Flags()
	createFlags.StringSliceVar(&opts.CIDs, "cids", []string{}, "List of CIDs to synchronize from the remote Directory. If empty, all objects will be synchronized.")
	createFlags.BoolVar(&opts.Stdin, "stdin", false, "Parse routing search output from stdin to create sync operations for each provider")

	// Add output format flags to all sync subcommands
	presenter.AddOutputFlags(createCmd)
	presenter.AddOutputFlags(listCmd)
	presenter.AddOutputFlags(statusCmd)
	presenter.AddOutputFlags(deleteCmd)
}
