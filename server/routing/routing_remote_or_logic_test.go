// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"encoding/json"
	"testing"
	"time"

	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/agntcy/dir/server/datastore"
	"github.com/agntcy/dir/server/types"
	ipfsdatastore "github.com/ipfs/go-datastore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This test bypasses DHT infrastructure issues and directly tests the calculateMatchScore method.
func TestRemoteSearch_ORLogicWithMinMatchScore(t *testing.T) {
	ctx := t.Context()

	// Create test datastore
	dstore, cleanup := setupTestDatastore(t)
	defer cleanup()

	// Create routeRemote instance for testing
	r := &routeRemote{
		dstore: dstore,
	}

	// Setup test scenario: simulate cached remote announcements
	testPeerID := "remote-peer-test"
	testCID := "test-record-cid"

	// Simulate Peer 1 announced these skills for the test record
	skillLabels := []string{
		"/skills/Natural Language Processing/Text Completion",
		"/skills/Natural Language Processing/Problem Solving",
	}

	// Store enhanced label announcements in datastore (simulating DHT cache)
	for _, label := range skillLabels {
		enhancedKey := BuildEnhancedLabelKey(types.Label(label), testCID, testPeerID)
		metadata := &types.LabelMetadata{
			Timestamp: time.Now(),
			LastSeen:  time.Now(),
		}
		metadataBytes, err := json.Marshal(metadata)
		require.NoError(t, err)

		err = dstore.Put(ctx, ipfsdatastore.NewKey(enhancedKey), metadataBytes)
		require.NoError(t, err)
	}

	t.Run("OR Logic Success - 2/3 queries match", func(t *testing.T) {
		// Test queries: 2 real skills + 1 fake skill
		queries := []*routingv1.RecordQuery{
			{Type: routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL, Value: "Natural Language Processing/Text Completion"},
			{Type: routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL, Value: "Natural Language Processing/Problem Solving"},
			{Type: routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL, Value: "NonexistentSkill"},
		}

		// Test calculateMatchScore directly (avoids server dependency)
		matchQueries, score := r.calculateMatchScore(ctx, testCID, queries, testPeerID)

		// Should have 2 matching queries out of 3
		assert.Len(t, matchQueries, 2, "Should have 2 matching queries")
		assert.Equal(t, uint32(2), score, "Score should be 2 (2 out of 3 queries matched)")

		// Test that minScore=2 would include this record
		assert.GreaterOrEqual(t, score, uint32(2), "Score meets minScore=2 threshold")

		// Test that minScore=3 would exclude this record
		assert.Less(t, score, uint32(3), "Score does not meet minScore=3 threshold")
	})

	t.Run("Single Query Match", func(t *testing.T) {
		// Single query that should match
		queries := []*routingv1.RecordQuery{
			{Type: routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL, Value: "Natural Language Processing/Text Completion"},
		}

		// Test calculateMatchScore
		matchQueries, score := r.calculateMatchScore(ctx, testCID, queries, testPeerID)

		// Should have 1 matching query
		assert.Len(t, matchQueries, 1, "Should have 1 matching query")
		assert.Equal(t, uint32(1), score, "Score should be 1")
	})

	t.Run("Perfect Match - 2/2 queries match", func(t *testing.T) {
		// Two queries that should both match
		queries := []*routingv1.RecordQuery{
			{Type: routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL, Value: "Natural Language Processing/Text Completion"},
			{Type: routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL, Value: "Natural Language Processing/Problem Solving"},
		}

		// Test calculateMatchScore
		matchQueries, score := r.calculateMatchScore(ctx, testCID, queries, testPeerID)

		// Should have 2 matching queries out of 2
		assert.Len(t, matchQueries, 2, "Should have 2 matching queries")
		assert.Equal(t, uint32(2), score, "Score should be 2 (both queries matched)")
	})

	t.Run("No Queries Match", func(t *testing.T) {
		// Query that doesn't match anything
		queries := []*routingv1.RecordQuery{
			{Type: routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL, Value: "NonexistentSkill"},
		}

		// Test calculateMatchScore
		matchQueries, score := r.calculateMatchScore(ctx, testCID, queries, testPeerID)

		// Should have 0 matching queries
		assert.Empty(t, matchQueries, "Should have 0 matching queries")
		assert.Equal(t, uint32(0), score, "Score should be 0")
	})

	t.Run("Empty Queries", func(t *testing.T) {
		// No queries
		var queries []*routingv1.RecordQuery

		// Test calculateMatchScore
		matchQueries, score := r.calculateMatchScore(ctx, testCID, queries, testPeerID)

		// Should have 0 matching queries and 0 score
		assert.Empty(t, matchQueries, "Should have 0 matching queries with empty query list")
		assert.Equal(t, uint32(0), score, "Score should be 0 with empty queries")
	})

	t.Run("Hierarchical Skill Matching", func(t *testing.T) {
		// Test hierarchical skill matching (prefix matching)
		queries := []*routingv1.RecordQuery{
			{Type: routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL, Value: "Natural Language Processing"}, // Should match both skills via prefix
		}

		// Test calculateMatchScore
		matchQueries, score := r.calculateMatchScore(ctx, testCID, queries, testPeerID)

		// Should match at least 1 query (hierarchical matching)
		assert.GreaterOrEqual(t, len(matchQueries), 1, "Should have at least 1 matching query with hierarchical matching")
		assert.GreaterOrEqual(t, score, uint32(1), "Score should be at least 1 with hierarchical matching")
	})
}

// setupTestDatastore creates a test datastore for routing tests.
func setupTestDatastore(t *testing.T) (types.Datastore, func()) {
	t.Helper()

	dstore, err := datastore.New()
	require.NoError(t, err)

	cleanup := func() {
		if closer, ok := dstore.(interface{ Close() error }); ok {
			closer.Close()
		}
	}

	return dstore, cleanup
}
