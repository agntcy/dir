// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/agntcy/dir/server/datastore"
	"github.com/agntcy/dir/server/types"
	ipfsdatastore "github.com/ipfs/go-datastore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test the core Search functionality using a simplified approach.
func TestSearch_CoreLogic(t *testing.T) {
	ctx := t.Context()

	// Create test datastore
	dstore, cleanup := setupSearchTestDatastore(t)
	defer cleanup()

	// Setup test data - simulate remote announcements from different peers
	testData := []struct {
		cid    string
		peerID string
		labels []string
	}{
		{
			cid:    "ai-record-1",
			peerID: "remote-peer-1",
			labels: []string{"/skills/AI", "/skills/AI/ML"},
		},
		{
			cid:    "ai-record-2",
			peerID: "remote-peer-2",
			labels: []string{"/skills/AI/NLP"},
		},
		{
			cid:    "web-record",
			peerID: "remote-peer-3",
			labels: []string{"/skills/web-development", "/skills/javascript"},
		},
		{
			cid:    "local-record",
			peerID: testLocalPeerID, // This should be filtered out
			labels: []string{"/skills/AI"},
		},
	}

	// Store test label metadata
	for _, td := range testData {
		for _, label := range td.labels {
			enhancedKey := BuildEnhancedLabelKey(types.Label(label), td.cid, td.peerID)
			metadata := &types.LabelMetadata{
				Timestamp: time.Now(),
				LastSeen:  time.Now(),
			}
			metadataBytes, err := json.Marshal(metadata)
			require.NoError(t, err)

			err = dstore.Put(ctx, ipfsdatastore.NewKey(enhancedKey), metadataBytes)
			require.NoError(t, err)
		}
	}

	t.Run("search_filters_remote_records_only", func(t *testing.T) {
		// Test that we can find remote records and filter out local ones
		localPeerID := testLocalPeerID

		// Simulate searching for AI skills
		queries := []*routingv1.RecordQuery{
			{
				Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL,
				Value: "AI",
			},
		}

		// Use our simplified search logic
		results := simulateSearch(ctx, dstore, localPeerID, queries, 10, 1)

		// Should return 2 remote records (ai-record-1, ai-record-2) but not local-record
		assert.Len(t, results, 2)

		expectedCIDs := []string{"ai-record-1", "ai-record-2"}

		foundCIDs := make(map[string]bool)
		for _, result := range results {
			foundCIDs[result.GetRecordRef().GetCid()] = true

			// Verify it's not from local peer
			assert.NotEqual(t, localPeerID, result.GetPeer().GetId())

			// Verify structure
			assert.NotNil(t, result.GetRecordRef())
			assert.NotNil(t, result.GetPeer())
			assert.Positive(t, result.GetMatchScore())
		}

		for _, expectedCID := range expectedCIDs {
			assert.True(t, foundCIDs[expectedCID], "Expected CID %s not found", expectedCID)
		}
	})

	t.Run("search_with_and_logic", func(t *testing.T) {
		// Test AND logic with multiple queries
		localPeerID := testLocalPeerID

		queries := []*routingv1.RecordQuery{
			{
				Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL,
				Value: "AI",
			},
			{
				Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL,
				Value: "AI/ML",
			},
		}

		results := simulateSearch(ctx, dstore, localPeerID, queries, 10, 2)

		// Only ai-record-1 should match both AI and AI/ML
		assert.Len(t, results, 1)
		assert.Equal(t, "ai-record-1", results[0].GetRecordRef().GetCid())
		assert.Equal(t, "remote-peer-1", results[0].GetPeer().GetId())
		assert.Equal(t, uint32(2), results[0].GetMatchScore())
	})

	t.Run("search_with_limit", func(t *testing.T) {
		// Test result limiting
		localPeerID := testLocalPeerID

		queries := []*routingv1.RecordQuery{
			{
				Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL,
				Value: "AI",
			},
		}

		results := simulateSearch(ctx, dstore, localPeerID, queries, 1, 1) // Limit to 1

		assert.Len(t, results, 1)
		assert.NotEqual(t, localPeerID, results[0].GetPeer().GetId())
	})

	t.Run("search_with_high_min_score", func(t *testing.T) {
		// Test minimum match score filtering
		localPeerID := testLocalPeerID

		queries := []*routingv1.RecordQuery{
			{
				Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL,
				Value: "AI",
			},
		}

		results := simulateSearch(ctx, dstore, localPeerID, queries, 10, 5) // Very high score

		assert.Empty(t, results) // No results should meet the high score requirement
	})

	t.Run("search_no_queries_returns_all_remote", func(t *testing.T) {
		// Test that no queries returns all remote records
		localPeerID := testLocalPeerID

		results := simulateSearch(ctx, dstore, localPeerID, []*routingv1.RecordQuery{}, 10, 0)

		// Should return 3 remote records (excluding local-peer)
		assert.Len(t, results, 3)

		for _, result := range results {
			assert.NotEqual(t, localPeerID, result.GetPeer().GetId())
		}
	})

	t.Run("search_with_different_local_peer_id", func(t *testing.T) {
		// Test with a different localPeerID to validate the filtering logic
		differentLocalPeer := "remote-peer-1" // This peer has records in our test data

		queries := []*routingv1.RecordQuery{
			{
				Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL,
				Value: "AI",
			},
		}

		results := simulateSearch(ctx, dstore, differentLocalPeer, queries, 10, 1)

		// Should return different results since "remote-peer-1" is now considered "local"
		// and should be filtered out. Should find records from other peers: remote-peer-2, remote-peer-3, local-peer
		assert.Len(t, results, 2) // ai-record-2 (remote-peer-2) + local-record (local-peer)

		foundPeers := make(map[string]bool)

		for _, result := range results {
			assert.NotEqual(t, differentLocalPeer, result.GetPeer().GetId())

			foundPeers[result.GetPeer().GetId()] = true
		}

		// Should contain records from peers other than "remote-peer-1"
		expectedPeers := []string{"remote-peer-2", testLocalPeerID}
		for _, expectedPeer := range expectedPeers {
			assert.True(t, foundPeers[expectedPeer], "Should find record from peer %s", expectedPeer)
		}
	})

	t.Run("search_validates_peer_filtering_logic", func(t *testing.T) {
		// Test that changing localPeerID actually changes which records are filtered
		queries := []*routingv1.RecordQuery{
			{
				Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL,
				Value: "AI",
			},
		}

		// Search with testLocalPeerID as local
		resultsA := simulateSearch(ctx, dstore, testLocalPeerID, queries, 10, 1)

		// Search with "remote-peer-1" as local
		resultsB := simulateSearch(ctx, dstore, "remote-peer-1", queries, 10, 1)

		// Results may have same count but should contain different peers
		// resultsA filters out testLocalPeerID, resultsB filters out "remote-peer-1"
		// Both should return 2 results, but from different peer combinations

		// Collect all peer IDs from each result set
		peersA := make(map[string]bool)
		for _, result := range resultsA {
			peersA[result.GetPeer().GetId()] = true
		}

		peersB := make(map[string]bool)
		for _, result := range resultsB {
			peersB[result.GetPeer().GetId()] = true
		}

		// The peer sets should be different (different peers filtered out)
		assert.NotEqual(t, peersA, peersB, "Different localPeerID should result in different peer sets")
	})
}

