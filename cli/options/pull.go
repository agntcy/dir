// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package options

import "github.com/spf13/cobra"

type PullOptions struct {
	*BaseOption

	FormatRaw bool
}

func NewPullOptions(baseOption *BaseOption, cmd *cobra.Command) *PullOptions {
	opts := &PullOptions{
		BaseOption: baseOption,
	}

	opts.AddRegisterFns(func() error {
		flags := cmd.Flags()
		flags.BoolVar(&opts.FormatRaw, "raw", false, "Output in Raw format. Defaults to JSON.")

		return nil
	})

	return opts
}
