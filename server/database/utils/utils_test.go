// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"testing"

	searchv1 "github.com/agntcy/dir/api/search/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func applyOpts(opts []types.FilterOption) types.RecordFilters {
	var cfg types.RecordFilters
	for _, opt := range opts {
		opt(&cfg)
	}

	return cfg
}

func TestQueryToFilters_ScanOutcomeQueries(t *testing.T) {
	tests := []struct {
		name  string
		query *searchv1.RecordQuery
		check func(t *testing.T, cfg types.RecordFilters)
	}{
		{
			name:  "status is lowercased onto the include path",
			query: &searchv1.RecordQuery{Type: searchv1.RecordQueryType_RECORD_QUERY_TYPE_SCAN_STATUS, Value: "Partial"},
			check: func(t *testing.T, cfg types.RecordFilters) {
				t.Helper()
				assert.Equal(t, []string{"partial"}, cfg.ScanStatuses)
				assert.Empty(t, cfg.Excluded.ScanStatuses)
			},
		},
		{
			name:  "failure reason on the include path",
			query: &searchv1.RecordQuery{Type: searchv1.RecordQueryType_RECORD_QUERY_TYPE_SCAN_FAILURE_REASON, Value: "scanner-*"},
			check: func(t *testing.T, cfg types.RecordFilters) {
				t.Helper()
				assert.Equal(t, []string{"scanner-*"}, cfg.ScanFailureReasons)
				assert.Empty(t, cfg.Excluded.ScanFailureReasons)
			},
		},
		{
			name:  "blank status is dropped",
			query: &searchv1.RecordQuery{Type: searchv1.RecordQueryType_RECORD_QUERY_TYPE_SCAN_STATUS, Value: "  "},
			check: func(t *testing.T, cfg types.RecordFilters) {
				t.Helper()
				assert.Empty(t, cfg.ScanStatuses)
			},
		},
		{
			name:  "blank failure reason is dropped",
			query: &searchv1.RecordQuery{Type: searchv1.RecordQueryType_RECORD_QUERY_TYPE_SCAN_FAILURE_REASON, Value: ""},
			check: func(t *testing.T, cfg types.RecordFilters) {
				t.Helper()
				assert.Empty(t, cfg.ScanFailureReasons)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			opts, err := QueryToFilters([]*searchv1.RecordQuery{tc.query})
			require.NoError(t, err)
			tc.check(t, applyOpts(opts))
		})
	}
}

