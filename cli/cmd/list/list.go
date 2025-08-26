// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package list

import (
	"errors"

	"github.com/agntcy/dir/cli/cmd/list/info"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "list",
	Short: "Search for published records locally or across the network",
	Long: `Search for published data locally or across the network.
This API supports both unicast- mode for routing to specific objects,
and multicast mode for attribute-based matching and routing.

There are two modes of operation, 
	a) local mode where the data is queried from the local data store
	b) network mode where the data is queried across the network

Usage examples:

1. List all peers that are providing a specific object:

	dirctl list --cid <cid>

2. List published records on the local node:

	dirctl list "/skills/Text Completion"

3. List published records across the whole network:

	dirctl list "/skills/Text Completion" --network

---

NOTES:

To search for specific records across the network, you must specify 
matching labels passed as arguments. The matching is performed using
exact set-membership rule.

`,
	RunE: func(cmd *cobra.Command, args []string) error { //nolint:gocritic
		return runCommand(cmd, args)
	},
}

func runCommand(cmd *cobra.Command, labels []string) error {
	// Get the client from the context.
	client, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	// if we request --cid, ignore everything else
	if opts.Cid != "" {
		return listCid(cmd, client, opts.Cid)
	}

	// validate that we have labels for all the flows below
	if len(labels) == 0 {
		return errors.New("no labels specified")
	}

	if opts.Network {
		return listNetwork(cmd, client, labels)
	}

	return listPeer(cmd, client, opts.PeerID, labels)
}

func init() {
	Command.AddCommand(info.Command)
}
