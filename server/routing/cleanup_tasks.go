// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"sync"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/server/routing/internal/p2p"
	"github.com/agntcy/dir/server/routing/pubsub"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/server/types/adapters"
	"github.com/agntcy/dir/utils/logging"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
)

var cleanupLogger = logging.Logger("routing/cleanup")

// remoteLabelFilter identifies remote labels by checking if they lack a corresponding local record.
// Remote labels are those that don't have a matching "/records/CID" key in the datastore.
//
//nolint:containedctx
type remoteLabelFilter struct {
	dstore      types.Datastore
	ctx         context.Context
	localPeerID string
}

func (f *remoteLabelFilter) Filter(e query.Entry) bool {
	// With enhanced keys, we can check PeerID directly from the key
	// Key format: /skills/AI/CID123/Peer1
	keyPeerID := ExtractPeerIDFromKey(e.Key)
	if keyPeerID == "" {
		// Invalid key format, assume remote to be safe
		return true
	}

	// It's remote if the PeerID in the key is not our local peer
	return keyPeerID != f.localPeerID
}

// CleanupManager handles all background cleanup and republishing tasks for the routing system.
// This includes CID provider republishing, GossipSub label republishing, stale remote label cleanup, and orphaned record cleanup.
type CleanupManager struct {
	dstore      types.Datastore
	storeAPI    types.StoreAPI
	server      *p2p.Server
	publishFunc pubsub.PublishEventHandler // Publishing callback (captures routeRemote state)
}

// NewCleanupManager creates a new cleanup manager with the required dependencies.
// The publishFunc is injected from routeRemote.Publish to avoid circular dependencies
// while still providing access to DHT and GossipSub publishing logic.
//
// Parameters:
//   - dstore: Datastore for label storage
//   - storeAPI: Store API for record operations
//   - server: P2P server for DHT operations
//   - publishFunc: Callback for publishing (from routeRemote.Publish, see pubsub.PublishEventHandler)
func NewCleanupManager(
	dstore types.Datastore,
	storeAPI types.StoreAPI,
	server *p2p.Server,
	publishFunc pubsub.PublishEventHandler,
) *CleanupManager {
	return &CleanupManager{
		dstore:      dstore,
		storeAPI:    storeAPI,
		server:      server,
		publishFunc: publishFunc,
	}
}

// StartLabelRepublishTask starts a background task that periodically republishes local
// CID provider announcements to keep content discoverable (provider records expire after ProviderRecordTTL).
// The wg parameter is used to track this goroutine in the parent's WaitGroup.
func (c *CleanupManager) StartLabelRepublishTask(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(RepublishInterval)

	cleanupLogger.Info("Started CID provider republishing task", "interval", RepublishInterval)

	defer func() {
		ticker.Stop()
		wg.Done()
		cleanupLogger.Debug("CID provider republishing task stopped")
	}()

	for {
		select {
		case <-ctx.Done():
			cleanupLogger.Info("CID provider republishing task stopping (context cancelled)")

			return
		case <-ticker.C:
			c.republishLocalProviders(ctx)
		}
	}
}

// StartRemoteLabelCleanupTask starts a background task that periodically cleans up stale remote labels.
// This is critical for the pull-based architecture to remove cached labels from offline or deleted remote content.
// The wg parameter is used to track this goroutine in the parent's WaitGroup.
func (c *CleanupManager) StartRemoteLabelCleanupTask(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(CleanupInterval)

	cleanupLogger.Info("Starting remote label cleanup task", "interval", CleanupInterval)

	defer func() {
		ticker.Stop()
		wg.Done()
		cleanupLogger.Debug("Remote label cleanup task stopped")
	}()

	for {
		select {
		case <-ctx.Done():
			cleanupLogger.Info("Remote label cleanup task stopping (context cancelled)")

			return
		case <-ticker.C:
			if err := c.cleanupStaleRemoteLabels(ctx); err != nil {
				cleanupLogger.Error("Failed to cleanup stale remote labels", "error", err)
			}
		}
	}
}

