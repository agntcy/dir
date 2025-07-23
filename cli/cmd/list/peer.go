// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package list

import (
	"fmt"
	"strings"

	routetypes "github.com/agntcy/dir/api/routing/v1alpha2"
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
		LegacyListRequest: &routetypes.LegacyListRequest{
			Peer:   peer,
			Labels: labels,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to list peer records: %w", err)
	}

	// Print the results
	for item := range items {
		var cid string
		if ref := item.GetRef(); ref != nil {
			cid = ref.GetCid()
		} else {
			cid = "unknown"
		}

		presenter.Printf(cmd,
			"Peer %s\n  CID: %s\n  Labels: %s\n",
			item.GetPeer().GetId(),
			cid,
			strings.Join(item.GetLabels(), ", "),
		)
	}

	return nil
}
