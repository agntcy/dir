// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package options

import (
	commonOptions "github.com/agntcy/dir/cli/cmd/options"
	"github.com/spf13/cobra"
)

type PushOptions struct {
	*commonOptions.BaseOption
	FromStdIn bool
}

func NewPushOptions(base *commonOptions.BaseOption, cmd *cobra.Command) *PushOptions {
	opts := &PushOptions{
		BaseOption: base,
	}

	opts.AddRegisterFns(func() error {
		flags := cmd.Flags()
		flags.BoolVar(&opts.FromStdIn, "stdin", false,
			"Read compiled data from standard input. Useful for piping. Reads from file if empty. "+
				"Ignored if file is provided as an argument.",
		)

		return nil
	})

	return opts
}
