// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package list

import (
	"errors"

	"github.com/agntcy/dir/cli/cmd/list/info"
	"github.com/agntcy/dir/cli/cmd/list/options"
	commonOptions "github.com/agntcy/dir/cli/cmd/options"
	"github.com/agntcy/dir/cli/util/context"
	"github.com/spf13/cobra"
)

func NewCommand(option *commonOptions.BaseOption) *cobra.Command {
	cmd := &cobra.Command{
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

	dirctl list --digest <digest>

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
	}

	opts := options.NewListOptions(option, cmd)

	cmd.RunE = func(cmd *cobra.Command, args []string) error { //nolint:gocritic
		return runCommand(cmd, opts, args)
	}

	cmd.AddCommand(info.NewCommand(option))

	return cmd
}

func runCommand(cmd *cobra.Command, opts *options.ListOptions, labels []string) error {
	// Get the client from the context.
	client, ok := context.GetDirClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	// if we request --digest, ignore everything else
	if opts.Digest != "" {
		return listDigest(cmd, client, opts.Digest)
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
