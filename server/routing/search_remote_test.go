// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"testing"

	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/agntcy/dir/server/routing/rpc"
	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func skillQuery(value string) *routingv1.RecordQuery {
	return &routingv1.RecordQuery{Type: routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL, Value: value}
}

func TestScoreMatch(t *testing.T) {
	labels := []types.Label{
		types.Label("/skills/Natural Language Processing/Text Completion"),
		types.Label("/skills/Natural Language Processing/Problem Solving"),
		types.Label("/domains/healthcare"),
	}

	tests := []struct {
		name    string
		queries []*routingv1.RecordQuery
		score   uint32
	}{
		{
			name:    "counts only the queries that match",
			queries: []*routingv1.RecordQuery{skillQuery("Natural Language Processing/Text Completion"), skillQuery("Natural Language Processing/Problem Solving"), skillQuery("Nonexistent")},
			score:   2,
		},
		{
			name:    "single match",
			queries: []*routingv1.RecordQuery{skillQuery("Natural Language Processing/Text Completion")},
			score:   1,
		},
		{
			name:    "a parent skill matches its descendants, but counts once per query",
			queries: []*routingv1.RecordQuery{skillQuery("Natural Language Processing")},
			score:   1,
		},
		{
			name: "queries of different kinds both count",
			queries: []*routingv1.RecordQuery{
				skillQuery("Natural Language Processing"),
				{Type: routingv1.RecordQueryType_RECORD_QUERY_TYPE_DOMAIN, Value: "healthcare"},
			},
			score: 2,
		},
		{
			name:    "no match scores zero",
			queries: []*routingv1.RecordQuery{skillQuery("Nonexistent")},
			score:   0,
		},
		{
			name:    "no queries scores zero",
			queries: nil,
			score:   0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			matched, score := scoreMatch(test.queries, labels)

			assert.Equal(t, test.score, score)
			assert.Len(t, matched, int(test.score))
		})
	}
}

func TestScoreMatchWithoutLabels(t *testing.T) {
	matched, score := scoreMatch([]*routingv1.RecordQuery{skillQuery("AI")}, nil)

	assert.Empty(t, matched)
	assert.Equal(t, uint32(0), score)
}

func TestDiscoveryKeyPicksTheDeepestLabel(t *testing.T) {
	queries := []*routingv1.RecordQuery{
		skillQuery("AI"),
		skillQuery("AI/ML/Deep Learning"),
		skillQuery("AI/ML"),
	}

	key, label, ok := discoveryKey(queries)
	require.True(t, ok)
	assert.Equal(t, types.Label("/skills/AI/ML/Deep Learning"), label)

	expected, err := labelKey(types.Label("/skills/AI/ML/Deep Learning"))
	require.NoError(t, err)
	assert.Equal(t, expected, key)
}

func TestDiscoveryKeyMatchesTheKeyThePublisherAdvertises(t *testing.T) {
	// A record tagged /skills/AI/ML is only findable under /skills/AI because
	// its holder advertises that ancestor too, so the searcher's key for "AI"
	// has to be the same one expandLabel produces.
	advertised := expandLabel(types.Label("/skills/AI/ML"))
	require.Contains(t, advertised, types.Label("/skills/AI"))

	published, err := labelKey(types.Label("/skills/AI"))
	require.NoError(t, err)

	searched, _, ok := discoveryKey([]*routingv1.RecordQuery{skillQuery("AI")})
	require.True(t, ok)

	assert.Equal(t, published, searched)
}

func TestDiscoveryKeyRejectsQueriesWithNoLabel(t *testing.T) {
	tests := []struct {
		name    string
		queries []*routingv1.RecordQuery
	}{
		{name: "no queries", queries: nil},
		{name: "empty value", queries: []*routingv1.RecordQuery{skillQuery("   ")}},
		{
			name:    "unspecified type matches everything and names nothing",
			queries: []*routingv1.RecordQuery{{Type: routingv1.RecordQueryType_RECORD_QUERY_TYPE_UNSPECIFIED, Value: "AI"}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, ok := discoveryKey(test.queries)
			assert.False(t, ok)
		})
	}
}

func TestDiscoveryKeySkipsUnusableQueries(t *testing.T) {
	queries := []*routingv1.RecordQuery{
		{Type: routingv1.RecordQueryType_RECORD_QUERY_TYPE_UNSPECIFIED, Value: "anything"},
		skillQuery("AI"),
	}

	_, label, ok := discoveryKey(queries)
	require.True(t, ok)
	assert.Equal(t, types.Label("/skills/AI"), label)
}

func TestPeerQueries(t *testing.T) {
	queries := []*routingv1.RecordQuery{
		skillQuery("AI/ML"),
		{Type: routingv1.RecordQueryType_RECORD_QUERY_TYPE_DOMAIN, Value: "healthcare"},
		{Type: routingv1.RecordQueryType_RECORD_QUERY_TYPE_MODULE, Value: "runtime/model"},
		{Type: routingv1.RecordQueryType_RECORD_QUERY_TYPE_LOCATOR, Value: "docker-image"},
		{Type: routingv1.RecordQueryType_RECORD_QUERY_TYPE_UNSPECIFIED, Value: "dropped"},
		skillQuery(""),
	}

	assert.Equal(t, []rpc.RecordQuery{
		{Type: "skills", Value: "AI/ML"},
		{Type: "domains", Value: "healthcare"},
		{Type: "modules", Value: "runtime/model"},
		{Type: "locators", Value: "docker-image"},
	}, peerQueries(queries))
}

func TestPeerLimit(t *testing.T) {
	// Every record a peer returns matched at least one query, so at the default
	// threshold the caller keeps all of them and can ask for exactly its limit.
	assert.Equal(t, uint32(10), peerLimit(10, DefaultMinMatchScore))

	// A higher threshold is applied by the caller, so the peer has to offer more
	// candidates than the caller will keep.
	assert.Equal(t, uint32(0), peerLimit(10, 2))
}
