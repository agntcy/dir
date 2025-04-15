// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package publish

var opts = &options{}

type options struct {
	Network bool
}

func init() {
	flags := Command.Flags()
	flags.BoolVar(&opts.Network, "network", false, "Publish data to the network")
}
