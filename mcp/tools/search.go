// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"context"
	"fmt"

	searchv1 "github.com/agntcy/dir/api/search/v1"
	"github.com/agntcy/dir/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// SearchLocalInput defines the input parameters for local search.
type SearchLocalInput struct {
	Limit       int      `json:"limit,omitempty"        jsonschema:"Maximum number of results to return (default: 100 max: 1000)"`
	Offset      int      `json:"offset,omitempty"       jsonschema:"Pagination offset (default: 0)"`
	Names       []string `json:"names,omitempty"        jsonschema:"Agent name patterns (supports wildcards: * ? [])"`
	Versions    []string `json:"versions,omitempty"     jsonschema:"Version patterns (supports wildcards: * ? [])"`
	SkillIDs    []string `json:"skill_ids,omitempty"    jsonschema:"Skill ID patterns (exact match only)"`
	SkillNames  []string `json:"skill_names,omitempty"  jsonschema:"Skill name patterns (supports wildcards: * ? [])"`
	Locators    []string `json:"locators,omitempty"     jsonschema:"Locator patterns (supports wildcards: * ? [])"`
	Modules     []string `json:"modules,omitempty"      jsonschema:"Module patterns (supports wildcards: * ? [])"`
	DomainIDs   []string `json:"domain_ids,omitempty"   jsonschema:"Domain ID patterns (exact match only)"`
	DomainNames []string `json:"domain_names,omitempty" jsonschema:"Domain name patterns (supports wildcards: * ? [])"`
}

// SearchLocalOutput defines the output of local search.
type SearchLocalOutput struct {
	RecordCIDs   []string `json:"record_cids,omitempty"   jsonschema:"Array of matching record CIDs"`
	Count        int      `json:"count"                   jsonschema:"Number of results returned"`
	HasMore      bool     `json:"has_more"                jsonschema:"Whether more results are available beyond the limit"`
	ErrorMessage string   `json:"error_message,omitempty" jsonschema:"Error message if search failed"`
}

const (
	defaultLimit = 100
	maxLimit     = 1000
)

// SearchLocal searches for agent records on the local directory node.
func SearchLocal(ctx context.Context, _ *mcp.CallToolRequest, input SearchLocalInput) (
	*mcp.CallToolResult,
	SearchLocalOutput,
	error,
) {
	// Validate and set defaults
	limit := defaultLimit
	if input.Limit > 0 {
		limit = input.Limit
		if limit > maxLimit {
			return nil, SearchLocalOutput{
				ErrorMessage: fmt.Sprintf("limit cannot exceed %d", maxLimit),
			}, nil
		}
	} else if input.Limit < 0 {
		return nil, SearchLocalOutput{
			ErrorMessage: "limit must be positive",
		}, nil
	}

	offset := 0
	if input.Offset > 0 {
		offset = input.Offset
	} else if input.Offset < 0 {
		return nil, SearchLocalOutput{
			ErrorMessage: "offset cannot be negative",
		}, nil
	}

	// Build queries from input
	queries := buildQueries(input)
	if len(queries) == 0 {
		return nil, SearchLocalOutput{
			ErrorMessage: "at least one query filter must be provided",
		}, nil
	}

	// Load client configuration
	config, err := client.LoadConfig()
	if err != nil {
		return nil, SearchLocalOutput{
			ErrorMessage: fmt.Sprintf("Failed to load client configuration: %v", err),
		}, nil
	}

	// Create Directory client
	c, err := client.New(ctx, client.WithConfig(config))
	if err != nil {
		return nil, SearchLocalOutput{
			ErrorMessage: fmt.Sprintf("Failed to create Directory client: %v", err),
		}, nil
	}
	defer c.Close()

	// Execute search
	// Safe conversions: limit is capped at 1000, offset is validated by client
	limit32 := uint32(limit)   // #nosec G115
	offset32 := uint32(offset) // #nosec G115

	ch, err := c.Search(ctx, &searchv1.SearchRequest{
		Limit:   &limit32,
		Offset:  &offset32,
		Queries: queries,
	})
	if err != nil {
		return nil, SearchLocalOutput{
			ErrorMessage: fmt.Sprintf("Search failed: %v", err),
		}, nil
	}

	// Collect results
	recordCIDs := make([]string, 0, limit)

	for cid := range ch {
		if cid == "" {
			continue
		}

		recordCIDs = append(recordCIDs, cid)

		// Check if we've reached the limit
		if len(recordCIDs) >= limit {
			break
		}
	}

	// Determine if there are more results
	hasMore := len(recordCIDs) == limit

	return nil, SearchLocalOutput{
		RecordCIDs: recordCIDs,
		Count:      len(recordCIDs),
		HasMore:    hasMore,
	}, nil
}

// buildQueries converts input filters to RecordQuery objects.
func buildQueries(input SearchLocalInput) []*searchv1.RecordQuery {
	queries := make([]*searchv1.RecordQuery, 0,
		len(input.Names)+len(input.Versions)+len(input.SkillIDs)+
			len(input.SkillNames)+len(input.Locators)+len(input.Modules)+
			len(input.DomainIDs)+len(input.DomainNames))

	// Add name queries
	for _, name := range input.Names {
		queries = append(queries, &searchv1.RecordQuery{
			Type:  searchv1.RecordQueryType_RECORD_QUERY_TYPE_NAME,
			Value: name,
		})
	}

	// Add version queries
	for _, version := range input.Versions {
		queries = append(queries, &searchv1.RecordQuery{
			Type:  searchv1.RecordQueryType_RECORD_QUERY_TYPE_VERSION,
			Value: version,
		})
	}

	// Add skill-id queries
	for _, skillID := range input.SkillIDs {
		queries = append(queries, &searchv1.RecordQuery{
			Type:  searchv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL_ID,
			Value: skillID,
		})
	}

	// Add skill-name queries
	for _, skillName := range input.SkillNames {
		queries = append(queries, &searchv1.RecordQuery{
			Type:  searchv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL_NAME,
			Value: skillName,
		})
	}

	// Add locator queries
	for _, locator := range input.Locators {
		queries = append(queries, &searchv1.RecordQuery{
			Type:  searchv1.RecordQueryType_RECORD_QUERY_TYPE_LOCATOR,
			Value: locator,
		})
	}

	// Add module queries
	for _, module := range input.Modules {
		queries = append(queries, &searchv1.RecordQuery{
			Type:  searchv1.RecordQueryType_RECORD_QUERY_TYPE_MODULE,
			Value: module,
		})
	}

	// Add domain-id queries
	for _, domainID := range input.DomainIDs {
		queries = append(queries, &searchv1.RecordQuery{
			Type:  searchv1.RecordQueryType_RECORD_QUERY_TYPE_DOMAIN_ID,
			Value: domainID,
		})
	}

	// Add domain-name queries
	for _, domainName := range input.DomainNames {
		queries = append(queries, &searchv1.RecordQuery{
			Type:  searchv1.RecordQueryType_RECORD_QUERY_TYPE_DOMAIN_NAME,
			Value: domainName,
		})
	}

	return queries
}
