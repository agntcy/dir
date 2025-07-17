// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package list

import (
	"fmt"
	"strings"

	routetypes "github.com/agntcy/dir/api/routing/v1alpha1"
	"github.com/agntcy/dir/cli/presenter"
	"github.com/agntcy/dir/client"
	"github.com/spf13/cobra"
)

func listPeer(cmd *cobra.Command, client *client.Client, peerID string, labels []string) error {
	// Is peer set
	// if not, run local list only
	var peer *routetypes.Peer
	if peerID != "" {
		peer = &routetypes.Peer{
			Id: peerID,
		}
	}

	// Start the list request
	items, err := client.List(cmd.Context(), &routetypes.ListRequest{
		Peer:    peer,
		Labels:  labels,
	})
	if err != nil {
		return fmt.Errorf("failed to list peer records: %w", err)
	}

	// Print the results
	for item := range items {
		presenter.Printf(cmd,
			"Peer %s\n  Digest: %s\n  Labels: %s\n",
			item.GetPeer().GetId(),
			item.GetRecord().GetDigest(),
			strings.Join(item.GetLabels(), ", "),
		)
	}

	return nil
}
