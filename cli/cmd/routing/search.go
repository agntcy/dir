// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package routing

import (
	"errors"
	"fmt"
	"sort"

	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search for remote records from other peers",
	Long: `Search for remote records from other peers using the routing API.

Provide a free-text query as a positional argument to use natural-language
search: the OASF extractor (set up by 'dirctl init') decomposes the phrase
into skill and domain signals. Each signal is queried independently against
the routing DHT. The discovered records (up to --limit) are then reordered
by match score, best-first. This is not a global top-N: only the window of
records returned by the DHT within the limit is reordered.

Omit the positional argument to use structured search with explicit flags
(--skill, --domain, --locator, --module).

Key Features:
- Remote-only: Only returns records from other peers
- OR logic: Records returned if they match ≥ minScore queries
- Match scoring: Discovered window reordered by match score (highest first); not a global top-N
- Peer information: Shows which peer provides each record

Usage examples:

1. Natural-language search (requires 'dirctl init'):
   dirctl routing search "Github MCP server that manages issues"
   dirctl routing search "real-time fraud detection for banking"

2. Structured search for AI-related records:
   dirctl routing search --skill "AI"

3. Search with multiple criteria:
   dirctl routing search --skill "AI" --skill "ML" --min-score 2

4. Search with result limiting:
   dirctl routing search --skill "web-development" --limit 5

5. Output formats:
   # Get results as JSON
   dirctl routing search --skill "AI" --output json

   # Search and pipe to sync
   dirctl routing search --skill "AI" --output json | dirctl sync create --stdin

   # Get raw results for scripting
   dirctl routing search --skill "web" --output raw

`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 1 {
			return runNLRoutingSearch(cmd, args[0])
		}

		return runStructuredRoutingSearch(cmd)
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
	Verbose  bool
}

const (
	defaultSearchLimit = 10
	// defaultMinScore matches the server-side DefaultMinMatchScore constant for consistency.
	defaultMinScore = 1
)

func init() {
	searchCmd.Flags().StringArrayVar(&searchOpts.Skills, "skill", nil, "Search for records with specific skill (e.g., --skill 'AI' --skill 'ML')")
	searchCmd.Flags().StringArrayVar(&searchOpts.Locators, "locator", nil, "Search for records with specific locator type (e.g., --locator 'docker-image')")
	searchCmd.Flags().StringArrayVar(&searchOpts.Domains, "domain", nil, "Search for records with specific domain (e.g., --domain 'research' --domain 'analytics')")
	searchCmd.Flags().StringArrayVar(&searchOpts.Modules, "module", nil, "Search for records with specific module (e.g., --module 'core/llm/model')")
	searchCmd.Flags().Uint32Var(&searchOpts.Limit, "limit", defaultSearchLimit, "Maximum number of results to return")
	searchCmd.Flags().Uint32Var(&searchOpts.MinScore, "min-score", defaultMinScore, "Minimum match score (number of queries that must match)")
	searchCmd.Flags().BoolVar(&searchOpts.Verbose, "verbose", false, "Print extracted signals and per-signal details to stderr")

	presenter.AddOutputFlags(searchCmd)
}

// runStructuredRoutingSearch handles the flag-driven path (--skill, --domain, etc.).
func runStructuredRoutingSearch(cmd *cobra.Command) error {
	queries := make([]*routingv1.RecordQuery, 0, len(searchOpts.Skills)+len(searchOpts.Locators)+len(searchOpts.Domains)+len(searchOpts.Modules))

	for _, skill := range searchOpts.Skills {
		queries = append(queries, &routingv1.RecordQuery{
			Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL,
			Value: skill,
		})
	}

	for _, locator := range searchOpts.Locators {
		queries = append(queries, &routingv1.RecordQuery{
			Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_LOCATOR,
			Value: locator,
		})
	}

	for _, domain := range searchOpts.Domains {
		queries = append(queries, &routingv1.RecordQuery{
			Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_DOMAIN,
			Value: domain,
		})
	}

	for _, module := range searchOpts.Modules {
		queries = append(queries, &routingv1.RecordQuery{
			Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_MODULE,
			Value: module,
		})
	}

	if len(queries) == 0 {
		presenter.PrintSmartf(cmd, "No search criteria specified. Use --skill, --locator, --domain, or --module flags.\n")
		presenter.PrintSmartf(cmd, "Examples:\n")
		presenter.PrintSmartf(cmd, "  dirctl routing search --skill 'AI' --locator 'docker-image'\n")
		presenter.PrintSmartf(cmd, "  dirctl routing search --domain 'research' --module 'core/llm/model'\n")

		return nil
	}

	return runRoutingSearch(cmd, queries)
}

// runRoutingSearch executes a routing search with the given queries, collects
// all responses, sorts them by match_score descending, and prints the result.
// Called by both the structured and NL paths.
func runRoutingSearch(cmd *cobra.Command, queries []*routingv1.RecordQuery) error {
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	req := &routingv1.SearchRequest{
		Queries: queries,
	}

	if searchOpts.Limit > 0 {
		req.Limit = &searchOpts.Limit
	}

	if searchOpts.MinScore > 0 {
		req.MinMatchScore = &searchOpts.MinScore
	}

	resultCh, err := c.SearchRouting(cmd.Context(), req)
	if err != nil {
		return fmt.Errorf("failed to search routing: %w", err)
	}

	var responses []*routingv1.SearchResponse

	for result := range resultCh {
		responses = append(responses, result)
	}

	// Reorder the discovered window by match_score descending; not a global top-N.
	sort.SliceStable(responses, func(i, j int) bool {
		return responses[i].GetMatchScore() > responses[j].GetMatchScore()
	})

	results := make([]any, len(responses))
	for i, r := range responses {
		results[i] = r
	}

	return presenter.PrintMessage(cmd, "remote records", "Remote records found", results)
}
