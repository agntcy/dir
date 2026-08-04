// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
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
				Value: "runtime/model",
			},
			labels:   []types.Label{types.Label("/modules/runtime/model"), types.Label("/skills/AI")},
			expected: true,
		},
		{
			name: "module_prefix_match",
			query: &routingv1.RecordQuery{
				Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_MODULE,
				Value: "runtime",
			},
			labels:   []types.Label{types.Label("/modules/runtime/model"), types.Label("/skills/AI")},
			expected: true,
		},
		{
			name: "module_no_match",
			query: &routingv1.RecordQuery{
				Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_MODULE,
				Value: "security",
			},
			labels:   []types.Label{types.Label("/modules/runtime/model"), types.Label("/skills/AI")},
			expected: false,
		},
		{
			name: "module_partial_no_match",
			query: &routingv1.RecordQuery{
				Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_MODULE,
				Value: "runtime/model/python/3.9",
			},
			labels:   []types.Label{types.Label("/modules/runtime/model"), types.Label("/skills/AI")},
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
