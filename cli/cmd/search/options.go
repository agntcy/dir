// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package search

import (
	searchv1 "github.com/agntcy/dir/api/search/v1"
	"github.com/spf13/cobra"
)

var opts = &options{}

type options struct {
	Limit   uint32
	Offset  uint32
	Format  string
	Filters Filters
}

// registerFlags adds search flags to the command.
func registerFlags(cmd *cobra.Command) {
	flags := cmd.Flags()

	flags.StringVar(&opts.Format, "format", "cid", "Output format: cid (default) or record")
	flags.Uint32Var(&opts.Limit, "limit", 100, "Maximum number of results to return (default: 100)") //nolint:mnd
	flags.Uint32Var(&opts.Offset, "offset", 0, "Pagination offset (default: 0)")

	RegisterFilterFlags(cmd, &opts.Filters)
}

// buildQueriesFromFlags builds API queries.
func buildQueriesFromFlags() []*searchv1.RecordQuery {
	return BuildQueries(&opts.Filters)
}
