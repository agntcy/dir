// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

func (r *Record) GetAgents(filters ...string) ([]any, error) {
	// Convert filters
	// = filters

	// Required filters to support:
	// - Appendix A: Filter Expression Syntax
	// - Filter by version, e.g. "version=1.0.0", or "include_versions=true" to include all versions
	// - No version filter, default to latest (last created_at)

	// Fetch
	// run a query here to fetch: records, join with skills, modules, domains, signature
	// var records []Record
	// err := db.Preload("Skills").Preload("Modules").Preload("Domains").Preload("Annotations").Preload("Signatures").Find(&records).Error
	// if err != nil {
	// 	return nil, err
	// }

	// // Apply conversion to catalog format
	// result := make([]interface{}, len(records))
	// for i, record := range records {
	// 	catalog, err := record.ToCatalog()
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	result[i] = catalog
	// }
	return nil, nil
}

// returns either a single Entry or a nested Entry.
func (r *Record) ToCatalog() (any, error) {
	// Fetch
	return nil, nil // nolint:nilnil
}

type CollectionSummary struct {
	Module  []string // can only be MCP, A2A, or Skills
	Skills  []string
	Domains []string
}

func (c *CollectionSummary) WellKnown() any {
	return `
{
	"specVersion": "1.0",
	"host": {
		"displayName": "Acme Enterprise AI",
		"identifier": "did:web:acme.com"
	},
	"entries": [], // published entries, can be MANY
	"collections": [
		{
			"displayName": "MCP Record Catalog",
			"url": "https://localhost:8080/agents?type=mcp",
			"description": "Returns all available MCP records."
		},
		{
			"displayName": "A2A Record Catalog",
			"url": "https://localhost:8080/agents?type=a2a",
			"description": "Returns all available A2A records."
		},
		{
			"displayName": "Agents with Skill X",
			"url": "https://localhost:8080/agents?type=skill_x&skill_name={skill_name}",
			"description": "Returns all available agents with Skill X."
		}
	]
}
  `
}
