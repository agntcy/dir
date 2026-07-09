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
	Sort    string // "relevance", "popularity", or "recency"
	Verbose bool
	Filters Filters
}

// registerFlags adds search flags to the command.
func registerFlags(cmd *cobra.Command) {
	flags := cmd.Flags()

	flags.StringVar(&opts.Format, "format", "cid", "Output format: cid (default) or record")
	flags.Uint32Var(&opts.Limit, "limit", 100, "Maximum number of results to return (default: 100)") //nolint:mnd
	flags.Uint32Var(&opts.Offset, "offset", 0, "Pagination offset (default: 0)")
	flags.StringVar(&opts.Sort, "sort", "", "Sort mode for structured queries: relevance, popularity, or recency (default: recency)")
	flags.BoolVar(&opts.Verbose, "verbose", false, "Print NL signal decomposition and per-signal hit counts to stderr (natural-language search only)")

	RegisterFilterFlags(cmd, &opts.Filters)
}

// buildQueriesFromFlags builds API queries.
func buildQueriesFromFlags() []*searchv1.RecordQuery {
	return BuildQueries(&opts.Filters)
}

// sortMode converts the --sort flag value to the proto SortMode enum.
func sortMode() searchv1.SortMode {
	switch opts.Sort {
	case "relevance":
		return searchv1.SortMode_SORT_MODE_RELEVANCE
	case "popularity":
		return searchv1.SortMode_SORT_MODE_POPULARITY
	case "recency":
		return searchv1.SortMode_SORT_MODE_RECENCY
	default:
		return searchv1.SortMode_SORT_MODE_UNSPECIFIED
	}
}
