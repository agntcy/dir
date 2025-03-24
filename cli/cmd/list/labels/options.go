// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package labels

var opts = &options{}

type options struct {
	PeerId string
}

func init() {
	flags := Command.Flags()
	flags.StringVar(&opts.PeerId, "peer-id", "", "Peer ID to search for labels")

	Command.MarkFlagRequired("peer-id")
}
