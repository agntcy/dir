// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"context"
	"testing"

	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/stretchr/testify/assert"
)

func TestQueryMatchesLabels(t *testing.T) {
	testCases := []struct {
		name     string
		query    *routingv1.RecordQuery
		labels   []string
		expected bool
	}{
		// Skill queries
		{
			name: "skill_exact_match",
			query: &routingv1.RecordQuery{
				Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL,
				Value: "AI",
			},
			labels:   []string{"/skills/AI", "/skills/web-development"},
			expected: true,
		},
		{
			name: "skill_prefix_match",
			query: &routingv1.RecordQuery{
				Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL,
				Value: "AI",
			},
			labels:   []string{"/skills/AI/ML", "/skills/web-development"},
			expected: true,
		},
		{
			name: "skill_no_match",
			query: &routingv1.RecordQuery{
				Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL,
				Value: "blockchain",
			},
			labels:   []string{"/skills/AI", "/skills/web-development"},
			expected: false,
		},
		{
			name: "skill_partial_no_match",
			query: &routingv1.RecordQuery{
				Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL,
				Value: "AI/ML/deep-learning",
			},
			labels:   []string{"/skills/AI/ML", "/skills/web-development"},
			expected: false,
		},

		// Locator queries
		{
			name: "locator_exact_match",
			query: &routingv1.RecordQuery{
				Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_LOCATOR,
				Value: "docker-image",
			},
			labels:   []string{"/locators/docker-image", "/skills/AI"},
			expected: true,
		},
		{
			name: "locator_no_match",
			query: &routingv1.RecordQuery{
				Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_LOCATOR,
				Value: "git-repo",
			},
			labels:   []string{"/locators/docker-image", "/skills/AI"},
			expected: false,
		},

		// Unspecified queries
		{
			name: "unspecified_always_matches",
			query: &routingv1.RecordQuery{
				Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_UNSPECIFIED,
				Value: "anything",
			},
			labels:   []string{"/skills/AI"},
			expected: true,
		},
		{
			name: "unspecified_matches_empty_labels",
			query: &routingv1.RecordQuery{
				Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_UNSPECIFIED,
				Value: "anything",
			},
			labels:   []string{},
			expected: true,
		},

		// Edge cases
		{
			name: "empty_labels",
			query: &routingv1.RecordQuery{
				Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL,
				Value: "AI",
			},
			labels:   []string{},
			expected: false,
		},
		{
			name: "case_sensitive_skill",
			query: &routingv1.RecordQuery{
				Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL,
				Value: "ai", // lowercase
			},
			labels:   []string{"/skills/AI"}, // uppercase
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := QueryMatchesLabels(tc.query, tc.labels)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestMatchesAllQueries(t *testing.T) {
	ctx := t.Context()
	testCID := "bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi"

	// Mock label retriever that returns predefined labels
	mockLabelRetriever := func(_ context.Context, cid string) []string {
		if cid == testCID {
			return []string{"/skills/AI", "/skills/AI/ML", "/domains/technology", "/locators/docker-image"}
		}

		return []string{}
	}

	testCases := []struct {
		name     string
		cid      string
		queries  []*routingv1.RecordQuery
		expected bool
	}{
		{
			name:     "no_queries_matches_all",
			cid:      testCID,
			queries:  []*routingv1.RecordQuery{},
			expected: true,
		},
		{
			name: "single_matching_query",
			cid:  testCID,
			queries: []*routingv1.RecordQuery{
				{
					Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL,
					Value: "AI",
				},
			},
			expected: true,
		},
		{
			name: "single_non_matching_query",
			cid:  testCID,
			queries: []*routingv1.RecordQuery{
				{
					Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL,
					Value: "blockchain",
				},
			},
			expected: false,
		},
		{
			name: "multiple_matching_queries_and_logic",
			cid:  testCID,
			queries: []*routingv1.RecordQuery{
				{
					Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL,
					Value: "AI",
				},
				{
					Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_LOCATOR,
					Value: "docker-image",
				},
			},
			expected: true,
		},
		{
			name: "mixed_matching_and_non_matching_queries",
			cid:  testCID,
			queries: []*routingv1.RecordQuery{
				{
					Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL,
					Value: "AI", // matches
				},
				{
					Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL,
					Value: "blockchain", // doesn't match
				},
			},
			expected: false, // AND logic - all must match
		},
		{
			name: "unknown_cid",
			cid:  "unknown-cid",
			queries: []*routingv1.RecordQuery{
				{
					Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL,
					Value: "AI",
				},
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := MatchesAllQueries(ctx, tc.cid, tc.queries, mockLabelRetriever)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGetMatchingQueries(t *testing.T) {
	testQueries := []*routingv1.RecordQuery{
		{
			Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL,
			Value: "AI",
		},
		{
			Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL,
			Value: "web-development",
		},
		{
			Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_LOCATOR,
			Value: "docker-image",
		},
	}

	testCases := []struct {
		name              string
		labelKey          string
		expectedMatches   int
		expectedQueryType routingv1.RecordQueryType
	}{
		{
			name:              "skill_ai_matches",
			labelKey:          "/skills/AI/bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi/peer1",
			expectedMatches:   1,
			expectedQueryType: routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL,
		},
		{
			name:              "skill_web_dev_matches",
			labelKey:          "/skills/web-development/bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi/peer1",
			expectedMatches:   1,
			expectedQueryType: routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL,
		},
		{
			name:              "locator_matches",
			labelKey:          "/locators/docker-image/bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi/peer1",
			expectedMatches:   1,
			expectedQueryType: routingv1.RecordQueryType_RECORD_QUERY_TYPE_LOCATOR,
		},
		{
			name:            "no_matches",
			labelKey:        "/skills/blockchain/bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi/peer1",
			expectedMatches: 0,
		},
		{
			name:            "malformed_key",
			labelKey:        "/invalid-key",
			expectedMatches: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			matches := GetMatchingQueries(tc.labelKey, testQueries)
			assert.Len(t, matches, tc.expectedMatches)

			if tc.expectedMatches > 0 {
				assert.Equal(t, tc.expectedQueryType, matches[0].GetType())
			}
		})
	}
}

func TestQueryMatchingEdgeCases(t *testing.T) {
	t.Run("nil_query", func(t *testing.T) {
		// This should not panic
		result := QueryMatchesLabels(nil, []string{"/skills/AI"})
		assert.False(t, result)
	})

	t.Run("unknown_query_type", func(t *testing.T) {
		query := &routingv1.RecordQuery{
			Type:  routingv1.RecordQueryType(999), // Unknown type
			Value: "test",
		}
		result := QueryMatchesLabels(query, []string{"/skills/AI"})
		assert.False(t, result)
	})

	t.Run("empty_query_value", func(t *testing.T) {
		query := &routingv1.RecordQuery{
			Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL,
			Value: "",
		}
		result := QueryMatchesLabels(query, []string{"/skills/"})
		assert.True(t, result) // Empty value matches "/skills/" prefix
	})

	t.Run("nil_labels", func(t *testing.T) {
		query := &routingv1.RecordQuery{
			Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL,
			Value: "AI",
		}
		result := QueryMatchesLabels(query, nil)
		assert.False(t, result)
	})
}

// Test the integration between MatchesAllQueries and QueryMatchesLabels.
func TestQueryMatchingIntegration(t *testing.T) {
	ctx := t.Context()

	// Test with a more complex label retriever
	complexLabelRetriever := func(_ context.Context, cid string) []string {
		switch cid {
		case "ai-record":
			return []string{"/skills/AI", "/skills/AI/ML", "/skills/AI/NLP"}
		case "web-record":
			return []string{"/skills/web-development", "/skills/javascript", "/locators/git-repo"}
		case "mixed-record":
			return []string{"/skills/AI", "/skills/web-development", "/locators/docker-image"}
		default:
			return []string{}
		}
	}

	t.Run("complex_and_logic_test", func(t *testing.T) {
		queries := []*routingv1.RecordQuery{
			{
				Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL,
				Value: "AI",
			},
			{
				Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL,
				Value: "web-development",
			},
		}

		// Only mixed-record should match both queries
		assert.True(t, MatchesAllQueries(ctx, "mixed-record", queries, complexLabelRetriever))
		assert.False(t, MatchesAllQueries(ctx, "ai-record", queries, complexLabelRetriever))
		assert.False(t, MatchesAllQueries(ctx, "web-record", queries, complexLabelRetriever))
	})

	t.Run("hierarchical_skill_matching", func(t *testing.T) {
		queries := []*routingv1.RecordQuery{
			{
				Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL,
				Value: "AI/ML",
			},
		}

		// Should match records with AI/ML or more specific skills
		assert.True(t, MatchesAllQueries(ctx, "ai-record", queries, complexLabelRetriever))
		assert.False(t, MatchesAllQueries(ctx, "web-record", queries, complexLabelRetriever))
		assert.False(t, MatchesAllQueries(ctx, "mixed-record", queries, complexLabelRetriever)) // Only has /skills/AI, not AI/ML
	})
}
