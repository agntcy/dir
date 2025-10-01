// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"context"
	"testing"

	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
)

func TestQueryMatchesLabels(t *testing.T) {
	testCases := []struct {
		name     string
		query    *routingv1.RecordQuery
		labels   []types.Label
		expected bool
	}{
		// Skill queries
		{
			name: "skill_exact_match",
			query: &routingv1.RecordQuery{
				Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL,
				Value: "AI",
			},
			labels:   []types.Label{types.Label("/skills/AI"), types.Label("/skills/web-development")},
			expected: true,
		},
		{
			name: "skill_prefix_match",
			query: &routingv1.RecordQuery{
				Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL,
				Value: "AI",
			},
			labels:   []types.Label{types.Label("/skills/AI/ML"), types.Label("/skills/web-development")},
			expected: true,
		},
		{
			name: "skill_no_match",
			query: &routingv1.RecordQuery{
				Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL,
				Value: "blockchain",
			},
			labels:   []types.Label{types.Label("/skills/AI"), types.Label("/skills/web-development")},
			expected: false,
		},
		{
			name: "skill_partial_no_match",
			query: &routingv1.RecordQuery{
				Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL,
				Value: "AI/ML/deep-learning",
			},
			labels:   []types.Label{types.Label("/skills/AI/ML"), types.Label("/skills/web-development")},
			expected: false,
		},

		// Locator queries
		{
			name: "locator_exact_match",
			query: &routingv1.RecordQuery{
				Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_LOCATOR,
				Value: "docker-image",
			},
			labels:   []types.Label{types.Label("/locators/docker-image"), types.Label("/skills/AI")},
			expected: true,
		},
		{
			name: "locator_no_match",
			query: &routingv1.RecordQuery{
				Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_LOCATOR,
				Value: "git-repo",
			},
			labels:   []types.Label{types.Label("/locators/docker-image"), types.Label("/skills/AI")},
			expected: false,
		},

		// Domain queries
		{
			name: "domain_exact_match",
			query: &routingv1.RecordQuery{
				Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_DOMAIN,
				Value: "healthcare",
			},
			labels:   []types.Label{types.Label("/domains/healthcare"), types.Label("/skills/AI")},
			expected: true,
		},
		{
			name: "domain_prefix_match",
			query: &routingv1.RecordQuery{
				Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_DOMAIN,
				Value: "healthcare",
			},
			labels:   []types.Label{types.Label("/domains/healthcare/diagnostics"), types.Label("/skills/AI")},
			expected: true,
		},
		{
			name: "domain_no_match",
			query: &routingv1.RecordQuery{
				Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_DOMAIN,
				Value: "finance",
			},
			labels:   []types.Label{types.Label("/domains/healthcare"), types.Label("/skills/AI")},
			expected: false,
		},
		{
			name: "domain_partial_no_match",
			query: &routingv1.RecordQuery{
				Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_DOMAIN,
				Value: "healthcare/diagnostics/radiology",
			},
			labels:   []types.Label{types.Label("/domains/healthcare/diagnostics"), types.Label("/skills/AI")},
			expected: false,
		},

		// Module queries
		{
			name: "module_exact_match",
			query: &routingv1.RecordQuery{
				Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_MODULE,
				Value: "runtime/language",
			},
			labels:   []types.Label{types.Label("/modules/runtime/language"), types.Label("/skills/AI")},
			expected: true,
		},
		{
			name: "module_prefix_match",
			query: &routingv1.RecordQuery{
				Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_MODULE,
				Value: "runtime",
			},
			labels:   []types.Label{types.Label("/modules/runtime/language"), types.Label("/skills/AI")},
			expected: true,
		},
		{
			name: "module_no_match",
			query: &routingv1.RecordQuery{
				Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_MODULE,
				Value: "security",
			},
			labels:   []types.Label{types.Label("/modules/runtime/language"), types.Label("/skills/AI")},
			expected: false,
		},
		{
			name: "module_partial_no_match",
			query: &routingv1.RecordQuery{
				Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_MODULE,
				Value: "runtime/language/python/3.9",
			},
			labels:   []types.Label{types.Label("/modules/runtime/language"), types.Label("/skills/AI")},
			expected: false,
		},

		// Unspecified queries
		{
			name: "unspecified_always_matches",
			query: &routingv1.RecordQuery{
				Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_UNSPECIFIED,
				Value: "anything",
			},
			labels:   []types.Label{types.Label("/skills/AI")},
			expected: true,
		},
		{
			name: "unspecified_matches_empty_labels",
			query: &routingv1.RecordQuery{
				Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_UNSPECIFIED,
				Value: "anything",
			},
			labels:   []types.Label{},
			expected: true,
		},

		// Edge cases
		{
			name: "empty_labels",
			query: &routingv1.RecordQuery{
				Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL,
				Value: "AI",
			},
			labels:   []types.Label{},
			expected: false,
		},
		{
			name: "case_sensitive_skill",
			query: &routingv1.RecordQuery{
				Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL,
				Value: "ai", // lowercase
			},
			labels:   []types.Label{types.Label("/skills/AI")}, // uppercase
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
	mockLabelRetriever := func(_ context.Context, cid string) []types.Label {
		if cid == testCID {
			return []types.Label{
				types.Label("/skills/AI"),
				types.Label("/skills/AI/ML"),
				types.Label("/domains/technology"),
				types.Label("/modules/runtime/language"),
				types.Label("/locators/docker-image"),
			}
		}

		return []types.Label{}
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
			name: "domain_query_matches",
			cid:  testCID,
			queries: []*routingv1.RecordQuery{
				{
					Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_DOMAIN,
					Value: "technology",
				},
			},
			expected: true,
		},
		{
			name: "module_query_matches",
			cid:  testCID,
			queries: []*routingv1.RecordQuery{
				{
					Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_MODULE,
					Value: "runtime/language",
				},
			},
			expected: true,
		},
		{
			name: "all_query_types_match",
			cid:  testCID,
			queries: []*routingv1.RecordQuery{
				{
					Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL,
					Value: "AI",
				},
				{
					Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_DOMAIN,
					Value: "technology",
				},
				{
					Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_MODULE,
					Value: "runtime/language",
				},
				{
					Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_LOCATOR,
					Value: "docker-image",
				},
			},
			expected: true, // All should match
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
			Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_DOMAIN,
			Value: "healthcare",
		},
		{
			Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_MODULE,
			Value: "runtime/language",
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
			name:              "domain_matches",
			labelKey:          "/domains/healthcare/bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi/peer1",
			expectedMatches:   1,
			expectedQueryType: routingv1.RecordQueryType_RECORD_QUERY_TYPE_DOMAIN,
		},
		{
			name:              "module_matches",
			labelKey:          "/modules/runtime/language/bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi/peer1",
			expectedMatches:   1,
			expectedQueryType: routingv1.RecordQueryType_RECORD_QUERY_TYPE_MODULE,
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
		result := QueryMatchesLabels(nil, []types.Label{types.Label("/skills/AI")})
		assert.False(t, result)
	})

	t.Run("unknown_query_type", func(t *testing.T) {
		query := &routingv1.RecordQuery{
			Type:  routingv1.RecordQueryType(999), // Unknown type
			Value: "test",
		}
		result := QueryMatchesLabels(query, []types.Label{types.Label("/skills/AI")})
		assert.False(t, result)
	})

	t.Run("empty_query_value", func(t *testing.T) {
		query := &routingv1.RecordQuery{
			Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL,
			Value: "",
		}
		result := QueryMatchesLabels(query, []types.Label{types.Label("/skills/")})
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
	complexLabelRetriever := func(_ context.Context, cid string) []types.Label {
		switch cid {
		case "ai-record":
			return []types.Label{
				types.Label("/skills/AI"),
				types.Label("/skills/AI/ML"),
				types.Label("/skills/AI/NLP"),
			}
		case "web-record":
			return []types.Label{
				types.Label("/skills/web-development"),
				types.Label("/skills/javascript"),
				types.Label("/locators/git-repo"),
			}
		case "mixed-record":
			return []types.Label{
				types.Label("/skills/AI"),
				types.Label("/skills/web-development"),
				types.Label("/domains/healthcare"),
				types.Label("/modules/runtime/language"),
				types.Label("/locators/docker-image"),
			}
		default:
			return []types.Label{}
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
