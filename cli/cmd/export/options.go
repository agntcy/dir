// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package export

import (
	"github.com/agntcy/dir/cli/cmd/search"
	"github.com/agntcy/dir/cli/presenter"
)

var opts = &options{}

type options struct {
	Format     string
	OutputFile string
	OutputDir  string
	Limit      uint32
	Filters    search.Filters
}

func init() {
	flags := Command.Flags()
	flags.StringVar(&opts.Format, "format", "oasf", "Export format: oasf, agent-skill, a2a, mcp-ghcopilot")
	flags.StringVar(&opts.OutputFile, "output-file", "", "File path to write the exported data (default: stdout)")
	flags.StringVar(&opts.OutputDir, "output-dir", "", "Directory for batch export from search results")
	flags.Uint32Var(&opts.Limit, "limit", 100, "Maximum number of records to export in batch mode") //nolint:mnd

	search.RegisterFilterFlags(Command, &opts.Filters)
	presenter.AddOutputFlags(Command)
}
