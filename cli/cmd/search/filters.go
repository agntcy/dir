// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package search

import (
	searchv1 "github.com/agntcy/dir/api/search/v1"
	"github.com/spf13/cobra"
)

// Filters holds the values for all record-search filter flags.
// Embed this in any command's options struct to reuse the standard filter set.
type Filters struct {
	Names          []string
	Versions       []string
	SkillIDs       []string
	SkillNames     []string
	Locators       []string
	Modules        []string
	DomainIDs      []string
	DomainNames    []string
	CreatedAts     []string
	Authors        []string
	SchemaVersions []string
	ModuleIDs      []string
	Verified       bool
	Trusted        bool
}

// RegisterFilterFlags binds the standard search-filter flags to cmd, storing
// values in f. Call this from your command's init().
func RegisterFilterFlags(cmd *cobra.Command, f *Filters) {
	flags := cmd.Flags()

	flags.StringArrayVar(&f.Names, "name", nil,
		"Search for records with specific name (e.g., --name 'my-agent' --name 'web-*')")
	flags.StringArrayVar(&f.Versions, "version", nil,
		"Search for records with specific version (e.g., --version 'v1.0.0' --version 'v1.*')")
	flags.StringArrayVar(&f.SkillIDs, "skill-id", nil,
		"Search for records with specific skill ID (e.g., --skill-id '10201')")
	flags.StringArrayVar(&f.SkillNames, "skill", nil,
		"Search for records with specific skill name (e.g., --skill 'natural_language_processing')")
	flags.StringArrayVar(&f.Locators, "locator", nil,
		"Search for records with specific locator type (e.g., --locator 'docker-image')")
	flags.StringArrayVar(&f.Modules, "module", nil,
		"Search for records with specific module (e.g., --module 'core/llm/model')")
	flags.StringArrayVar(&f.DomainIDs, "domain-id", nil,
		"Search for records with specific domain ID (e.g., --domain-id '604')")
	flags.StringArrayVar(&f.DomainNames, "domain", nil,
		"Search for records with specific domain name (e.g., --domain '*education*')")
	flags.StringArrayVar(&f.CreatedAts, "created-at", nil,
		"Search for records with specific created_at timestamp (e.g., --created-at '2024-*')")
	flags.StringArrayVar(&f.Authors, "author", nil,
		"Search for records with specific author (e.g., --author 'john*')")
	flags.StringArrayVar(&f.SchemaVersions, "schema-version", nil,
		"Search for records with specific schema version (e.g., --schema-version '0.8.*')")
	flags.StringArrayVar(&f.ModuleIDs, "module-id", nil,
		"Search for records with specific module ID (e.g., --module-id '201')")
	flags.BoolVar(&f.Verified, "verified", false,
		"Filter for records with verified name ownership only")
	flags.BoolVar(&f.Trusted, "trusted", false,
		"Filter for records with trusted signature only (signature verification passed)")
}

// queryMapping maps a Filters field accessor to its RecordQueryType.
type queryMapping struct {
	values    []string
	queryType searchv1.RecordQueryType
}

// BuildQueries converts populated filter fields into API query objects.
func BuildQueries(f *Filters) []*searchv1.RecordQuery {
	mappings := []queryMapping{
		{f.Names, searchv1.RecordQueryType_RECORD_QUERY_TYPE_NAME},
		{f.Versions, searchv1.RecordQueryType_RECORD_QUERY_TYPE_VERSION},
		{f.SkillIDs, searchv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL_ID},
		{f.SkillNames, searchv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL_NAME},
		{f.Locators, searchv1.RecordQueryType_RECORD_QUERY_TYPE_LOCATOR},
		{f.Modules, searchv1.RecordQueryType_RECORD_QUERY_TYPE_MODULE_NAME},
		{f.DomainIDs, searchv1.RecordQueryType_RECORD_QUERY_TYPE_DOMAIN_ID},
		{f.DomainNames, searchv1.RecordQueryType_RECORD_QUERY_TYPE_DOMAIN_NAME},
		{f.CreatedAts, searchv1.RecordQueryType_RECORD_QUERY_TYPE_CREATED_AT},
		{f.Authors, searchv1.RecordQueryType_RECORD_QUERY_TYPE_AUTHOR},
		{f.SchemaVersions, searchv1.RecordQueryType_RECORD_QUERY_TYPE_SCHEMA_VERSION},
		{f.ModuleIDs, searchv1.RecordQueryType_RECORD_QUERY_TYPE_MODULE_ID},
	}

	var size int
	for _, m := range mappings {
		size += len(m.values)
	}

	queries := make([]*searchv1.RecordQuery, 0, size)

	for _, m := range mappings {
		for _, v := range m.values {
			queries = append(queries, &searchv1.RecordQuery{
				Type:  m.queryType,
				Value: v,
			})
		}
	}

	if f.Verified {
		queries = append(queries, &searchv1.RecordQuery{
			Type:  searchv1.RecordQueryType_RECORD_QUERY_TYPE_VERIFIED,
			Value: "true",
		})
	}

	if f.Trusted {
		queries = append(queries, &searchv1.RecordQuery{
			Type:  searchv1.RecordQueryType_RECORD_QUERY_TYPE_TRUSTED,
			Value: "true",
		})
	}

	return queries
}
