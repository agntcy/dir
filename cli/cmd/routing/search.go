// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"errors"
	"fmt"
	"strings"

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
	Features []string
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
	searchCmd.Flags().StringArrayVar(&searchOpts.Features, "feature", nil, "Search for records with specific feature (can be repeated)")
	searchCmd.Flags().Uint32Var(&searchOpts.Limit, "limit", defaultSearchLimit, "Maximum number of results to return")
	searchCmd.Flags().Uint32Var(&searchOpts.MinScore, "min-score", defaultMinScore, "Minimum match score (number of queries that must match)")

	// Add examples in flag help
	searchCmd.Flags().Lookup("skill").Usage = "Search for records with specific skill (e.g., --skill 'AI' --skill 'ML')"
	searchCmd.Flags().Lookup("locator").Usage = "Search for records with specific locator type (e.g., --locator 'docker-image')"
	searchCmd.Flags().Lookup("domain").Usage = "Search for records with specific domain (e.g., --domain 'research' --domain 'analytics')"
	searchCmd.Flags().Lookup("feature").Usage = "Search for records with specific feature (e.g., --feature 'runtime/language' --feature 'runtime/framework')"
}

func runSearchCommand(cmd *cobra.Command) error {
	// Get the client from the context
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	// Build queries from flags
	queries := make([]*routingv1.RecordQuery, 0, len(searchOpts.Skills)+len(searchOpts.Locators)+len(searchOpts.Domains)+len(searchOpts.Features))

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

	// Add feature queries
	for _, feature := range searchOpts.Features {
		queries = append(queries, &routingv1.RecordQuery{
			Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_FEATURE,
			Value: feature,
		})
	}

	// Validate that we have at least some criteria
	if len(queries) == 0 {
		presenter.Printf(cmd, "No search criteria specified. Use --skill, --locator, --domain, or --feature flags.\n")
		presenter.Printf(cmd, "Examples:\n")
		presenter.Printf(cmd, "  dirctl routing search --skill 'AI' --locator 'docker-image'\n")
		presenter.Printf(cmd, "  dirctl routing search --domain 'research' --feature 'runtime/language'\n")

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

	// Process and display results
	resultCount := 0
	for result := range resultCh {
		resultCount++

		// Display result information
		presenter.Printf(cmd, "Record: %s\n", result.GetRecordRef().GetCid())
		presenter.Printf(cmd, "  Provider: %s\n", result.GetPeer().GetId())
		presenter.Printf(cmd, "  Match Score: %d/%d\n", result.GetMatchScore(), len(queries))

		// Display matching queries
		if len(result.GetMatchQueries()) > 0 {
			presenter.Printf(cmd, "  Matching Queries:\n")

			for _, query := range result.GetMatchQueries() {
				queryType := strings.TrimPrefix(query.GetType().String(), "RECORD_QUERY_TYPE_")
				presenter.Printf(cmd, "    - %s: %s\n", queryType, query.GetValue())
			}
		}

		presenter.Printf(cmd, "\n")
	}

	if resultCount == 0 {
		presenter.Printf(cmd, "No remote records found matching your criteria.\n")
		presenter.Printf(cmd, "Try:\n")
		presenter.Printf(cmd, "  - Broader search terms\n")
		presenter.Printf(cmd, "  - Lower --min-score value\n")
		presenter.Printf(cmd, "  - Check if other peers have published matching records\n")
	} else {
		presenter.Printf(cmd, "Found %d remote record(s)\n", resultCount)
	}

	return nil
}
