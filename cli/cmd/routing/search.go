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
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search for remote records from other peers",
	Long: `Search for remote records from other peers using the routing API.

This command discovers records that have been published by other peers in the network.
It uses cached network announcements and filters out local records.

Key Features:
- Remote-only: Only returns records from other peers
- OR logic: Records returned if they match ≥ minScore queries
- Match scoring: Shows how well records match your criteria
- Peer information: Shows which peer provides each record

Usage examples:

1. Search for AI-related records:
   dirctl routing search --skill "AI"

2. Search with multiple criteria (AND logic):
   dirctl routing search --skill "AI" --skill "ML" --min-score 2

3. Search with result limiting:
   dirctl routing search --skill "web-development" --limit 5

4. Output formats:
   # Get results as JSON
   dirctl routing search --skill "AI" --output json
   
   # Search and pipe to sync
   dirctl routing search --skill "AI" --output json | dirctl sync create --stdin
   
   # Get raw results for scripting
   dirctl routing search --skill "web" --output raw

`,
	//nolint:gocritic // Lambda required due to signature mismatch - runSearchCommand doesn't use args
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runSearchCommand(cmd)
	},
}

// Search command options.
var searchOpts struct {
	Skills   []string
	Locators []string
	Domains  []string
	Modules  []string
	Limit    uint32
	MinScore uint32
}

const (
	defaultSearchLimit = 10
	// defaultMinScore matches the server-side DefaultMinMatchScore constant for consistency.
	defaultMinScore = 1
)

func init() {
	// Add flags for search options
	searchCmd.Flags().StringArrayVar(&searchOpts.Skills, "skill", nil, "Search for records with specific skill (can be repeated)")
	searchCmd.Flags().StringArrayVar(&searchOpts.Locators, "locator", nil, "Search for records with specific locator type (can be repeated)")
	searchCmd.Flags().StringArrayVar(&searchOpts.Domains, "domain", nil, "Search for records with specific domain (can be repeated)")
	searchCmd.Flags().StringArrayVar(&searchOpts.Modules, "module", nil, "Search for records with specific module (can be repeated)")
	searchCmd.Flags().Uint32Var(&searchOpts.Limit, "limit", defaultSearchLimit, "Maximum number of results to return")
	searchCmd.Flags().Uint32Var(&searchOpts.MinScore, "min-score", defaultMinScore, "Minimum match score (number of queries that must match)")

	// Add examples in flag help
	searchCmd.Flags().Lookup("skill").Usage = "Search for records with specific skill (e.g., --skill 'AI' --skill 'ML')"
	searchCmd.Flags().Lookup("locator").Usage = "Search for records with specific locator type (e.g., --locator 'docker-image')"
	searchCmd.Flags().Lookup("domain").Usage = "Search for records with specific domain (e.g., --domain 'research' --domain 'analytics')"
	searchCmd.Flags().Lookup("module").Usage = "Search for records with specific module (e.g., --module 'runtime/language' --module 'runtime/framework')"

	// Add standard output format flags
	presenter.AddOutputFlags(searchCmd)
}

func runSearchCommand(cmd *cobra.Command) error {
	// Get the client from the context
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	// Build queries from flags
	queries := make([]*routingv1.RecordQuery, 0, len(searchOpts.Skills)+len(searchOpts.Locators)+len(searchOpts.Domains)+len(searchOpts.Modules))

	// Add skill queries
	for _, skill := range searchOpts.Skills {
		queries = append(queries, &routingv1.RecordQuery{
			Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL,
			Value: skill,
		})
	}

	// Add locator queries
	for _, locator := range searchOpts.Locators {
		queries = append(queries, &routingv1.RecordQuery{
			Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_LOCATOR,
			Value: locator,
		})
	}

	// Add domain queries
	for _, domain := range searchOpts.Domains {
		queries = append(queries, &routingv1.RecordQuery{
			Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_DOMAIN,
			Value: domain,
		})
	}

	// Add module queries
	for _, module := range searchOpts.Modules {
		queries = append(queries, &routingv1.RecordQuery{
			Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_MODULE,
			Value: module,
		})
	}

	// Validate that we have at least some criteria
	if len(queries) == 0 {
		presenter.PrintSmartf(cmd, "No search criteria specified. Use --skill, --locator, --domain, or --module flags.\n")
		presenter.PrintSmartf(cmd, "Examples:\n")
		presenter.PrintSmartf(cmd, "  dirctl routing search --skill 'AI' --locator 'docker-image'\n")
		presenter.PrintSmartf(cmd, "  dirctl routing search --domain 'research' --module 'runtime/language'\n")

		return nil
	}

	// Build search request
	req := &routingv1.SearchRequest{
		Queries: queries,
	}

	// Add optional parameters
	if searchOpts.Limit > 0 {
		req.Limit = &searchOpts.Limit
	}

	if searchOpts.MinScore > 0 {
		req.MinMatchScore = &searchOpts.MinScore
	}

	// Execute search
	resultCh, err := c.SearchRouting(cmd.Context(), req)
	if err != nil {
		return fmt.Errorf("failed to search routing: %w", err)
	}

	// Collect results
	results := make([]interface{}, 0, searchOpts.Limit)
	for result := range resultCh {
		results = append(results, result)
	}

	return presenter.PrintMessage(cmd, "remote records", "Remote records found", results)
}
