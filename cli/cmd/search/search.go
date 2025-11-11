// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package search

import (
	"errors"
	"fmt"

	searchv1 "github.com/agntcy/dir/api/search/v1"
	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "search",
	Short: "Search for records",
	Long: `Search for records in the directory using various filters and options.

This command provides a consistent interface with routing search commands.

Usage examples:

1. Basic search with specific filters and limit:

	dirctl search --limit 10 \
		--offset 0 \
		--name "my-agent-name" \
		--version "v1.0.0" \
		--skill-id "10201" \
		--skill "Text Completion" \
		--locator "docker-image:https://example.com/docker-image" \
		--module "my-custom-module-name" \
		--domain-id "604" \
		--domain "*education*" 

2. Wildcard search examples:

	# Find all web-related agents
	dirctl search --name "web*"
	
	# Find all v1.x versions
	dirctl search --version "v1.*"
	
	# Find agents with Python or JavaScript skills
	dirctl search --skill "python*" --skill "*script"
	
	# Find agents with HTTP-based locators
	dirctl search --locator "http*"
	
	# Find agents with plugin modules
	dirctl search --module "*-plugin*"
	
	# Find agents in education or healthcare domains
	dirctl search --domain "*education*" --domain "healthcare/*"

3. Question mark wildcard (? matches exactly one character):

	# Find version v1.0.x where x is any single digit
	dirctl search --version "v1.0.?"
	
	# Find agents with 3-character names ending in "api"
	dirctl search --name "???api"
	
	# Find skills with single character variations
	dirctl search --skill "Pytho?"

4. List wildcards ([] matches any character within brackets):

	# Find agents with numeric suffixes
	dirctl search --name "agent-[0-9]"
	
	# Find versions starting with v followed by any digit
	dirctl search --version "v[0-9].*"
	
	# Find skills starting with uppercase letters A-M
	dirctl search --skill "[A-M]*"
	
	# Find locators with specific protocols
	dirctl search --locator "[hf]tt[ps]*"

5. Complex wildcard patterns:

	# Find API services with v2 versions
	dirctl search --name "api-*-service" --version "v2.*"
	
	# Find machine learning agents
	dirctl search --skill "*machine*learning*"
	
	# Find agents with container locators
	dirctl search --locator "*docker*" --locator "*container*"
	
	# Combine different wildcard types
	dirctl search --name "web-[0-9]?" --version "v?.*.?"

6. Output formats:

	# Get results as JSON for programmatic use
	dirctl search --name "web*" --output json
	
	# Get results as JSONL (one per line) for streaming
	dirctl search --skill "AI" --output jsonl
	
	# Get raw CIDs only for piping to other commands
	dirctl search --name "web*" --output raw | xargs -I {} dirctl pull {}

`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runCommand(cmd)
	},
}

func runCommand(cmd *cobra.Command) error {
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	// Build queries from direct field flags
	queries := buildQueriesFromFlags()

	ch, err := c.Search(cmd.Context(), &searchv1.SearchRequest{
		Limit:   &opts.Limit,
		Offset:  &opts.Offset,
		Queries: queries,
	})
	if err != nil {
		return fmt.Errorf("failed to search: %w", err)
	}

	// Collect results and convert to interface{} slice
	results := make([]interface{}, 0, opts.Limit)

	for recordCid := range ch {
		if recordCid == "" {
			continue
		}

		results = append(results, recordCid)
	}

	return presenter.PrintMessage(cmd, "record CIDs", "Record CIDs found", results)
}

// buildQueriesFromFlags builds API queries.
func buildQueriesFromFlags() []*searchv1.RecordQuery {
	queries := make([]*searchv1.RecordQuery, 0,
		len(opts.Names)+len(opts.Versions)+len(opts.SkillIDs)+
			len(opts.SkillNames)+len(opts.Locators)+len(opts.Modules)+
			len(opts.DomainIDs)+len(opts.DomainNames))

	// Add name queries
	for _, name := range opts.Names {
		queries = append(queries, &searchv1.RecordQuery{
			Type:  searchv1.RecordQueryType_RECORD_QUERY_TYPE_NAME,
			Value: name,
		})
	}

	// Add version queries
	for _, version := range opts.Versions {
		queries = append(queries, &searchv1.RecordQuery{
			Type:  searchv1.RecordQueryType_RECORD_QUERY_TYPE_VERSION,
			Value: version,
		})
	}

	// Add skill-id queries
	for _, skillID := range opts.SkillIDs {
		queries = append(queries, &searchv1.RecordQuery{
			Type:  searchv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL_ID,
			Value: skillID,
		})
	}

	// Add skill-name queries
	for _, skillName := range opts.SkillNames {
		queries = append(queries, &searchv1.RecordQuery{
			Type:  searchv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL_NAME,
			Value: skillName,
		})
	}

	// Add locator queries
	for _, locator := range opts.Locators {
		queries = append(queries, &searchv1.RecordQuery{
			Type:  searchv1.RecordQueryType_RECORD_QUERY_TYPE_LOCATOR,
			Value: locator,
		})
	}

	// Add module queries
	for _, module := range opts.Modules {
		queries = append(queries, &searchv1.RecordQuery{
			Type:  searchv1.RecordQueryType_RECORD_QUERY_TYPE_MODULE,
			Value: module,
		})
	}

	// Add domain-id queries
	for _, domainID := range opts.DomainIDs {
		queries = append(queries, &searchv1.RecordQuery{
			Type:  searchv1.RecordQueryType_RECORD_QUERY_TYPE_DOMAIN_ID,
			Value: domainID,
		})
	}

	// Add domain-name queries
	for _, domainName := range opts.DomainNames {
		queries = append(queries, &searchv1.RecordQuery{
			Type:  searchv1.RecordQueryType_RECORD_QUERY_TYPE_DOMAIN_NAME,
			Value: domainName,
		})
	}

	return queries
}