// republishLocalProviders republishes all local CID provider announcements and labels
// to ensure they remain discoverable. This maintains both DHT provider records and
// GossipSub label announcements for optimal network propagation.
func (c *CleanupManager) republishLocalProviders(ctx context.Context) {
	cleanupLogger.Info("Starting CID provider and label republishing cycle")

	// Query all local records from the datastore
	results, err := c.dstore.Query(ctx, query.Query{
		Prefix: "/records/",
	})
	if err != nil {
		cleanupLogger.Error("Failed to query local records for republishing", "error", err)

		return
	}
	defer results.Close()

	republishedCount := 0
	labelRepublishedCount := 0
	errorCount := 0

	var orphanedCIDs []string

	for result := range results.Next() {
		if result.Error != nil {
			cleanupLogger.Warn("Error reading local record for republishing", "error", result.Error)

			continue
		}

		// Extract CID from record key: /records/CID123 → CID123
		cidStr := path.Base(result.Key)
		if cidStr == "" {
			continue
		}

		// Verify the record still exists in storage
		ref := &corev1.RecordRef{Cid: cidStr}

		_, err := c.storeAPI.Lookup(ctx, ref)
		if err != nil {
			cleanupLogger.Warn("Record no longer exists in storage, marking as orphaned", "cid", cidStr, "error", err)
			orphanedCIDs = append(orphanedCIDs, cidStr)
			errorCount++

			continue
		}

		// Pull the record from storage for republishing
		record, err := c.storeAPI.Pull(ctx, ref)
		if err != nil {
			cleanupLogger.Warn("Failed to pull record for republishing",
				"cid", cidStr,
				"error", err)

			errorCount++

			continue
		}

		// Wrap record with adapter for interface-based publishing
		adapter := adapters.NewRecordAdapter(record)

		// Use injected publishing function (handles both DHT and GossipSub)
		// This reuses routeRemote.Publish logic without circular dependency
		if err := c.publishFunc(ctx, adapter); err != nil {
			cleanupLogger.Warn("Failed to republish record to network",
				"cid", cidStr,
				"error", err)

			errorCount++

			continue
		}

		cleanupLogger.Debug("Successfully republished record to network", "cid", cidStr)

		republishedCount++
		labelRepublishedCount++ // Count label republishing (done inside publishFunc)
	}

	// Clean up orphaned local records and their labels
	if len(orphanedCIDs) > 0 {
		cleanedCount := c.cleanupOrphanedLocalLabels(ctx, orphanedCIDs)
		cleanupLogger.Info("Cleaned up orphaned local records", "count", cleanedCount)
	}

	cleanupLogger.Info("Completed republishing cycle",
		"dhtRepublished", republishedCount,
		"gossipSubRepublished", labelRepublishedCount,
		"errors", errorCount,
		"orphaned", len(orphanedCIDs))
}

// cleanupStaleRemoteLabels removes remote labels that haven't been seen recently.
func (c *CleanupManager) cleanupStaleRemoteLabels(ctx context.Context) error {
	localPeerID := c.server.Host().ID().String()

	cleanupLogger.Debug("Starting stale remote label cleanup")

	// Query all label keys with remote filter
	// We'll query each namespace separately and combine results
	var allResults []query.Result

	for _, namespace := range types.AllLabelTypes() {
		nsResults, err := c.dstore.Query(ctx, query.Query{
			Prefix: namespace.Prefix(),
			Filters: []query.Filter{
				&remoteLabelFilter{
					dstore:      c.dstore,
					ctx:         ctx,
					localPeerID: localPeerID,
				},
			},
		})
		if err != nil {
			cleanupLogger.Warn("Failed to query namespace", "namespace", namespace, "error", err)

			continue
		}

		// Collect results from this namespace
		for result := range nsResults.Next() {
			allResults = append(allResults, result)
		}

		nsResults.Close()
	}

	var staleKeys []datastore.Key

	// Check each remote label for staleness
	for _, result := range allResults {
		if result.Error != nil {
			cleanupLogger.Warn("Error reading label entry", "key", result.Key, "error", result.Error)

			continue
		}

		// Parse enhanced key to get peer information
		_, _, keyPeerID, err := ParseEnhancedLabelKey(result.Key)
		if err != nil {
			cleanupLogger.Warn("Failed to parse enhanced label key, marking for deletion",
				"key", result.Key, "error", err)

			staleKeys = append(staleKeys, datastore.NewKey(result.Key))

			continue
		}

		var metadata types.LabelMetadata
		if err := json.Unmarshal(result.Value, &metadata); err != nil {
			cleanupLogger.Warn("Failed to parse label metadata, marking for deletion",
				"key", result.Key, "error", err)

			staleKeys = append(staleKeys, datastore.NewKey(result.Key))

			continue
		}

		// Validate metadata before checking staleness
		if err := metadata.Validate(); err != nil {
			cleanupLogger.Warn("Invalid label metadata found during cleanup, marking for deletion",
				"key", result.Key, "error", err)

			staleKeys = append(staleKeys, datastore.NewKey(result.Key))

			continue
		}

		// Check if label is stale using the IsStale method
		if metadata.IsStale(MaxLabelAge) {
			cleanupLogger.Debug("Found stale remote label",
				"key", result.Key, "age", metadata.Age(), "peer", keyPeerID)

			staleKeys = append(staleKeys, datastore.NewKey(result.Key))
		}
	}

	// Delete stale labels in batch
	if len(staleKeys) > 0 {
		batch, err := c.dstore.Batch(ctx)
		if err != nil {
			return fmt.Errorf("failed to create batch for cleanup: %w", err)
		}

		for _, key := range staleKeys {
			if err := batch.Delete(ctx, key); err != nil {
				cleanupLogger.Warn("Failed to delete stale label", "key", key.String(), "error", err)
			}
		}

		if err := batch.Commit(ctx); err != nil {
			return fmt.Errorf("failed to commit stale label cleanup: %w", err)
		}

		cleanupLogger.Info("Cleaned up stale remote labels", "count", len(staleKeys))
	} else {
		cleanupLogger.Debug("No stale remote labels found")
	}

	return nil
}

