// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package search

import (
	"strconv"

	searchv1 "github.com/agntcy/dir/api/search/v1"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// FilterValues holds one filter's include and exclude values.
//
// The two are kept apart rather than sharing a slice because the server treats
// them differently: values of one filter are OR'd together, while each excluded
// value becomes its own NOT EXISTS condition, and those are AND'd. Splitting at
// the flag means "--skill python --exclude-skill nlp" needs no parsing to
// separate the two intents.
type FilterValues struct {
	Include []string
	Exclude []string
}

// Filters holds the values for all record-search filter flags.
// Embed this in any command's options struct to reuse the standard filter set.
type Filters struct {
	Names          FilterValues
	Versions       FilterValues
	SkillIDs       FilterValues
	SkillNames     FilterValues
	Locators       FilterValues
	Modules        FilterValues
	DomainIDs      FilterValues
	DomainNames    FilterValues
	CreatedAts     FilterValues
	Authors        FilterValues
	SchemaVersions FilterValues
	ModuleIDs      FilterValues
	Annotations    FilterValues
	Owners         FilterValues

	ScanStatuses       FilterValues
	ScanFailureReasons FilterValues

	// ScanSeverity is a single threshold rather than a repeatable match, so it
	// sits outside valueFilters.
	ScanSeverity        string
	ExcludeScanSeverity string

	Verified bool
	Trusted  bool
	Safe     bool

	// flags is the flag set the filter flags were registered on. It tells
	// "--trusted=false" (filter for records that failed the check) apart from
	// the flag being absent (do not filter on trust at all). Nil when Filters
	// is populated programmatically rather than from flags.
	flags *pflag.FlagSet
}

// changed reports whether the named filter flag was explicitly set.
func (f *Filters) changed(name string) bool {
	return f.flags != nil && f.flags.Changed(name)
}

// valueFilter describes one repeatable filter: the flag pair it registers, the
// help text for each, and the query type both build. It is the single source of
// truth for flag registration and query building alike, so adding a filter is
// one entry rather than three edits.
type valueFilter struct {
	flag         string
	usage        string
	excludeUsage string
	queryType    searchv1.RecordQueryType
	values       *FilterValues
}

// valueFilters returns the filter table bound to f's fields.
func (f *Filters) valueFilters() []valueFilter {
	return []valueFilter{
		{
			flag:         "name",
			usage:        "Search for records with specific name (e.g., --name 'my-agent' --name 'web-*')",
			excludeUsage: "Exclude records with specific name (e.g., --exclude-name 'test-*')",
			queryType:    searchv1.RecordQueryType_RECORD_QUERY_TYPE_NAME,
			values:       &f.Names,
		},
		{
			flag:         "version",
			usage:        "Search for records with specific version (e.g., --version 'v1.0.0' --version 'v1.*')",
			excludeUsage: "Exclude records with specific version (e.g., --exclude-version '>=2.0.0')",
			queryType:    searchv1.RecordQueryType_RECORD_QUERY_TYPE_VERSION,
			values:       &f.Versions,
		},
		{
			flag:         "skill-id",
			usage:        "Search for records with specific skill ID (e.g., --skill-id '10201')",
			excludeUsage: "Exclude records with specific skill ID (e.g., --exclude-skill-id '10201')",
			queryType:    searchv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL_ID,
			values:       &f.SkillIDs,
		},
		{
			flag:         "skill",
			usage:        "Search for records with specific skill name (e.g., --skill 'natural_language_processing')",
			excludeUsage: "Exclude records with specific skill name (e.g., --exclude-skill 'natural_language_processing')",
			queryType:    searchv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL_NAME,
			values:       &f.SkillNames,
		},
		{
			flag:         "locator",
			usage:        "Search for records with specific locator type (e.g., --locator 'docker-image')",
			excludeUsage: "Exclude records with specific locator type (e.g., --exclude-locator 'docker-image')",
			queryType:    searchv1.RecordQueryType_RECORD_QUERY_TYPE_LOCATOR,
			values:       &f.Locators,
		},
		{
			flag:         "module",
			usage:        "Search for records with specific module (e.g., --module 'core/llm/model')",
			excludeUsage: "Exclude records with specific module (e.g., --exclude-module 'core/llm/model')",
			queryType:    searchv1.RecordQueryType_RECORD_QUERY_TYPE_MODULE_NAME,
			values:       &f.Modules,
		},
		{
			flag:         "domain-id",
			usage:        "Search for records with specific domain ID (e.g., --domain-id '604')",
			excludeUsage: "Exclude records with specific domain ID (e.g., --exclude-domain-id '604')",
			queryType:    searchv1.RecordQueryType_RECORD_QUERY_TYPE_DOMAIN_ID,
			values:       &f.DomainIDs,
		},
		{
			flag:         "domain",
			usage:        "Search for records with specific domain name (e.g., --domain '*education*')",
			excludeUsage: "Exclude records with specific domain name (e.g., --exclude-domain '*education*')",
			queryType:    searchv1.RecordQueryType_RECORD_QUERY_TYPE_DOMAIN_NAME,
			values:       &f.DomainNames,
		},
		{
			flag:         "created-at",
			usage:        "Search for records with specific created_at timestamp (e.g., --created-at '2024-*')",
			excludeUsage: "Exclude records with specific created_at timestamp (e.g., --exclude-created-at '<2024-01-01')",
			queryType:    searchv1.RecordQueryType_RECORD_QUERY_TYPE_CREATED_AT,
			values:       &f.CreatedAts,
		},
		{
			flag:         "author",
			usage:        "Search for records with specific author (e.g., --author 'john*')",
			excludeUsage: "Exclude records with specific author (e.g., --exclude-author 'bot*')",
			queryType:    searchv1.RecordQueryType_RECORD_QUERY_TYPE_AUTHOR,
			values:       &f.Authors,
		},
		{
			flag:         "schema-version",
			usage:        "Filter records by OASF schema version (e.g., --schema-version '0.8.*'); in natural-language search mode, also restricts the extractor to that taxonomy version (repeatable)",
			excludeUsage: "Exclude records with specific OASF schema version (e.g., --exclude-schema-version '0.7.*')",
			queryType:    searchv1.RecordQueryType_RECORD_QUERY_TYPE_SCHEMA_VERSION,
			values:       &f.SchemaVersions,
		},
		{
			flag:         "module-id",
			usage:        "Search for records with specific module ID (e.g., --module-id '201')",
			excludeUsage: "Exclude records with specific module ID (e.g., --exclude-module-id '201')",
			queryType:    searchv1.RecordQueryType_RECORD_QUERY_TYPE_MODULE_ID,
			values:       &f.ModuleIDs,
		},
		{
			flag:         "scan-status",
			usage:        "Search for records with a scan result of a given status: completed, partial, or failed (e.g., --scan-status failed)",
			excludeUsage: "Exclude records with a scan result of a given status (e.g., --exclude-scan-status partial)",
			queryType:    searchv1.RecordQueryType_RECORD_QUERY_TYPE_SCAN_STATUS,
			values:       &f.ScanStatuses,
		},
		{
			flag:         "scan-failure-reason",
			usage:        "Search for records whose scan could not run for a given reason (e.g., --scan-failure-reason source-unreachable --scan-failure-reason 'scanner-*')",
			excludeUsage: "Exclude records whose scan could not run for a given reason (e.g., --exclude-scan-failure-reason llm-not-configured)",
			queryType:    searchv1.RecordQueryType_RECORD_QUERY_TYPE_SCAN_FAILURE_REASON,
			values:       &f.ScanFailureReasons,
		},
		{
			flag:         "annotation",
			usage:        "Search for records with specific annotation in key:value format (e.g., --annotation 'manager:alice' --annotation 'team:*')",
			excludeUsage: "Exclude records with specific annotation in key:value format (e.g., --exclude-annotation 'env:test')",
			queryType:    searchv1.RecordQueryType_RECORD_QUERY_TYPE_ANNOTATION,
			values:       &f.Annotations,
		},
		{
			flag:         "owner",
			usage:        "Search for records with specific owner identity (e.g., --owner 'spiffe://example.org/agent')",
			excludeUsage: "Exclude records with specific owner identity (e.g., --exclude-owner 'spiffe://example.org/agent')",
			queryType:    searchv1.RecordQueryType_RECORD_QUERY_TYPE_OWNER,
			values:       &f.Owners,
		},
	}
}

// RegisterFilterFlags binds the standard search-filter flags to cmd, storing
// values in f. Call this from your command's init().
func RegisterFilterFlags(cmd *cobra.Command, f *Filters) {
	registerFilterFlags(cmd.Flags(), f)
}

// RegisterPersistentFilterFlags binds search-filter flags as persistent flags on
// cmd so subcommands inherit them.
func RegisterPersistentFilterFlags(cmd *cobra.Command, f *Filters) {
	registerFilterFlags(cmd.PersistentFlags(), f)
}

func registerFilterFlags(flags *pflag.FlagSet, f *Filters) {
	f.flags = flags

	for _, filter := range f.valueFilters() {
		flags.StringArrayVar(&filter.values.Include, filter.flag, nil, filter.usage)
		flags.StringArrayVar(&filter.values.Exclude, "exclude-"+filter.flag, nil, filter.excludeUsage)
	}

	flags.StringVar(&f.ScanSeverity, "scan-severity", "",
		"Filter for records whose highest scan severity meets or exceeds a threshold (NONE, INFO, LOW, MEDIUM, HIGH, CRITICAL)")
	flags.StringVar(&f.ExcludeScanSeverity, "exclude-scan-severity", "",
		"Exclude records with a scan report at or above a threshold (NONE, INFO, LOW, MEDIUM, HIGH, CRITICAL); never-scanned records are kept")

	flags.BoolVar(&f.Verified, "verified", false,
		"Filter for records with verified name ownership (--verified) or without it (--verified=false)")
	flags.BoolVar(&f.Trusted, "trusted", false,
		"Filter for records with a trusted signature (--trusted) or without one (--trusted=false)")
	flags.BoolVar(&f.Safe, "safe", false,
		"Filter for records where every security scanner reported is_safe=true (--safe), or where at least one did not (--safe=false)")
}

// BuildQueries converts populated filter fields into API query objects.
func BuildQueries(f *Filters) []*searchv1.RecordQuery {
	filters := f.valueFilters()

	var size int
	for _, filter := range filters {
		size += len(filter.values.Include) + len(filter.values.Exclude)
	}

	queries := make([]*searchv1.RecordQuery, 0, size)

	for _, filter := range filters {
		for _, value := range filter.values.Include {
			queries = append(queries, &searchv1.RecordQuery{
				Type:  filter.queryType,
				Value: value,
			})
		}

		for _, value := range filter.values.Exclude {
			queries = append(queries, &searchv1.RecordQuery{
				Type:   filter.queryType,
				Value:  value,
				Negate: true,
			})
		}
	}

	if f.ScanSeverity != "" {
		queries = append(queries, &searchv1.RecordQuery{
			Type:  searchv1.RecordQueryType_RECORD_QUERY_TYPE_SCAN_SEVERITY,
			Value: f.ScanSeverity,
		})
	}

	if f.ExcludeScanSeverity != "" {
		queries = append(queries, &searchv1.RecordQuery{
			Type:   searchv1.RecordQueryType_RECORD_QUERY_TYPE_SCAN_SEVERITY,
			Value:  f.ExcludeScanSeverity,
			Negate: true,
		})
	}

	// Boolean filters are tri-state: an absent flag does not filter at all,
	// while an explicit --trusted=false filters for records that failed the
	// check. Filters populated in code have no flag set and keep the original
	// "emit only when true" behaviour.
	boolFilters := []struct {
		flag      string
		value     bool
		queryType searchv1.RecordQueryType
	}{
		{"verified", f.Verified, searchv1.RecordQueryType_RECORD_QUERY_TYPE_VERIFIED},
		{"trusted", f.Trusted, searchv1.RecordQueryType_RECORD_QUERY_TYPE_TRUSTED},
		{"safe", f.Safe, searchv1.RecordQueryType_RECORD_QUERY_TYPE_SCAN_SAFE},
	}

	for _, boolFilter := range boolFilters {
		if !boolFilter.value && !f.changed(boolFilter.flag) {
			continue
		}

		queries = append(queries, &searchv1.RecordQuery{
			Type:  boolFilter.queryType,
			Value: strconv.FormatBool(boolFilter.value),
		})
	}

	return queries
}
