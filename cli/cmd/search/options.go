// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package search

import "github.com/agntcy/dir/cli/presenter"

var opts = &options{}

type options struct {
	Limit  uint32
	Offset uint32

	Query Query
}

func init() {
	flags := Command.Flags()

	flags.Uint32Var(&opts.Limit, "limit", 100, "Maximum number of results to return (default: 100)") //nolint:mnd
	flags.Uint32Var(&opts.Offset, "offset", 0, "Pagination offset (default: 0)")

	flags.VarP(&opts.Query, "query", "q", "Search query terms")

	// Add output format flags
	presenter.AddOutputFlags(Command)
}
