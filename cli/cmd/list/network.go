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

func listNetwork(cmd *cobra.Command, client *client.Client, labels []string) error {
	// Network listing should use Search API since List is local-only
	presenter.Printf(cmd, "Note: Network mode now uses Search API for network-wide discovery.\n\n")

	// Convert legacy labels to RecordQuery format
	queries := convertLabelsToRecordQueries(labels)

	// Use Search API for network-wide discovery
	// TODO: This should be implemented when Search API is available in the client
	// For now, fall back to local List with a warning
	presenter.Printf(cmd, "Warning: Search API not yet implemented in CLI.\n")
	presenter.Printf(cmd, "Falling back to local records only:\n\n")

	// Start the list request (local-only fallback)
	items, err := client.List(cmd.Context(), &routingv1.ListRequest{
		Queries: queries,
	})
	if err != nil {
		return fmt.Errorf("failed to list local records: %w", err)
	}

	// Print the results
	for item := range items {
		cid := item.GetRecordRef().GetCid()

		presenter.Printf(cmd,
			"Local Record:\n  CID: %s\n  Labels: %s\n",
			cid,
			strings.Join(item.GetLabels(), ", "),
		)

		presenter.Printf(cmd, "\n")
	}

	return nil
}
