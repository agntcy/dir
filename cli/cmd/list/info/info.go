// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wsl
package info

import (
	"errors"
	"fmt"

	routetypes "github.com/agntcy/dir/api/routing/v1alpha1"
	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "info",
	Short: "Get summary details about published data",
	Long: `Get aggregated summary about the data held in your local
data store or across the network.	

Usage examples:

1. List summary about locally published data:

	dir list info
	
2. List summary about published data across the network:

   	dir list info --network
	
`,
	RunE: func(cmd *cobra.Command, _ []string) error { //nolint:gocritic
		return runCommand(cmd)
	},
}

func runCommand(cmd *cobra.Command) error {
	// Get the client from the context.
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
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

	// Start the list request
	items, err := c.List(cmd.Context(), &routetypes.ListRequest{
		Peer:    peer,
		MaxHops: &maxHops,
	})
	if err != nil {
		return fmt.Errorf("failed to list peers: %w", err)
	}

	// Print the results
	for item := range items {
		peerName := item.GetPeer().GetId()

		// in case we have nothing for that host, skip
		if len(item.GetLabelCounts()) == 0 {
			continue
		}

		// otherwise, print each label and count
		for label, count := range item.GetLabelCounts() {
			presenter.Printf(cmd, "Peer %s | Label: %s | Total: %d\n", peerName, label, count)
		}
	}

	return nil
}
