// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package options

import (
	commonOptions "github.com/agntcy/dir/cli/cmd/options"
	"github.com/spf13/cobra"
)

type UnpublishOptions struct {
	*commonOptions.BaseOption

	Network bool
}

func NewUnpublishOptions(baseOption *commonOptions.BaseOption, cmd *cobra.Command) *UnpublishOptions {
	opts := &UnpublishOptions{
		BaseOption: baseOption,
	}

	opts.AddRegisterFns(func() error {
		flags := cmd.Flags()
		flags.BoolVar(&opts.Network, "network", false, "Unpublish data from the network")

		return nil
	})

	return opts
}
