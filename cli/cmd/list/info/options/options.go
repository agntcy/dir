// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package options

import (
	"fmt"

	commonOptions "github.com/agntcy/dir/cli/cmd/options"
	"github.com/spf13/cobra"
)

type ListInfoOptions struct {
	*commonOptions.BaseOption

	PeerID  string
	Network bool
}

func NewListInfoOptions(baseOption *commonOptions.BaseOption, cmd *cobra.Command) *ListInfoOptions {
	opts := &ListInfoOptions{
		BaseOption: baseOption,
	}

	opts.AddRegisterFns(func() error {
		flags := cmd.Flags()
		flags.StringVar(&opts.PeerID, "peer", "", "Get publication summary for a single peer")
		flags.BoolVar(&opts.Network, "network", false, "Get publication summary for the network")

		if err := flags.MarkHidden("peer"); err != nil {
			return fmt.Errorf("unable to mark flag 'peer' as hidden: %w", err)
		}

		cmd.MarkFlagsMutuallyExclusive("peer", "network")

		return nil
	})

	return opts
}
