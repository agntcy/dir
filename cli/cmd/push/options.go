// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package push

import (
	signcmd "github.com/agntcy/dir/cli/cmd/sign"
	"github.com/agntcy/dir/cli/presenter"
	"github.com/agntcy/dir/client"
)

var opts = &options{}

type options struct {
	FromStdin bool
	Sign      bool

	// Signing options
	client.SignOpts
}

func init() {
	flags := Command.Flags()
	flags.BoolVar(&opts.FromStdin, "stdin", false,
		"Read compiled data from standard input. Useful for piping. Reads from file if empty. "+
			"Ignored if file is provided as an argument.",
	)
	flags.BoolVar(&opts.Sign, "sign", false,
		"Sign the record with the specified signing options.",
	)

	signcmd.AddSigningFlags(flags)

	// Add output format flags
	presenter.AddOutputFlags(Command)
}
