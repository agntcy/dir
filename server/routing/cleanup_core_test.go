// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/agntcy/dir/server/datastore"
	"github.com/agntcy/dir/server/types"
	ipfsdatastore "github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test the core cleanup logic without complex server dependencies.
func TestCleanup_CoreLogic(t *testing.T) {
	ctx := t.Context()

	dstore, cleanup := setupCleanupCoreTestDatastore(t)
	defer cleanup()

	t.Run("cleanup_labels_for_specific_cid", func(t *testing.T) {
		testCID := "test-cid-123"
		localPeerID := testLocalPeerID

		// Setup test data: record + labels
		recordKey := ipfsdatastore.NewKey("/records/" + testCID)
		err := dstore.Put(ctx, recordKey, []byte{})
		require.NoError(t, err)

		// Add labels for this CID (local and remote)
		testLabels := []struct {
			label           string
			peerID          string
			shouldBeDeleted bool
		}{
			{"/skills/AI", localPeerID, true},    // Local - should be deleted
			{"/skills/ML", localPeerID, true},    // Local - should be deleted
			{"/skills/AI", "remote-peer", false}, // Remote - should be kept
		}

		for _, tl := range testLabels {
			enhancedKey := BuildEnhancedLabelKey(types.Label(tl.label), testCID, tl.peerID)
			err = dstore.Put(ctx, ipfsdatastore.NewKey(enhancedKey), []byte("metadata"))
			require.NoError(t, err)
		}

		// Test cleanup logic
		success := simulateCleanupLabelsForCID(ctx, dstore, testCID, localPeerID)
		assert.True(t, success)

		// Verify record key was deleted
		exists, err := dstore.Has(ctx, recordKey)
		require.NoError(t, err)
		assert.False(t, exists)

		// Verify label cleanup
		for _, tl := range testLabels {
			enhancedKey := BuildEnhancedLabelKey(types.Label(tl.label), testCID, tl.peerID)
			exists, err := dstore.Has(ctx, ipfsdatastore.NewKey(enhancedKey))
			require.NoError(t, err)

			if tl.shouldBeDeleted {
				assert.False(t, exists, "Local label should be deleted")
			} else {
				assert.True(t, exists, "Remote label should be kept")
			}
		}
	})

	t.Run("stale_label_detection", func(t *testing.T) {
		// Test the core logic of stale label detection
		now := time.Now()

		testCases := []struct {
			name     string
			metadata *types.LabelMetadata
			isStale  bool
		}{
			{
				name: "fresh_label",
				metadata: &types.LabelMetadata{
					Timestamp: now.Add(-time.Hour),
					LastSeen:  now.Add(-time.Hour),
				},
				isStale: false,
			},
			{
				name: "stale_label",
				metadata: &types.LabelMetadata{
					Timestamp: now.Add(-MaxLabelAge - time.Hour),
					LastSeen:  now.Add(-MaxLabelAge - time.Hour),
				},
				isStale: true,
			},
			{
				name: "borderline_fresh",
				metadata: &types.LabelMetadata{
					Timestamp: now.Add(-MaxLabelAge + time.Minute),
					LastSeen:  now.Add(-MaxLabelAge + time.Minute),
				},
				isStale: false,
			},
			{
				name: "borderline_stale",
				metadata: &types.LabelMetadata{
					Timestamp: now.Add(-MaxLabelAge - time.Minute),
					LastSeen:  now.Add(-MaxLabelAge - time.Minute),
				},
				isStale: true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := tc.metadata.IsStale(MaxLabelAge)
				assert.Equal(t, tc.isStale, result)
			})
		}
	})

	t.Run("remote_label_filter_logic", func(t *testing.T) {
		// Test the remote label filtering logic
		localPeerID := testLocalPeerID

		testCases := []struct {
			name     string
			key      string
			expected bool // true if should be included (is remote)
		}{
			{
				name:     "remote_label",
				key:      "/skills/AI/bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi/remote-peer",
				expected: true,
			},
			{
				name:     "local_label",
				key:      "/skills/AI/bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi/" + testLocalPeerID,
				expected: false,
			},
			{
				name:     "malformed_key_treated_as_remote",
				key:      "/invalid-key",
				expected: true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Test the core filtering logic
				keyPeerID := ExtractPeerIDFromKey(tc.key)
				isRemote := (keyPeerID != localPeerID) || (keyPeerID == "")
				assert.Equal(t, tc.expected, isRemote)
			})
		}
	})

	t.Run("batch_deletion_efficiency", func(t *testing.T) {
		// Test that batch operations work correctly
		testCID := "batch-test-cid"
		localPeerID := testLocalPeerID

		// Setup multiple labels to delete
		labelsToDelete := []string{
			"/skills/AI",
			"/skills/ML",
			"/domains/tech",
			"/modules/nlp",
		}

		// Store record and labels
		recordKey := ipfsdatastore.NewKey("/records/" + testCID)
		err := dstore.Put(ctx, recordKey, []byte{})
		require.NoError(t, err)

		for _, label := range labelsToDelete {
			enhancedKey := BuildEnhancedLabelKey(types.Label(label), testCID, localPeerID)
			err = dstore.Put(ctx, ipfsdatastore.NewKey(enhancedKey), []byte("metadata"))
			require.NoError(t, err)
		}

		// Count keys before cleanup
		allResults, err := dstore.Query(ctx, query.Query{})
		require.NoError(t, err)

		var keysBefore []string
		for result := range allResults.Next() {
			keysBefore = append(keysBefore, result.Key)
		}

		allResults.Close()

		// Run cleanup
		success := simulateCleanupLabelsForCID(ctx, dstore, testCID, localPeerID)
		assert.True(t, success)

		// Count keys after cleanup
		allResults, err = dstore.Query(ctx, query.Query{})
		require.NoError(t, err)

		var keysAfter []string
		for result := range allResults.Next() {
			keysAfter = append(keysAfter, result.Key)
		}

		allResults.Close()

		// Should have deleted record + all labels (5 keys total)
		expectedDeleted := 1 + len(labelsToDelete) // 1 record + 4 labels
		actualDeleted := len(keysBefore) - len(keysAfter)
		assert.Equal(t, expectedDeleted, actualDeleted)
	})
}

