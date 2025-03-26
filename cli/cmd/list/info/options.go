// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:revive,wsl
package info

var opts = &options{}

type options struct {
	PeerID  string
	Network bool
}

func init() {
	flags := Command.Flags()
	flags.StringVar(&opts.PeerID, "peer", "", "Get publication summary for a single peer")
	flags.BoolVar(&opts.Network, "network", false, "Get publication summary for the network")

	Command.MarkFlagsMutuallyExclusive("peer", "network")
}
