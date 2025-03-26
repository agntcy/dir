// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:revive,wsl
package list

var opts = &options{}

type options struct {
	Digest  string
	PeerID  string
	Network bool
}

func init() {
	flags := Command.Flags()
	flags.StringVar(&opts.Digest, "digest", "", "Get published records for a given object")
	flags.StringVar(&opts.PeerID, "peer", "", "Get published records for a single peer")
	flags.BoolVar(&opts.Network, "network", false, "Get published records for the network")

	Command.MarkFlagsMutuallyExclusive("digest", "peer", "network")
}
