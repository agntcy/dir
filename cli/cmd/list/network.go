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

func listNetwork(cmd *cobra.Command, client *client.Client, labels []string) error {
	// Start the list request
	networkList := true

	items, err := client.List(cmd.Context(), &routetypes.ListRequest{
		Labels:  labels,
		Network: &networkList,
	})
	if err != nil {
		return fmt.Errorf("failed to list network records: %w", err)
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
