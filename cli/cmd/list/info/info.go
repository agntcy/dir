// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wsl
package info

import (
	"errors"
	"fmt"

	routingv1 "github.com/agntcy/dir/api/routing/v1"
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

	// Info command now shows local record statistics only (List is local-only)
	if opts.PeerID != "" {
		presenter.Printf(cmd, "Warning: --peer flag ignored. Info operation is local-only.\n")
		presenter.Printf(cmd, "Use 'dirctl search' for network-wide statistics.\n\n")
	}

	presenter.Printf(cmd, "Local Record Summary:\n\n")

	// Get all local records
	items, err := c.List(cmd.Context(), &routingv1.ListRequest{
		// No queries = list all local records
	})
	if err != nil {
		return fmt.Errorf("failed to list local records: %w", err)
	}

	// Count labels manually since we don't have label_counts in the new API
	labelCounts := make(map[string]uint64)
	totalRecords := 0

	for item := range items {
		totalRecords++

		// Count each label
		for _, label := range item.GetLabels() {
			labelCounts[label]++
		}
	}

	// Print summary statistics
	presenter.Printf(cmd, "Total Records: %d\n\n", totalRecords)

	if len(labelCounts) > 0 {
		presenter.Printf(cmd, "Label Distribution:\n")
		for label, count := range labelCounts {
			presenter.Printf(cmd, "  %s: %d\n", label, count)
		}
	} else {
		presenter.Printf(cmd, "No labels found.\n")
	}

	return nil
}