// Simplified search simulation for testing.
//
//nolint:gocognit // Test helper function that replicates search logic - complexity is necessary
func simulateSearch(ctx context.Context, dstore types.Datastore, localPeerID string, queries []*routingv1.RecordQuery, limit uint32, minMatchScore uint32) []*routingv1.SearchResponse {
	var results []*routingv1.SearchResponse

	processedCIDs := make(map[string]bool)
	processedCount := 0
	limitInt := int(limit)

	// Query all namespaces using shared function
	entries, err := QueryAllNamespaces(ctx, dstore)
	if err != nil {
		return results
	}

	for _, entry := range entries {
		if limitInt > 0 && processedCount >= limitInt {
			break
		}

		// Parse enhanced key
		_, keyCID, keyPeerID, err := ParseEnhancedLabelKey(entry.Key)
		if err != nil {
			continue
		}

		// Filter for REMOTE records only
		if keyPeerID == localPeerID {
			continue
		}

		// Avoid duplicates
		if processedCIDs[keyCID] {
			continue
		}

		// Check if matches all queries
		if testMatchesAllQueriesSimple(ctx, dstore, keyCID, queries, keyPeerID) {
			// Calculate score safely
			score := safeIntToUint32(len(queries))
			if len(queries) == 0 {
				score = 1
			}

			if score >= minMatchScore {
				results = append(results, &routingv1.SearchResponse{
					RecordRef:    &corev1.RecordRef{Cid: keyCID},
					Peer:         &routingv1.Peer{Id: keyPeerID},
					MatchQueries: queries,
					MatchScore:   score,
				})

				processedCIDs[keyCID] = true
				processedCount++

				if limitInt > 0 && processedCount >= limitInt {
					break
				}
			}
		}
	}

	return results
}

// Simplified query matching for testing.
func testMatchesAllQueriesSimple(ctx context.Context, dstore types.Datastore, cid string, queries []*routingv1.RecordQuery, peerID string) bool {
	if len(queries) == 0 {
		return true
	}

	// Get labels for this CID/PeerID using shared namespace iteration
	entries, err := QueryAllNamespaces(ctx, dstore)
	if err != nil {
		return false
	}

	var labelStrings []string

	for _, entry := range entries {
		label, keyCID, keyPeerID, err := ParseEnhancedLabelKey(entry.Key)
		if err != nil {
			continue
		}

		if keyCID == cid && keyPeerID == peerID {
			labelStrings = append(labelStrings, label.String())
		}
	}

	// Use shared query matching logic - convert strings to labels
	labelRetriever := func(_ context.Context, _ string) []types.Label {
		labelList := make([]types.Label, len(labelStrings))
		for i, labelStr := range labelStrings {
			labelList[i] = types.Label(labelStr)
		}

		return labelList
	}

	return MatchesAllQueries(ctx, cid, queries, labelRetriever)
}

// Helper functions for testing

// setupSearchTestDatastore creates a temporary datastore for search testing.
func setupSearchTestDatastore(t *testing.T) (types.Datastore, func()) {
	t.Helper()

	dsOpts := []datastore.Option{
		datastore.WithFsProvider("/tmp/test-search-" + t.Name()),
	}

	dstore, err := datastore.New(dsOpts...)
	require.NoError(t, err)

	cleanup := func() {
		_ = dstore.Close()
		_ = os.RemoveAll("/tmp/test-search-" + t.Name())
	}

	return dstore, cleanup
}
