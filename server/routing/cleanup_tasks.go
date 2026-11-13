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

// ProvideFunc announces a local CID to the DHT.
type ProvideFunc func(ctx context.Context, cid string) error

// CleanupManager handles the background tasks of the routing system: DHT reprovide,
// GossipSub reprovide, stale remote label cleanup and orphaned record cleanup.
type CleanupManager struct {
	dstore                  types.Datastore
	storeAPI                types.StoreAPI
	server                  *p2p.Server
	provideFunc             ProvideFunc                // DHT announcement (captures routeRemote state)
	announceFunc            pubsub.PublishEventHandler // GossipSub announcement (captures routeRemote state)
	dhtReprovideInterval    time.Duration              // How often local CIDs are re-announced to the DHT
	gossipReprovideInterval time.Duration              // How often local records are re-announced over GossipSub
	cleanupInterval         time.Duration              // How often stale remote labels are removed
	maxLabelAge             time.Duration              // Age after which a remote label is stale
}

// NewCleanupManager creates a new cleanup manager with the required dependencies.
// The intervals come from params, which already has its defaults applied.
func NewCleanupManager(
	dstore types.Datastore,
	storeAPI types.StoreAPI,
	server *p2p.Server,
	provideFunc ProvideFunc,
	announceFunc pubsub.PublishEventHandler,
	params dhtParams,
) *CleanupManager {
	return &CleanupManager{
		dstore:                  dstore,
		storeAPI:                storeAPI,
		server:                  server,
		provideFunc:             provideFunc,
		announceFunc:            announceFunc,
		dhtReprovideInterval:    params.DHTReprovideInterval,
		gossipReprovideInterval: params.GossipReprovideInterval,
		cleanupInterval:         params.CleanupInterval,
		maxLabelAge:             params.MaxLabelAge,
	}
}

// StartDHTReprovideTask periodically re-announces local CIDs to the DHT so their
// provider records do not expire on remote peers.
// The wg parameter is used to track this goroutine in the parent's WaitGroup.
func (c *CleanupManager) StartDHTReprovideTask(ctx context.Context, wg *sync.WaitGroup) {
	runEvery(ctx, wg, "DHT reprovide", c.dhtReprovideInterval, c.reprovideDHT)
}

// StartGossipReprovideTask periodically re-announces local records over GossipSub.
// The wg parameter is used to track this goroutine in the parent's WaitGroup.
func (c *CleanupManager) StartGossipReprovideTask(ctx context.Context, wg *sync.WaitGroup) {
	runEvery(ctx, wg, "GossipSub reprovide", c.gossipReprovideInterval, c.reprovideGossip)
}

// runEvery calls fn on every tick of interval until ctx is cancelled.
func runEvery(ctx context.Context, wg *sync.WaitGroup, name string, interval time.Duration, fn func(context.Context)) {
	ticker := time.NewTicker(interval)

	cleanupLogger.Info("Started task", "task", name, "interval", interval)

	defer func() {
		ticker.Stop()
		wg.Done()
		cleanupLogger.Debug("Task stopped", "task", name)
	}()

	for {
		select {
		case <-ctx.Done():
			cleanupLogger.Info("Task stopping (context cancelled)", "task", name)

			return
		case <-ticker.C:
			fn(ctx)
		}
	}
}

// StartRemoteLabelCleanupTask starts a background task that periodically cleans up stale remote labels.
// This is critical for the pull-based architecture to remove cached labels from offline or deleted remote content.
// The wg parameter is used to track this goroutine in the parent's WaitGroup.
func (c *CleanupManager) StartRemoteLabelCleanupTask(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(c.cleanupInterval)

	cleanupLogger.Info("Starting remote label cleanup task", "interval", c.cleanupInterval)

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

// localRecordCIDs returns the CIDs of the records published by this node.
func (c *CleanupManager) localRecordCIDs(ctx context.Context) ([]string, error) {
	results, err := c.dstore.Query(ctx, query.Query{Prefix: "/records/", KeysOnly: true})
	if err != nil {
		return nil, fmt.Errorf("failed to query local records: %w", err)
	}
	defer results.Close()

	var cids []string

	for result := range results.Next() {
		if result.Error != nil {
			cleanupLogger.Warn("Error reading local record", "error", result.Error)

			continue
		}

		// Record key: /records/CID123 → CID123
		if cid := path.Base(result.Key); cid != "" {
			cids = append(cids, cid)
		}
	}

	return cids, nil
}

// reprovideDHT re-announces every local CID to the DHT. It only needs the CID,
// so records are not pulled from storage. Records that no longer exist in
// storage are treated as orphaned and their local labels are removed.
func (c *CleanupManager) reprovideDHT(ctx context.Context) {
	cids, err := c.localRecordCIDs(ctx)
	if err != nil {
		cleanupLogger.Error("Failed to list local records for DHT reprovide", "error", err)

		return
	}

	var (
		provided, failed int
		orphanedCIDs     []string
	)

	for _, cidStr := range cids {
		if _, err := c.storeAPI.Lookup(ctx, &corev1.RecordRef{Cid: cidStr}); err != nil {
			cleanupLogger.Warn("Record no longer exists in storage, marking as orphaned", "cid", cidStr, "error", err)

			orphanedCIDs = append(orphanedCIDs, cidStr)

			continue
		}

		if err := c.provideFunc(ctx, cidStr); err != nil {
			cleanupLogger.Warn("Failed to reprovide CID to the DHT", "cid", cidStr, "error", err)

			failed++

			continue
		}

		provided++
	}

	if len(orphanedCIDs) > 0 {
		cleanedCount := c.cleanupOrphanedLocalLabels(ctx, orphanedCIDs)
		cleanupLogger.Info("Cleaned up orphaned local records", "count", cleanedCount)
	}

	cleanupLogger.Info("Completed DHT reprovide cycle",
		"provided", provided,
		"errors", failed,
		"orphaned", len(orphanedCIDs))
}

// reprovideGossip re-announces every local record over GossipSub. Unlike the DHT,
// the announcement carries the record labels, so each record is pulled from storage.
// Orphaned records are left to the DHT reprovide task.
func (c *CleanupManager) reprovideGossip(ctx context.Context) {
	cids, err := c.localRecordCIDs(ctx)
	if err != nil {
		cleanupLogger.Error("Failed to list local records for GossipSub reprovide", "error", err)

		return
	}

	var announced, failed int

	for _, cidStr := range cids {
		record, err := c.storeAPI.Pull(ctx, &corev1.RecordRef{Cid: cidStr})
		if err != nil {
			cleanupLogger.Warn("Failed to pull record for GossipSub reprovide", "cid", cidStr, "error", err)

			failed++

			continue
		}

		adapter, err := record.Decode()
		if err != nil {
			cleanupLogger.Warn("Failed to decode record for GossipSub reprovide", "cid", cidStr, "error", err)

			failed++

			continue
		}

		if err := c.announceFunc(ctx, adapter); err != nil {
			cleanupLogger.Warn("Failed to reprovide record over GossipSub", "cid", cidStr, "error", err)

			failed++

			continue
		}

		announced++
	}

	cleanupLogger.Info("Completed GossipSub reprovide cycle", "announced", announced, "errors", failed)
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
		if metadata.IsStale(c.maxLabelAge) {
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