// cleanupOrphanedLocalLabels removes local records and labels for CIDs that no longer exist in storage.
func (c *CleanupManager) cleanupOrphanedLocalLabels(ctx context.Context, orphanedCIDs []string) int {
	cleanedCount := 0

	for _, cid := range orphanedCIDs {
		if c.cleanupLabelsForCID(ctx, cid) {
			cleanedCount++
		}
	}

	return cleanedCount
}

// cleanupLabelsForCID removes all local records and labels associated with a specific CID.
func (c *CleanupManager) cleanupLabelsForCID(ctx context.Context, cid string) bool {
	batch, err := c.dstore.Batch(ctx)
	if err != nil {
		cleanupLogger.Error("Failed to create cleanup batch", "cid", cid, "error", err)

		return false
	}

	keysDeleted := 0

	// Remove the /records/ key
	recordKey := datastore.NewKey("/records/" + cid)
	if err := batch.Delete(ctx, recordKey); err != nil {
		cleanupLogger.Warn("Failed to delete record key", "key", recordKey.String(), "error", err)
	} else {
		keysDeleted++
	}

	// Find and remove all label keys for this CID across all namespaces
	localPeerID := c.server.Host().ID().String()

	for _, namespace := range types.AllLabelTypes() {
		// Query labels in this namespace that match our CID
		labelResults, err := c.dstore.Query(ctx, query.Query{
			Prefix: namespace.Prefix(),
		})
		if err != nil {
			cleanupLogger.Warn("Failed to query labels for cleanup", "namespace", namespace, "cid", cid, "error", err)

			continue
		}

		defer labelResults.Close()

		for result := range labelResults.Next() {
			// Parse enhanced key to get CID and PeerID
			_, keyCID, keyPeerID, err := ParseEnhancedLabelKey(result.Key)
			if err != nil {
				cleanupLogger.Warn("Failed to parse enhanced label key during cleanup, deleting",
					"key", result.Key, "error", err)
				// Delete malformed keys
				if err := batch.Delete(ctx, datastore.NewKey(result.Key)); err == nil {
					keysDeleted++
				}

				continue
			}

			// Check if this key matches our CID and is from local peer
			if keyCID == cid && keyPeerID == localPeerID {
				// Delete this local label
				labelKey := datastore.NewKey(result.Key)
				if err := batch.Delete(ctx, labelKey); err != nil {
					cleanupLogger.Warn("Failed to delete label key", "key", labelKey.String(), "error", err)
				} else {
					keysDeleted++

					cleanupLogger.Debug("Scheduled orphaned label for deletion", "key", result.Key)
				}
			}
		}
	}

	// Commit the batch deletion
	if err := batch.Commit(ctx); err != nil {
		cleanupLogger.Error("Failed to commit orphaned label cleanup", "cid", cid, "error", err)

		return false
	}

	if keysDeleted > 0 {
		cleanupLogger.Debug("Successfully cleaned up orphaned labels", "cid", cid, "keysDeleted", keysDeleted)
	}

	return keysDeleted > 0
}