func TestQueryToFilters_NegateRoutesToExcluded(t *testing.T) {
	tests := []struct {
		name  string
		query *searchv1.RecordQuery
		check func(t *testing.T, cfg types.RecordFilters)
	}{
		{
			name:  "skill name",
			query: &searchv1.RecordQuery{Type: searchv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL_NAME, Value: "nlp", Negate: true},
			check: func(t *testing.T, cfg types.RecordFilters) {
				t.Helper()
				assert.Equal(t, []string{"nlp"}, cfg.Excluded.SkillNames)
				assert.Empty(t, cfg.SkillNames)
			},
		},
		{
			name:  "skill id",
			query: &searchv1.RecordQuery{Type: searchv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL_ID, Value: "10201", Negate: true},
			check: func(t *testing.T, cfg types.RecordFilters) {
				t.Helper()
				assert.Equal(t, []uint64{10201}, cfg.Excluded.SkillIDs)
			},
		},
		{
			name:  "domain name",
			query: &searchv1.RecordQuery{Type: searchv1.RecordQueryType_RECORD_QUERY_TYPE_DOMAIN_NAME, Value: "healthcare", Negate: true},
			check: func(t *testing.T, cfg types.RecordFilters) {
				t.Helper()
				assert.Equal(t, []string{"healthcare"}, cfg.Excluded.DomainNames)
			},
		},
		{
			name:  "domain id",
			query: &searchv1.RecordQuery{Type: searchv1.RecordQueryType_RECORD_QUERY_TYPE_DOMAIN_ID, Value: "901", Negate: true},
			check: func(t *testing.T, cfg types.RecordFilters) {
				t.Helper()
				assert.Equal(t, []uint64{901}, cfg.Excluded.DomainIDs)
			},
		},
		{
			name:  "module name",
			query: &searchv1.RecordQuery{Type: searchv1.RecordQueryType_RECORD_QUERY_TYPE_MODULE_NAME, Value: "core/llm", Negate: true},
			check: func(t *testing.T, cfg types.RecordFilters) {
				t.Helper()
				assert.Equal(t, []string{"core/llm"}, cfg.Excluded.ModuleNames)
			},
		},
		{
			name:  "module id",
			query: &searchv1.RecordQuery{Type: searchv1.RecordQueryType_RECORD_QUERY_TYPE_MODULE_ID, Value: "201", Negate: true},
			check: func(t *testing.T, cfg types.RecordFilters) {
				t.Helper()
				assert.Equal(t, []uint64{201}, cfg.Excluded.ModuleIDs)
			},
		},
		{
			name:  "name",
			query: &searchv1.RecordQuery{Type: searchv1.RecordQueryType_RECORD_QUERY_TYPE_NAME, Value: "*beta*", Negate: true},
			check: func(t *testing.T, cfg types.RecordFilters) {
				t.Helper()
				assert.Equal(t, []string{"*beta*"}, cfg.Excluded.Names)
			},
		},
		{
			name:  "version",
			query: &searchv1.RecordQuery{Type: searchv1.RecordQueryType_RECORD_QUERY_TYPE_VERSION, Value: "0.9.0", Negate: true},
			check: func(t *testing.T, cfg types.RecordFilters) {
				t.Helper()
				assert.Equal(t, []string{"0.9.0"}, cfg.Excluded.Versions)
			},
		},
		{
			name:  "schema version",
			query: &searchv1.RecordQuery{Type: searchv1.RecordQueryType_RECORD_QUERY_TYPE_SCHEMA_VERSION, Value: "0.6.*", Negate: true},
			check: func(t *testing.T, cfg types.RecordFilters) {
				t.Helper()
				assert.Equal(t, []string{"0.6.*"}, cfg.Excluded.SchemaVersions)
			},
		},
		{
			name:  "created at",
			query: &searchv1.RecordQuery{Type: searchv1.RecordQueryType_RECORD_QUERY_TYPE_CREATED_AT, Value: "<2024-01-01", Negate: true},
			check: func(t *testing.T, cfg types.RecordFilters) {
				t.Helper()
				assert.Equal(t, []string{"<2024-01-01"}, cfg.Excluded.CreatedAts)
			},
		},
		{
			name:  "author",
			query: &searchv1.RecordQuery{Type: searchv1.RecordQueryType_RECORD_QUERY_TYPE_AUTHOR, Value: "*@spam.com", Negate: true},
			check: func(t *testing.T, cfg types.RecordFilters) {
				t.Helper()
				assert.Equal(t, []string{"*@spam.com"}, cfg.Excluded.Authors)
			},
		},
		{
			name:  "description",
			query: &searchv1.RecordQuery{Type: searchv1.RecordQueryType_RECORD_QUERY_TYPE_DESCRIPTION, Value: "*fraud*", Negate: true},
			check: func(t *testing.T, cfg types.RecordFilters) {
				t.Helper()
				assert.Equal(t, []string{"*fraud*"}, cfg.Excluded.Descriptions)
			},
		},
		{
			name:  "scan severity",
			query: &searchv1.RecordQuery{Type: searchv1.RecordQueryType_RECORD_QUERY_TYPE_SCAN_SEVERITY, Value: "high", Negate: true},
			check: func(t *testing.T, cfg types.RecordFilters) {
				t.Helper()
				assert.Equal(t, []string{"HIGH"}, cfg.Excluded.ScanSeverities)
			},
		},
		{
			name:  "locator type only",
			query: &searchv1.RecordQuery{Type: searchv1.RecordQueryType_RECORD_QUERY_TYPE_LOCATOR, Value: "docker_image", Negate: true},
			check: func(t *testing.T, cfg types.RecordFilters) {
				t.Helper()
				assert.Equal(t, []string{"docker_image"}, cfg.Excluded.LocatorTypes)
			},
		},
		{
			name:  "locator type and url",
			query: &searchv1.RecordQuery{Type: searchv1.RecordQueryType_RECORD_QUERY_TYPE_LOCATOR, Value: "docker_image:ghcr.io/*", Negate: true},
			check: func(t *testing.T, cfg types.RecordFilters) {
				t.Helper()
				assert.Equal(t, []string{"docker_image"}, cfg.Excluded.LocatorTypes)
				assert.Equal(t, []string{"ghcr.io/*"}, cfg.Excluded.LocatorURLs)
			},
		},
		{
			name:  "annotation key and value",
			query: &searchv1.RecordQuery{Type: searchv1.RecordQueryType_RECORD_QUERY_TYPE_ANNOTATION, Value: "env:prod", Negate: true},
			check: func(t *testing.T, cfg types.RecordFilters) {
				t.Helper()
				assert.Equal(t, []string{"env"}, cfg.Excluded.AnnotationKeys)
				assert.Equal(t, []string{"prod"}, cfg.Excluded.AnnotationValues)
			},
		},
		{
			name:  "annotation key only",
			query: &searchv1.RecordQuery{Type: searchv1.RecordQueryType_RECORD_QUERY_TYPE_ANNOTATION, Value: "env", Negate: true},
			check: func(t *testing.T, cfg types.RecordFilters) {
				t.Helper()
				assert.Equal(t, []string{"env"}, cfg.Excluded.AnnotationKeys)
				assert.Empty(t, cfg.Excluded.AnnotationValues)
			},
		},
		{
			name:  "scan status",
			query: &searchv1.RecordQuery{Type: searchv1.RecordQueryType_RECORD_QUERY_TYPE_SCAN_STATUS, Value: "FAILED", Negate: true},
			check: func(t *testing.T, cfg types.RecordFilters) {
				t.Helper()
				assert.Equal(t, []string{"failed"}, cfg.Excluded.ScanStatuses)
			},
		},
		{
			name:  "scan failure reason",
			query: &searchv1.RecordQuery{Type: searchv1.RecordQueryType_RECORD_QUERY_TYPE_SCAN_FAILURE_REASON, Value: "Source-Unreachable", Negate: true},
			check: func(t *testing.T, cfg types.RecordFilters) {
				t.Helper()
				assert.Equal(t, []string{"source-unreachable"}, cfg.Excluded.ScanFailureReasons)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			opts, err := QueryToFilters([]*searchv1.RecordQuery{tc.query})
			require.NoError(t, err)
			tc.check(t, applyOpts(opts))
		})
	}
}

