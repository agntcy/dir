// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package unpublish

var opts = &options{}

type options struct {
	Network bool
}

func init() {
	flags := Command.Flags()
	flags.BoolVar(&opts.Network, "network", false, "Unpublish data from the network")
}
