// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"errors"
	"fmt"

	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/agntcy/dir/client"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List local records with optional filtering",
	Long: `List local records with optional filtering.

This command queries records that are stored locally on this peer only.
It does NOT query the network or other peers.

Key Features:
- Local-only: Only shows records published on this peer
- Fast: Uses local storage index, no network access
- Filtering: Supports skill and locator queries with AND logic
- Efficient: Extracts labels from storage keys, no content parsing

Usage examples:

1. List all local records:
   dirctl routing list

2. List records with specific skill:
   dirctl routing list --skill "AI"

3. List records with multiple criteria (AND logic):
   dirctl routing list --skill "AI" --locator "docker-image"

4. List specific record by CID:
   dirctl routing list --cid <cid>

Note: For network-wide discovery, use 'dirctl routing search' instead.
`,
	//nolint:gocritic // Lambda required due to signature mismatch - runListCommand doesn't use args
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runListCommand(cmd)
	},
}

// List command options.
var listOpts struct {
	Cid      string
	Skills   []string
	Locators []string
	Domains  []string
	Features []string
	Limit    uint32
}

func init() {
	// Add flags for list options
	listCmd.Flags().StringVar(&listOpts.Cid, "cid", "", "List specific record by CID")
	listCmd.Flags().StringArrayVar(&listOpts.Skills, "skill", nil, "Filter by skill (can be repeated)")
	listCmd.Flags().StringArrayVar(&listOpts.Locators, "locator", nil, "Filter by locator type (can be repeated)")
	listCmd.Flags().StringArrayVar(&listOpts.Domains, "domain", nil, "Filter by domain (can be repeated)")
	listCmd.Flags().StringArrayVar(&listOpts.Features, "feature", nil, "Filter by feature (can be repeated)")
	listCmd.Flags().Uint32Var(&listOpts.Limit, "limit", 0, "Maximum number of results (0 = no limit)")

	// Add examples in flag help
	listCmd.Flags().Lookup("skill").Usage = "Filter by skill (e.g., --skill 'AI' --skill 'web-development')"
	listCmd.Flags().Lookup("locator").Usage = "Filter by locator type (e.g., --locator 'docker-image')"
	listCmd.Flags().Lookup("domain").Usage = "Filter by domain (e.g., --domain 'research' --domain 'analytics')"
	listCmd.Flags().Lookup("feature").Usage = "Filter by feature (e.g., --feature 'runtime/language' --feature 'runtime/framework')"
	listCmd.Flags().Lookup("cid").Usage = "List specific record by CID"
}

func runListCommand(cmd *cobra.Command) error {
	// Get the client from the context
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	// Handle CID-specific listing
	if listOpts.Cid != "" {
		return listByCID(cmd, c, listOpts.Cid)
	}

	// Build queries from flags
	queries := make([]*routingv1.RecordQuery, 0, len(listOpts.Skills)+len(listOpts.Locators)+len(listOpts.Domains)+len(listOpts.Features))

	// Add skill queries
	for _, skill := range listOpts.Skills {
		queries = append(queries, &routingv1.RecordQuery{
			Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL,
			Value: skill,
		})
	}

	// Add locator queries
	for _, locator := range listOpts.Locators {
		queries = append(queries, &routingv1.RecordQuery{
			Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_LOCATOR,
			Value: locator,
		})
	}

	// Add domain queries
	for _, domain := range listOpts.Domains {
		queries = append(queries, &routingv1.RecordQuery{
			Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_DOMAIN,
			Value: domain,
		})
	}

	// Add feature queries
	for _, feature := range listOpts.Features {
		queries = append(queries, &routingv1.RecordQuery{
			Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_FEATURE,
			Value: feature,
		})
	}

	// Build list request
	req := &routingv1.ListRequest{
		Queries: queries,
	}

	// Add optional limit
	if listOpts.Limit > 0 {
		req.Limit = &listOpts.Limit
	}

	// Execute list
	resultCh, err := c.List(cmd.Context(), req)
	if err != nil {
		return fmt.Errorf("failed to list: %w", err)
	}

	// Process and display results
	resultCount := 0
	for result := range resultCh {
		resultCount++

		// Display result information
		presenter.Printf(cmd, "Local Record: %s\n", result.GetRecordRef().GetCid())

		// Display labels
		if len(result.GetLabels()) > 0 {
			presenter.Printf(cmd, "  Labels:\n")

			for _, label := range result.GetLabels() {
				presenter.Printf(cmd, "    - %s\n", label)
			}
		}

		presenter.Printf(cmd, "\n")
	}

	if resultCount == 0 {
		if len(queries) == 0 {
			presenter.Printf(cmd, "No local records found.\n")
			presenter.Printf(cmd, "Use 'dirctl push' and 'dirctl publish' to add records.\n")
		} else {
			presenter.Printf(cmd, "No local records found matching your criteria.\n")
			presenter.Printf(cmd, "Try:\n")
			presenter.Printf(cmd, "  - Broader search terms\n")
			presenter.Printf(cmd, "  - 'dirctl routing search' to find records from other peers\n")
		}
	} else {
		presenter.Printf(cmd, "Found %d local record(s)\n", resultCount)
	}

	return nil
}

// listByCID lists a specific record by CID.
func listByCID(cmd *cobra.Command, c *client.Client, cid string) error {
	// For CID-specific queries, we can use an empty query list
	req := &routingv1.ListRequest{
		Queries: []*routingv1.RecordQuery{}, // Empty = list all, then we filter by CID match
	}

	resultCh, err := c.List(cmd.Context(), req)
	if err != nil {
		return fmt.Errorf("failed to list: %w", err)
	}

	// Look for the specific CID
	found := false

	for result := range resultCh {
		if result.GetRecordRef().GetCid() == cid {
			found = true

			presenter.Printf(cmd, "Local Record: %s\n", result.GetRecordRef().GetCid())

			if len(result.GetLabels()) > 0 {
				presenter.Printf(cmd, "  Labels:\n")

				for _, label := range result.GetLabels() {
					presenter.Printf(cmd, "    - %s\n", label)
				}
			}

			break // Found the record, no need to continue
		}
	}

	if !found {
		presenter.Printf(cmd, "Note: CID lookup should use Search API for network-wide provider discovery.\n\n")
		presenter.Printf(cmd, "Checking if CID exists in local records:\n")
		presenter.Printf(cmd, "CID %s not found in local records.\n", cid)
		presenter.Printf(cmd, "Use 'dirctl routing search' to find providers across the network.\n")
	}

	return nil
}
