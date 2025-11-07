// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
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

5. Output formats:
   # Get results as JSON
   dirctl routing list --skill "AI" --output json
   
   # Get results as JSONL for streaming
   dirctl routing list --output jsonl
   
   # Get raw CIDs only
   dirctl routing list --skill "AI" --output raw

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
	Modules  []string
	Limit    uint32
}

func init() {
	// Add flags for list options
	listCmd.Flags().StringVar(&listOpts.Cid, "cid", "", "List specific record by CID")
	listCmd.Flags().StringArrayVar(&listOpts.Skills, "skill", nil, "Filter by skill (can be repeated)")
	listCmd.Flags().StringArrayVar(&listOpts.Locators, "locator", nil, "Filter by locator type (can be repeated)")
	listCmd.Flags().StringArrayVar(&listOpts.Domains, "domain", nil, "Filter by domain (can be repeated)")
	listCmd.Flags().StringArrayVar(&listOpts.Modules, "module", nil, "Filter by module (can be repeated)")
	listCmd.Flags().Uint32Var(&listOpts.Limit, "limit", 0, "Maximum number of results (0 = no limit)")

	// Add examples in flag help
	listCmd.Flags().Lookup("skill").Usage = "Filter by skill (e.g., --skill 'AI' --skill 'web-development')"
	listCmd.Flags().Lookup("locator").Usage = "Filter by locator type (e.g., --locator 'docker-image')"
	listCmd.Flags().Lookup("domain").Usage = "Filter by domain (e.g., --domain 'research' --domain 'analytics')"
	listCmd.Flags().Lookup("module").Usage = "Filter by module (e.g., --module 'runtime/language' --module 'runtime/framework')"
	listCmd.Flags().Lookup("cid").Usage = "List specific record by CID"

	// Add output format flags
	presenter.AddOutputFlags(listCmd)
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
	queries := make([]*routingv1.RecordQuery, 0, len(listOpts.Skills)+len(listOpts.Locators)+len(listOpts.Domains)+len(listOpts.Modules))

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

	// Add module queries
	for _, module := range listOpts.Modules {
		queries = append(queries, &routingv1.RecordQuery{
			Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_MODULE,
			Value: module,
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

	// Collect results and convert to interface{} slice in a single loop
	results := make([]interface{}, 0, listOpts.Limit)
	for result := range resultCh {
		results = append(results, result)
	}

	return presenter.PrintMessage(cmd, "local records", "Local records found", results)
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

	// Collect results and convert to interface{} slice in a single loop
	results := make([]interface{}, 0, listOpts.Limit)

	for result := range resultCh {
		if result.GetRecordRef().GetCid() == cid {
			results = append(results, result)
		}
	}

	return presenter.PrintMessage(cmd, "local records", "Local records found", results)
}
