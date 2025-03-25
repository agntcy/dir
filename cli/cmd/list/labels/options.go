// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:revive,wsl
package labels

var opts = &options{}

type options struct {
	PeerId string //nolint:stylecheck
}

func init() {
	flags := Command.Flags()
	flags.StringVar(&opts.PeerId, "peer", "", "Peer ID to search for labels")

	// Command.MarkFlagRequired("peer") //nolint:errcheck
}
