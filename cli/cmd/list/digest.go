// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package list

import (
	"fmt"
	"strings"

	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/agntcy/dir/cli/presenter"
	"github.com/agntcy/dir/client"
	"github.com/spf13/cobra"
)

const (
	UnknownCID = "unknown"
)

func listCid(cmd *cobra.Command, client *client.Client, cid string) error {
	// CID-specific listing should use Search API to find providers across network
	presenter.Printf(cmd, "Note: CID lookup should use Search API for network-wide provider discovery.\n")
	presenter.Printf(cmd, "Checking if CID exists in local records:\n\n")

	// For now, check if we have this record locally
	items, err := client.List(cmd.Context(), &routingv1.ListRequest{
		// No queries = list all local records, then filter
	})
	if err != nil {
		return fmt.Errorf("failed to list local records: %w", err)
	}

	found := false
	// Check if the requested CID exists in our local records
	for item := range items {
		if item.GetRecordRef().GetCid() == cid {
			found = true

			presenter.Printf(cmd,
				"Local Record Found:\n  CID: %s\n  Labels: %s\n",
				item.GetRecordRef().GetCid(),
				strings.Join(item.GetLabels(), ", "),
			)

			break
		}
	}

	if !found {
		presenter.Printf(cmd, "CID %s not found in local records.\n", cid)
		presenter.Printf(cmd, "Use 'dirctl search' to find providers across the network.\n")
	}

	return nil
}
