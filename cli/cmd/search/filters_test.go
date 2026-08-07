// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package search

import (
	"strings"
	"testing"

	searchv1 "github.com/agntcy/dir/api/search/v1"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// parseFilters binds the standard filter flags to a fresh Filters value on a
// throwaway command and parses args into it, so tests never touch the
// package-level opts singleton.
func parseFilters(t *testing.T, args ...string) *Filters {
	t.Helper()

	cmd := &cobra.Command{Use: "search", RunE: func(*cobra.Command, []string) error { return nil }}
	filters := &Filters{}
	RegisterFilterFlags(cmd, filters)

	cmd.SetArgs(args)
	require.NoError(t, cmd.Execute())

	return filters
}

// queryValue returns the value of the first query of the given type, or "" when
// no such query was built.
func queryValue(queries []*searchv1.RecordQuery, queryType searchv1.RecordQueryType) string {
	for _, query := range queries {
		if query.GetType() == queryType {
			return query.GetValue()
		}
	}

	return ""
}

// valueFilterFlags pairs every repeatable filter's flag name with the query type
// it builds. Both the include and the exclude case are driven from this table.
var valueFilterFlags = []struct {
	flag      string
	queryType searchv1.RecordQueryType
}{
	{"name", searchv1.RecordQueryType_RECORD_QUERY_TYPE_NAME},
	{"version", searchv1.RecordQueryType_RECORD_QUERY_TYPE_VERSION},
	{"skill-id", searchv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL_ID},
	{"skill", searchv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL_NAME},
	{"locator", searchv1.RecordQueryType_RECORD_QUERY_TYPE_LOCATOR},
	{"module", searchv1.RecordQueryType_RECORD_QUERY_TYPE_MODULE_NAME},
	{"domain-id", searchv1.RecordQueryType_RECORD_QUERY_TYPE_DOMAIN_ID},
	{"domain", searchv1.RecordQueryType_RECORD_QUERY_TYPE_DOMAIN_NAME},
	{"created-at", searchv1.RecordQueryType_RECORD_QUERY_TYPE_CREATED_AT},
	{"author", searchv1.RecordQueryType_RECORD_QUERY_TYPE_AUTHOR},
	{"schema-version", searchv1.RecordQueryType_RECORD_QUERY_TYPE_SCHEMA_VERSION},
	{"module-id", searchv1.RecordQueryType_RECORD_QUERY_TYPE_MODULE_ID},
	{"annotation", searchv1.RecordQueryType_RECORD_QUERY_TYPE_ANNOTATION},
}

// Every value filter must register an --exclude- twin. This walks the real flag
// set rather than the table above, so a filter added to valueFilters without its
// exclude counterpart fails here even if nobody updates the test table.
func TestEveryValueFilterHasAnExcludeTwin(t *testing.T) {
	cmd := &cobra.Command{Use: "search"}
	RegisterFilterFlags(cmd, &Filters{})

	// Booleans are negated with the tri-state =false form, not an exclude flag.
	triState := map[string]bool{"verified": true, "trusted": true, "safe": true, "help": true}

	var missing []string

	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		if strings.HasPrefix(flag.Name, "exclude-") || triState[flag.Name] {
			return
		}

		if cmd.Flags().Lookup("exclude-"+flag.Name) == nil {
			missing = append(missing, flag.Name)
		}
	})

	assert.Empty(t, missing, "value filters registered without an --exclude- twin")
}

func TestBuildQueriesIncludeFlags(t *testing.T) {
	for _, test := range valueFilterFlags {
		t.Run(test.flag, func(t *testing.T) {
			queries := BuildQueries(parseFilters(t, "--"+test.flag, "someValue"))

			require.Len(t, queries, 1)
			assert.Equal(t, test.queryType, queries[0].GetType())
			assert.Equal(t, "someValue", queries[0].GetValue())
			assert.False(t, queries[0].GetNegate(), "include flags must not negate")
		})
	}
}

func TestBuildQueriesExcludeFlags(t *testing.T) {
	for _, test := range valueFilterFlags {
		t.Run(test.flag, func(t *testing.T) {
			queries := BuildQueries(parseFilters(t, "--exclude-"+test.flag, "someValue"))

			require.Len(t, queries, 1)
			assert.Equal(t, test.queryType, queries[0].GetType())
			assert.Equal(t, "someValue", queries[0].GetValue())
			assert.True(t, queries[0].GetNegate(), "exclude flags must negate")
		})
	}
}

// Include and exclude of the same filter must land in separate queries rather
// than sharing one slice, since the server OR's includes but AND's exclusions.
func TestBuildQueriesCombinesIncludeAndExcludeOnSameFilter(t *testing.T) {
	queries := BuildQueries(parseFilters(t, "--skill", "python", "--exclude-skill", "nlp"))

	require.Len(t, queries, 2)

	for _, query := range queries {
		assert.Equal(t, searchv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL_NAME, query.GetType())
	}

	assert.Equal(t, "python", queries[0].GetValue())
	assert.False(t, queries[0].GetNegate())
	assert.Equal(t, "nlp", queries[1].GetValue())
	assert.True(t, queries[1].GetNegate())
}

func TestBuildQueriesExcludeFlagsAreRepeatable(t *testing.T) {
	queries := BuildQueries(parseFilters(t, "--exclude-domain", "a", "--exclude-domain", "b"))

	require.Len(t, queries, 2)
	assert.Equal(t, "a", queries[0].GetValue())
	assert.Equal(t, "b", queries[1].GetValue())

	for _, query := range queries {
		assert.True(t, query.GetNegate())
	}
}

