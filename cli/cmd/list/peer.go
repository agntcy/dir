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

// convertLabelsToRecordQueries converts legacy label format to RecordQuery format.
func convertLabelsToRecordQueries(labels []string) []*routingv1.RecordQuery {
	var queries []*routingv1.RecordQuery

	for _, label := range labels {
		switch {
		case strings.HasPrefix(label, "/skills/"):
			skillName := strings.TrimPrefix(label, "/skills/")
			queries = append(queries, &routingv1.RecordQuery{
				Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL,
				Value: skillName,
			})
		case strings.HasPrefix(label, "/locators/"):
			locatorType := strings.TrimPrefix(label, "/locators/")
			queries = append(queries, &routingv1.RecordQuery{
				Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_LOCATOR,
				Value: locatorType,
			})
		}
	}

	return queries
}

func listPeer(cmd *cobra.Command, client *client.Client, peerID string, labels []string) error {
	// Note: peerID is ignored since List is now local-only
	// For querying specific peers, use Search API instead
	if peerID != "" {
		presenter.Printf(cmd, "Warning: --peer flag ignored. List operation is local-only.\n")
		presenter.Printf(cmd, "Use 'dirctl search' for network-wide queries.\n\n")
	}

	// Convert legacy labels to RecordQuery format
	queries := convertLabelsToRecordQueries(labels)

	// Start the list request (local-only)
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
