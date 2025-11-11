// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"testing"

	searchv1 "github.com/agntcy/dir/api/search/v1"
	"github.com/stretchr/testify/assert"
)

func TestBuildQueries(t *testing.T) {
	tests := []struct {
		name     string
		input    SearchLocalInput
		expected int
	}{
		{
			name: "single name query",
			input: SearchLocalInput{
				Names: []string{"test-agent"},
			},
			expected: 1,
		},
		{
			name: "multiple query types",
			input: SearchLocalInput{
				Names:      []string{"agent-*"},
				Versions:   []string{"v1.*"},
				SkillNames: []string{"*python*"},
			},
			expected: 3,
		},
		{
			name: "all query types",
			input: SearchLocalInput{
				Names:       []string{"agent1", "agent2"},
				Versions:    []string{"v1.0.0"},
				SkillIDs:    []string{"10201"},
				SkillNames:  []string{"Python"},
				Locators:    []string{"docker-image:*"},
				Modules:     []string{"core-module"},
				DomainIDs:   []string{"604"},
				DomainNames: []string{"*education*"},
			},
			expected: 9,
		},
		{
			name:     "no queries",
			input:    SearchLocalInput{},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queries := buildQueries(tt.input)
			assert.Len(t, queries, tt.expected)
		})
	}
}

func TestBuildQueriesTypes(t *testing.T) {
	input := SearchLocalInput{
		Names:       []string{"test-agent"},
		Versions:    []string{"v1.0.0"},
		SkillIDs:    []string{"10201"},
		SkillNames:  []string{"Python"},
		Locators:    []string{"docker-image:test"},
		Modules:     []string{"core"},
		DomainIDs:   []string{"604"},
		DomainNames: []string{"*education*"},
	}

	queries := buildQueries(input)
	assert.Len(t, queries, 8)

	// Verify query types are correctly mapped
	expectedTypes := []searchv1.RecordQueryType{
		searchv1.RecordQueryType_RECORD_QUERY_TYPE_NAME,
		searchv1.RecordQueryType_RECORD_QUERY_TYPE_VERSION,
		searchv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL_ID,
		searchv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL_NAME,
		searchv1.RecordQueryType_RECORD_QUERY_TYPE_LOCATOR,
		searchv1.RecordQueryType_RECORD_QUERY_TYPE_MODULE,
		searchv1.RecordQueryType_RECORD_QUERY_TYPE_DOMAIN_ID,
		searchv1.RecordQueryType_RECORD_QUERY_TYPE_DOMAIN_NAME,
	}

	for i, query := range queries {
		assert.Equal(t, expectedTypes[i], query.GetType())
	}
}

func TestBuildQueriesValues(t *testing.T) {
	input := SearchLocalInput{
		Names:    []string{"agent-*", "test-agent"},
		Versions: []string{"v1.*"},
	}

	queries := buildQueries(input)
	assert.Len(t, queries, 3)

	// Verify values are preserved
	assert.Equal(t, "agent-*", queries[0].GetValue())
	assert.Equal(t, "test-agent", queries[1].GetValue())
	assert.Equal(t, "v1.*", queries[2].GetValue())
}
