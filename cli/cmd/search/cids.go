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

var cidsCmd = &cobra.Command{
	Use:   "cids",
	Short: "Search for record CIDs",
	Long: `Search for records and return only their CIDs.

This is efficient for piping to other commands like pull or delete.

Usage examples:

1. Basic search with filters:
   dirctl search cids --name "my-agent-name" --version "v1.0.0"

2. Wildcard search:
   dirctl search cids --name "web*" --skill "python*"

3. Pipe to pull command:
   dirctl search cids --name "web*" --output raw | xargs -I {} dirctl pull {}

4. Output formats:
   dirctl search cids --name "web*" --output json
   dirctl search cids --name "web*" --output jsonl
   dirctl search cids --name "web*" --output raw
`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runCidsCommand(cmd)
	},
}

func init() {
	registerFlags(cidsCmd)
}

func runCidsCommand(cmd *cobra.Command) error {
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	// Build queries from direct field flags
	queries := buildQueriesFromFlags()

	ch, err := c.SearchCIDs(cmd.Context(), &searchv1.SearchRequest{
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
			len(opts.DomainIDs)+len(opts.DomainNames)+
			len(opts.CreatedAts)+len(opts.Authors)+
			len(opts.SchemaVersions)+len(opts.ModuleIDs))

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
			Type:  searchv1.RecordQueryType_RECORD_QUERY_TYPE_MODULE_NAME,
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

	// Add created-at queries
	for _, createdAt := range opts.CreatedAts {
		queries = append(queries, &searchv1.RecordQuery{
			Type:  searchv1.RecordQueryType_RECORD_QUERY_TYPE_CREATED_AT,
			Value: createdAt,
		})
	}

	// Add author queries
	for _, author := range opts.Authors {
		queries = append(queries, &searchv1.RecordQuery{
			Type:  searchv1.RecordQueryType_RECORD_QUERY_TYPE_AUTHOR,
			Value: author,
		})
	}

	// Add schema-version queries
	for _, schemaVersion := range opts.SchemaVersions {
		queries = append(queries, &searchv1.RecordQuery{
			Type:  searchv1.RecordQueryType_RECORD_QUERY_TYPE_SCHEMA_VERSION,
			Value: schemaVersion,
		})
	}

	// Add module-id queries
	for _, moduleID := range opts.ModuleIDs {
		queries = append(queries, &searchv1.RecordQuery{
			Type:  searchv1.RecordQueryType_RECORD_QUERY_TYPE_MODULE_ID,
			Value: moduleID,
		})
	}

	return queries
}