// Exclusion adds no grammar: wildcards, comparison operators and a leading '!'
// all reach the server verbatim.
func TestBuildQueriesExcludePassesValuesThroughVerbatim(t *testing.T) {
	tests := []struct {
		name  string
		flag  string
		value string
	}{
		{"comparison operator", "exclude-version", ">=2.0.0"},
		{"wildcard", "exclude-name", "*bot*"},
		{"literal exclamation mark", "exclude-annotation", "!urgent:true"},
		{"colon separated locator", "exclude-locator", "docker-image:https://*"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			queries := BuildQueries(parseFilters(t, "--"+test.flag, test.value))

			require.Len(t, queries, 1)
			assert.Equal(t, test.value, queries[0].GetValue())
			assert.True(t, queries[0].GetNegate())
		})
	}
}

func TestBuildQueriesScanSeverity(t *testing.T) {
	t.Run("include", func(t *testing.T) {
		queries := BuildQueries(parseFilters(t, "--scan-severity", "MEDIUM"))

		require.Len(t, queries, 1)
		assert.Equal(t, searchv1.RecordQueryType_RECORD_QUERY_TYPE_SCAN_SEVERITY, queries[0].GetType())
		assert.Equal(t, "MEDIUM", queries[0].GetValue())
		assert.False(t, queries[0].GetNegate())
	})

	t.Run("exclude", func(t *testing.T) {
		queries := BuildQueries(parseFilters(t, "--exclude-scan-severity", "MEDIUM"))

		require.Len(t, queries, 1)
		assert.Equal(t, searchv1.RecordQueryType_RECORD_QUERY_TYPE_SCAN_SEVERITY, queries[0].GetType())
		assert.Equal(t, "MEDIUM", queries[0].GetValue())
		assert.True(t, queries[0].GetNegate())
	})
}

func TestBuildQueriesOmitsUnsetBooleanFilters(t *testing.T) {
	queries := BuildQueries(parseFilters(t))

	assert.Empty(t, queries)
}

func TestBuildQueriesBooleanFiltersAreTriState(t *testing.T) {
	tests := []struct {
		name      string
		arg       string
		queryType searchv1.RecordQueryType
		want      string
	}{
		{"verified true", "--verified", searchv1.RecordQueryType_RECORD_QUERY_TYPE_VERIFIED, "true"},
		{"verified false", "--verified=false", searchv1.RecordQueryType_RECORD_QUERY_TYPE_VERIFIED, "false"},
		{"trusted true", "--trusted", searchv1.RecordQueryType_RECORD_QUERY_TYPE_TRUSTED, "true"},
		{"trusted false", "--trusted=false", searchv1.RecordQueryType_RECORD_QUERY_TYPE_TRUSTED, "false"},
		{"safe true", "--safe", searchv1.RecordQueryType_RECORD_QUERY_TYPE_SCAN_SAFE, "true"},
		{"safe false", "--safe=false", searchv1.RecordQueryType_RECORD_QUERY_TYPE_SCAN_SAFE, "false"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			queries := BuildQueries(parseFilters(t, test.arg))

			require.Len(t, queries, 1)
			assert.Equal(t, test.want, queryValue(queries, test.queryType))
			assert.False(t, queries[0].GetNegate(), "tri-state booleans carry the value, not a negate flag")
		})
	}
}

func TestBuildQueriesCombinesNegatedTrustWithScanSeverity(t *testing.T) {
	queries := BuildQueries(parseFilters(t, "--trusted=false", "--scan-severity", "MEDIUM"))

	assert.Equal(t, "false", queryValue(queries, searchv1.RecordQueryType_RECORD_QUERY_TYPE_TRUSTED))
	assert.Equal(t, "MEDIUM", queryValue(queries, searchv1.RecordQueryType_RECORD_QUERY_TYPE_SCAN_SEVERITY))
}

// dirctl install registers the filters as persistent flags on a parent command,
// so tri-state detection has to see a flag set on a subcommand's invocation.
// Cobra shares the *pflag.Flag pointer between the two flag sets, which is what
// makes PersistentFlags().Changed() report it.
func TestBuildQueriesTriStateThroughPersistentFlags(t *testing.T) {
	newParentWithSub := func(t *testing.T, args ...string) *Filters {
		t.Helper()

		parent := &cobra.Command{Use: "install"}
		sub := &cobra.Command{Use: "run", RunE: func(*cobra.Command, []string) error { return nil }}
		parent.AddCommand(sub)

		filters := &Filters{}
		RegisterPersistentFilterFlags(parent, filters)

		parent.SetArgs(append([]string{"run"}, args...))
		require.NoError(t, parent.Execute())

		return filters
	}

	t.Run("unset", func(t *testing.T) {
		assert.Empty(t, BuildQueries(newParentWithSub(t)))
	})

	t.Run("explicit false", func(t *testing.T) {
		queries := BuildQueries(newParentWithSub(t, "--trusted=false"))

		require.Len(t, queries, 1)
		assert.Equal(t, "false", queryValue(queries, searchv1.RecordQueryType_RECORD_QUERY_TYPE_TRUSTED))
	})

	t.Run("exclude flag", func(t *testing.T) {
		queries := BuildQueries(newParentWithSub(t, "--exclude-skill", "nlp"))

		require.Len(t, queries, 1)
		assert.Equal(t, "nlp", queries[0].GetValue())
		assert.True(t, queries[0].GetNegate())
	})
}

// Filters populated in code rather than from flags keep the original
// "emit only when true" behaviour, so callers that never register flags are
// unaffected.
func TestBuildQueriesWithoutFlagSetEmitsOnlyTrueBooleans(t *testing.T) {
	assert.Empty(t, BuildQueries(&Filters{}))

	queries := BuildQueries(&Filters{Trusted: true})

	require.Len(t, queries, 1)
	assert.Equal(t, "true", queryValue(queries, searchv1.RecordQueryType_RECORD_QUERY_TYPE_TRUSTED))
}