func TestQueryToFilters_NegateFlipsBooleans(t *testing.T) {
	tests := []struct {
		name     string
		query    *searchv1.RecordQuery
		wantPtr  func(cfg types.RecordFilters) *bool
		wantBool bool
	}{
		{
			name:     "verified true negated becomes false",
			query:    &searchv1.RecordQuery{Type: searchv1.RecordQueryType_RECORD_QUERY_TYPE_VERIFIED, Value: "true", Negate: true},
			wantPtr:  func(cfg types.RecordFilters) *bool { return cfg.Verified },
			wantBool: false,
		},
		{
			name:     "trusted true negated becomes false",
			query:    &searchv1.RecordQuery{Type: searchv1.RecordQueryType_RECORD_QUERY_TYPE_TRUSTED, Value: "true", Negate: true},
			wantPtr:  func(cfg types.RecordFilters) *bool { return cfg.Trusted },
			wantBool: false,
		},
		{
			name:     "scan safe false negated becomes true",
			query:    &searchv1.RecordQuery{Type: searchv1.RecordQueryType_RECORD_QUERY_TYPE_SCAN_SAFE, Value: "false", Negate: true},
			wantPtr:  func(cfg types.RecordFilters) *bool { return cfg.ScanSafe },
			wantBool: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			opts, err := QueryToFilters([]*searchv1.RecordQuery{tc.query})
			require.NoError(t, err)

			cfg := applyOpts(opts)
			ptr := tc.wantPtr(cfg)
			require.NotNil(t, ptr)
			assert.Equal(t, tc.wantBool, *ptr)
		})
	}
}

func TestQueryToFilters_NonNegatedUnaffected(t *testing.T) {
	opts, err := QueryToFilters([]*searchv1.RecordQuery{
		{Type: searchv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL_NAME, Value: "nlp"},
	})
	require.NoError(t, err)

	cfg := applyOpts(opts)
	assert.Equal(t, []string{"nlp"}, cfg.SkillNames)
	assert.Empty(t, cfg.Excluded.SkillNames)
}
