// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package search

import (
	searchv1 "github.com/agntcy/dir/api/search/v1"
	"github.com/spf13/cobra"
)

var opts = &options{}

type options struct {
	Limit  uint32
	Offset uint32
	Format string

	// Direct field flags (consistent with routing search)
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
}

// registerFlags adds search flags to the command.
func registerFlags(cmd *cobra.Command) {
	flags := cmd.Flags()

	flags.StringVar(&opts.Format, "format", "cid", "Output format: cid (default) or record")
	flags.Uint32Var(&opts.Limit, "limit", 100, "Maximum number of results to return (default: 100)") //nolint:mnd
	flags.Uint32Var(&opts.Offset, "offset", 0, "Pagination offset (default: 0)")

	// Direct field flags
	flags.StringArrayVar(&opts.Names, "name", nil, "Search for records with specific name (can be repeated)")
	flags.StringArrayVar(&opts.Versions, "version", nil, "Search for records with specific version (can be repeated)")
	flags.StringArrayVar(&opts.SkillIDs, "skill-id", nil, "Search for records with specific skill ID (can be repeated)")
	flags.StringArrayVar(&opts.SkillNames, "skill", nil, "Search for records with specific skill name (can be repeated)")
	flags.StringArrayVar(&opts.Locators, "locator", nil, "Search for records with specific locator type (can be repeated)")
	flags.StringArrayVar(&opts.Modules, "module", nil, "Search for records with specific module (can be repeated)")
	flags.StringArrayVar(&opts.DomainIDs, "domain-id", nil, "Search for records with specific domain ID (can be repeated)")
	flags.StringArrayVar(&opts.DomainNames, "domain", nil, "Search for records with specific domain name (can be repeated)")
	flags.StringArrayVar(&opts.CreatedAts, "created-at", nil, "Search for records with specific created_at timestamp (can be repeated)")
	flags.StringArrayVar(&opts.Authors, "author", nil, "Search for records with specific author (can be repeated)")
	flags.StringArrayVar(&opts.SchemaVersions, "schema-version", nil, "Search for records with specific schema version (can be repeated)")
	flags.StringArrayVar(&opts.ModuleIDs, "module-id", nil, "Search for records with specific module ID (can be repeated)")

	// Add examples in flag help
	flags.Lookup("name").Usage = "Search for records with specific name (e.g., --name 'my-agent' --name 'web-*')"
	flags.Lookup("version").Usage = "Search for records with specific version (e.g., --version 'v1.0.0' --version 'v1.*')"
	flags.Lookup("skill-id").Usage = "Search for records with specific skill ID (e.g., --skill-id '10201')"
	flags.Lookup("skill").Usage = "Search for records with specific skill name (e.g., --skill 'natural_language_processing' --skill 'audio')"
	flags.Lookup("locator").Usage = "Search for records with specific locator type (e.g., --locator 'docker-image')"
	flags.Lookup("module").Usage = "Search for records with specific module (e.g., --module 'core/llm/model')"
	flags.Lookup("domain-id").Usage = "Search for records with specific domain ID (e.g., --domain-id '604')"
	flags.Lookup("domain").Usage = "Search for records with specific domain name (e.g., --domain '*education*' --domain 'healthcare/*')"
	flags.Lookup("created-at").Usage = "Search for records with specific created_at timestamp (e.g., --created-at '2024-*')"
	flags.Lookup("author").Usage = "Search for records with specific author (e.g., --author 'john*')"
	flags.Lookup("schema-version").Usage = "Search for records with specific schema version (e.g., --schema-version '0.8.*')"
	flags.Lookup("module-id").Usage = "Search for records with specific module ID (e.g., --module-id '201')"
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
