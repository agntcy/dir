// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package options

import (
	commonOptions "github.com/agntcy/dir/cli/cmd/options"
	"github.com/spf13/cobra"
)

type PublishOptions struct {
	*commonOptions.BaseOption

	Network bool
}

func NewPublishOptions(baseOption *commonOptions.BaseOption, cmd *cobra.Command) *PublishOptions {
	opts := &PublishOptions{
		BaseOption: baseOption,
	}

	opts.AddRegisterFns(func() error {
		flags := cmd.Flags()
		flags.BoolVar(&opts.Network, "network", false, "Publish data to the network")

		return nil
	})

	return opts
}
