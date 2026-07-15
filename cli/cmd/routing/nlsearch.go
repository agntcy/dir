// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package routing

import (
	"errors"
	"fmt"

	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/agntcy/dir/cli/internal/extractor"
	"github.com/agntcy/dir/cli/internal/nlsearch"
	sdk "github.com/agntcy/oasf-sdk/pkg/extractor"
	"github.com/spf13/cobra"
)

// runNLRoutingSearch decomposes a free-text query into skill and domain signals
// using the OASF extractor, then searches the routing layer for matching remote
// records. Keyword signals are dropped because the DHT only indexes skills,
// domains, locators, and modules — there is no name/description equivalent.
func runNLRoutingSearch(cmd *cobra.Command, query string) error {
	// routingMinScore is lower than the default (0.3) because the DHT indexes
	// exact taxonomy names: a weaker semantic match is still a valid search key,
	// and recall matters more here than precision.
	const routingMinScore = 0.1

	// Use semantic-only weights and a low SDK-level floor so the extractor
	// returns enough candidates for our routingMinScore filter to work with.
	ext, err := extractor.LoadConfigured(
		sdk.WithWeights(1, 0),
		sdk.WithDomainWeights(1, 0),
		sdk.WithDefaultMinScore(routingMinScore),
	)
	if err != nil {
		return fmt.Errorf("natural-language search requires the OASF extractor — run `dirctl init` to set it up: %w", err)
	}

	var queryOpts []sdk.QueryOption
	if len(searchOpts.SchemaVersions) > 0 {
		queryOpts = append(queryOpts, sdk.Versions(searchOpts.SchemaVersions...))
	}

	signals, err := nlsearch.DecomposeWithMinScore(cmd.Context(), query, ext, routingMinScore, queryOpts...)
	if err != nil {
		return fmt.Errorf("decompose query: %w", err)
	}

	// Only skill and domain signals have routing DHT equivalents.
	var queries []*routingv1.RecordQuery

	for _, sig := range signals {
		switch sig.Type {
		case nlsearch.SignalTypeSkillName:
			queries = append(queries, &routingv1.RecordQuery{
				Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL,
				Value: sig.Value,
			})
		case nlsearch.SignalTypeDomainName:
			queries = append(queries, &routingv1.RecordQuery{
				Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_DOMAIN,
				Value: sig.Value,
			})
		case nlsearch.SignalTypeKeyword:
			// Keywords have no DHT equivalent — the routing layer only indexes
			// skills, domains, locators, and modules.
		}
	}

	if len(queries) == 0 {
		return errors.New("no skill or domain signals extracted from query; the routing layer only indexes skills and domains — try a more descriptive phrase")
	}

	if searchOpts.Verbose {
		cmd.PrintErrf("[nl-routing-search] signals extracted (%d usable of %d total):\n", len(queries), len(signals))

		for _, q := range queries {
			cmd.PrintErrf("  %-8s  %s\n", q.GetType(), q.GetValue())
		}
	}

	return runRoutingSearch(cmd, queries)
}
