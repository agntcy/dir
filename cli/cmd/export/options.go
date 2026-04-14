// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package export

import "github.com/agntcy/dir/cli/presenter"

var opts = &options{}

type options struct {
	Format     string
	OutputFile string
}

func init() {
	flags := Command.Flags()
	flags.StringVar(&opts.Format, "format", "oasf", "Export format (extensible; built-in: oasf)")
	flags.StringVar(&opts.OutputFile, "output-file", "", "File path to write the exported data (default: stdout)")

	presenter.AddOutputFlags(Command)
}
