// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package options

import (
	"fmt"

	"github.com/spf13/cobra"
)

type ListOptions struct {
	*BaseOption

	Digest  string
	PeerID  string
	Network bool
}

func NewListOptions(baseOption *BaseOption, cmd *cobra.Command) *ListOptions {
	opts := &ListOptions{
		BaseOption: baseOption,
	}

	opts.AddRegisterFns(func() error {
		flags := cmd.Flags()
		flags.StringVar(&opts.Digest, "digest", "", "Get published records for a given object")
		flags.StringVar(&opts.PeerID, "peer", "", "Get published records for a single peer")
		flags.BoolVar(&opts.Network, "network", false, "Get published records for the network")

		if err := flags.MarkHidden("peer"); err != nil {
			return fmt.Errorf("unable to mark flag 'peer' as hidden: %w", err)
		}

		cmd.MarkFlagsMutuallyExclusive("digest", "peer", "network")

		return nil
	})

	return opts
}
