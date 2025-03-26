// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wsl
package info

import (
	"errors"
	"fmt"

	routetypes "github.com/agntcy/dir/api/routing/v1alpha1"
	"github.com/agntcy/dir/cli/presenter"
	"github.com/agntcy/dir/cli/util"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "info",
	Short: "Get summary details about published data",
	Long: `Usage example:

	# List summary about our published data.
   	dir list info
	
	# List summary about published data by a specific peer.
   	dir list info --peer <peer-id>
	
	# List summary about published data by the whole network.
	# NOTE: This starts a DHT walk, so it may take a while.
	# NOTE: Results are not guaranteed to be complete and up-to-date.
   	dir list info --network
	
`,
	RunE: func(cmd *cobra.Command, _ []string) error { //nolint:gocritic
		return runCommand(cmd)
	},
}

func runCommand(cmd *cobra.Command) error {
	// Get the client from the context.
	c, ok := util.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	// Is peer set
	var peer *routetypes.Peer
	if opts.PeerID != "" {
		peer = &routetypes.Peer{
			Id: opts.PeerID,
		}
	}

	// Set max hops when searching the network
	maxHops := uint32(10) //nolint:mnd
	networkList := opts.Network || peer != nil

	// Start the list request
	items, err := c.List(cmd.Context(), &routetypes.ListRequest{
		Peer:    peer,
		MaxHops: &maxHops,
		Network: &networkList,
	})
	if err != nil {
		return fmt.Errorf("failed to list peers: %w", err)
	}

	// Print the results
	for item := range items {
		peerName := item.GetPeer().GetId()

		// in case we have nothing for that host, skip
		if len(item.GetLabelCounts()) == 0 {
			// presenter.Printf(cmd, "Peer %s | <empty>\n", peerName)

			continue
		}

		// otherwise, print each label and count
		for label, count := range item.GetLabelCounts() {
			presenter.Printf(cmd, "Peer %s | Label: %s | Total: %d\n", peerName, label, count)
		}
	}

	return nil
}