// Simplified cleanup logic for testing (without server dependency).
func simulateCleanupLabelsForCID(ctx context.Context, dstore types.Datastore, cid string, localPeerID string) bool {
	batch, err := dstore.Batch(ctx)
	if err != nil {
		return false
	}

	keysDeleted := 0

	// Remove the /records/ key
	recordKey := ipfsdatastore.NewKey("/records/" + cid)
	if err := batch.Delete(ctx, recordKey); err == nil {
		keysDeleted++
	}

	// Find and remove all label keys for this CID using shared namespace iteration
	entries, err := QueryAllNamespaces(ctx, dstore)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		// Parse enhanced key
		_, keyCID, keyPeerID, err := ParseEnhancedLabelKey(entry.Key)
		if err != nil {
			// Delete malformed keys
			if err := batch.Delete(ctx, ipfsdatastore.NewKey(entry.Key)); err == nil {
				keysDeleted++
			}

			continue
		}

		// Check if this key matches our CID and is from local peer
		if keyCID == cid && keyPeerID == localPeerID {
			labelKey := ipfsdatastore.NewKey(entry.Key)
			if err := batch.Delete(ctx, labelKey); err == nil {
				keysDeleted++
			}
		}
	}

	// Commit the batch deletion
	if err := batch.Commit(ctx); err != nil {
		return false
	}

	return keysDeleted > 0
}

// Helper function for cleanup testing.
func setupCleanupCoreTestDatastore(t *testing.T) (types.Datastore, func()) {
	t.Helper()

	dsOpts := []datastore.Option{
		datastore.WithFsProvider("/tmp/test-cleanup-core-" + t.Name()),
	}

	dstore, err := datastore.New(dsOpts...)
	require.NoError(t, err)

	cleanup := func() {
		_ = dstore.Close()
		_ = os.RemoveAll("/tmp/test-cleanup-core-" + t.Name())
	}

	return dstore, cleanup
}
