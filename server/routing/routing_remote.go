// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/server/routing/internal/p2p"
	"github.com/agntcy/dir/server/routing/rpc"
	validators "github.com/agntcy/dir/server/routing/validators"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/providers"
	record "github.com/libp2p/go-libp2p-record"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/protocol"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var remoteLogger = logging.Logger("routing/remote")

// this interface handles routing across the network.
// TODO: we shoud add caching here.
type routeRemote struct {
	storeAPI types.StoreAPI
	server   *p2p.Server
	service  *rpc.Service
	notifyCh chan *handlerSync
	dstore   types.Datastore
}

//nolint:mnd
func newRemote(ctx context.Context,
	storeAPI types.StoreAPI,
	dstore types.Datastore,
	opts types.APIOptions,
) (*routeRemote, error) {
	// Create routing
	routeAPI := &routeRemote{
		storeAPI: storeAPI,
		notifyCh: make(chan *handlerSync, NotificationChannelSize),
		dstore:   dstore,
	}

	// Determine refresh interval: use config override for testing, otherwise use default
	refreshInterval := RefreshInterval
	if opts.Config().Routing.RefreshInterval > 0 {
		refreshInterval = opts.Config().Routing.RefreshInterval
	}

	// Create P2P server
	server, err := p2p.New(ctx,
		p2p.WithListenAddress(opts.Config().Routing.ListenAddress),
		p2p.WithBootstrapAddrs(opts.Config().Routing.BootstrapPeers),
		p2p.WithRefreshInterval(refreshInterval),
		p2p.WithRandevous(ProtocolRendezvous), // enable libp2p auto-discovery
		p2p.WithIdentityKeyPath(opts.Config().Routing.KeyPath),
		p2p.WithCustomDHTOpts(
			func(h host.Host) ([]dht.Option, error) {
				// create provider manager
				providerMgr, err := providers.NewProviderManager(h.ID(), h.Peerstore(), dstore)
				if err != nil {
					return nil, fmt.Errorf("failed to create provider manager: %w", err)
				}

				// create custom validators for label namespaces
				labelValidators := validators.CreateLabelValidators()
				validator := record.NamespacedValidator{
					validators.NamespaceSkills.String():   labelValidators[validators.NamespaceSkills.String()],
					validators.NamespaceDomains.String():  labelValidators[validators.NamespaceDomains.String()],
					validators.NamespaceFeatures.String(): labelValidators[validators.NamespaceFeatures.String()],
				}

				// return custom opts for DHT
				return []dht.Option{
					dht.Datastore(dstore),                           // custom DHT datastore
					dht.ProtocolPrefix(protocol.ID(ProtocolPrefix)), // custom DHT protocol prefix
					dht.Validator(validator),                        // custom validators for label namespaces
					dht.MaxRecordAge(RecordTTL),                     // set consistent TTL for all DHT records
					dht.ProviderStore(&handler{
						ProviderManager: providerMgr,
						hostID:          h.ID().String(),
						notifyCh:        routeAPI.notifyCh,
					}),
				}, nil
			},
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create p2p: %w", err)
	}

	// update server pointers
	routeAPI.server = server

	// Register RPC server
	rpcService, err := rpc.New(server.Host(), storeAPI)
	if err != nil {
		defer server.Close()

		return nil, fmt.Errorf("failed to create RPC service: %w", err)
	}

	// update service
	routeAPI.service = rpcService

	// run listener in background
	go routeAPI.handleNotify(ctx)

	// run label republishing task in background
	go routeAPI.startLabelRepublishTask(ctx)

	// run remote label cleanup task in background
	go routeAPI.startRemoteLabelCleanupTask(ctx)

	return routeAPI, nil
}

func (r *routeRemote) Publish(ctx context.Context, ref *corev1.RecordRef, record *corev1.Record) error {
	remoteLogger.Debug("Called remote routing's Publish method for network operations", "ref", ref, "record", record)

	// Parse record CID
	decodedCID, err := cid.Decode(ref.GetCid())
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "failed to parse CID: %v", err)
	}

	// Announce CID to DHT network
	err = r.server.DHT().Provide(ctx, decodedCID, true)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to announce object %v: %v", ref.GetCid(), err)
	}

	// Announce all label mappings to DHT network
	labels := getLabels(record)
	for _, label := range labels {
		r.announceLabelToDHT(ctx, label, ref.GetCid())
	}

	remoteLogger.Debug("Successfully announced object and labels to network",
		"ref", ref, "labels", len(labels), "peers", r.server.DHT().RoutingTable().Size())

	return nil
}

// NOTE: List method removed from routeRemote
// List is a local-only operation that should never interact with the network
// Use Search for network-wide discovery instead

func (r *routeRemote) handleNotify(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// check if anything on notify
	for {
		select {
		case <-ctx.Done():
			return
		case notif := <-r.notifyCh:
			switch notif.AnnouncementType {
			case AnnouncementTypeLabel:
				r.handleLabelNotification(ctx, notif)
			case AnnouncementTypeCID:
				r.handleCIDProviderNotification(ctx, notif)
			default:
				// Backward compatibility: treat as CID announcement
				r.handleCIDProviderNotification(ctx, notif)
			}
		}
	}
}

// handleLabelNotification handles notifications for label announcements.
func (r *routeRemote) handleLabelNotification(ctx context.Context, notif *handlerSync) {
	remoteLogger.Info("Processing enhanced label announcement",
		"enhanced_key", notif.LabelKey, "cid", notif.Ref.GetCid(), "peer", notif.Peer.ID)

	now := time.Now()

	// The notif.LabelKey is already in enhanced format: /skills/AI/CID123/Peer1
	enhancedKey := datastore.NewKey(notif.LabelKey)

	// Check if we already have this exact label from this peer
	var metadata *LabelMetadata

	if existingData, err := r.dstore.Get(ctx, enhancedKey); err == nil {
		// Update existing metadata
		var existingMetadata LabelMetadata
		if err := json.Unmarshal(existingData, &existingMetadata); err == nil {
			existingMetadata.Update()
			metadata = &existingMetadata
		}
	}

	// Create new metadata if we couldn't update existing
	if metadata == nil {
		metadata = &LabelMetadata{
			Timestamp: now,
			LastSeen:  now,
		}
	}

	// Serialize metadata to JSON
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		remoteLogger.Error("Failed to serialize label metadata",
			"enhanced_key", notif.LabelKey, "error", err)

		return
	}

	// Store with the enhanced key directly
	err = r.dstore.Put(ctx, enhancedKey, metadataBytes)
	if err != nil {
		remoteLogger.Error("Failed to store remote label announcement",
			"enhanced_key", notif.LabelKey, "error", err)

		return
	}

	remoteLogger.Info("Successfully stored remote label announcement",
		"enhanced_key", notif.LabelKey, "peer", notif.Peer.ID)
}

// "I have this content", while label announcements indicate "this content has these labels".
func (r *routeRemote) handleCIDProviderNotification(ctx context.Context, notif *handlerSync) {
	// Check if we have this record locally (for comparison/validation)
	_, err := r.storeAPI.Lookup(ctx, notif.Ref)
	if err == nil {
		remoteLogger.Debug("Local copy exists, validating remote announcement consistency",
			"cid", notif.Ref.GetCid(), "peer", notif.Peer.ID)
	} else {
		remoteLogger.Debug("No local copy, validating remote content availability",
			"cid", notif.Ref.GetCid(), "peer", notif.Peer.ID)
	}

	// TODO: we should subscribe to some records so we can create a local copy
	// of the record and its skills.
	// for now, we are only testing if we can reach out and fetch it from the
	// broadcasting node

	// Validate that the announcing peer actually has the content they claim to provide
	// Step 1: Try to lookup metadata from the announcing peer
	_, err = r.service.Lookup(ctx, notif.Peer.ID, notif.Ref)
	if err != nil {
		remoteLogger.Error("Peer announced CID but failed metadata lookup",
			"peer", notif.Peer.ID, "cid", notif.Ref.GetCid(), "error", err)

		return
	}

	// Step 2: Try to actually fetch the content from the announcing peer
	_, err = r.service.Pull(ctx, notif.Peer.ID, notif.Ref)
	if err != nil {
		remoteLogger.Error("Peer announced CID but failed content delivery",
			"peer", notif.Peer.ID, "cid", notif.Ref.GetCid(), "error", err)

		return
	}

	// TODO: we can perform validation and data synchronization here.
	// Depending on the server configuration, we can decide if we want to
	// pull this model into our own cache, rebroadcast it, or ignore it.

	// MONITORING: Log successful content validation for network analytics
	remoteLogger.Info("Successfully validated announced content",
		"peer", notif.Peer.ID, "cid", notif.Ref.GetCid())
}

// label mappings to prevent them from expiring (DHT PutValue records expire after DHTRecordTTL).
func (r *routeRemote) startLabelRepublishTask(ctx context.Context) {
	ticker := time.NewTicker(RepublishInterval)
	defer ticker.Stop()

	remoteLogger.Info("Started label republishing task", "interval", RepublishInterval)

	for {
		select {
		case <-ctx.Done():
			remoteLogger.Info("Label republishing task stopped")

			return
		case <-ticker.C:
			r.republishLocalLabels(ctx)
		}
	}
}

// to ensure they remain discoverable in the DHT.
func (r *routeRemote) republishLocalLabels(ctx context.Context) {
	remoteLogger.Info("Starting label republishing cycle")

	// Query all local records from the datastore
	results, err := r.dstore.Query(ctx, query.Query{
		Prefix: "/records/",
	})
	if err != nil {
		remoteLogger.Error("Failed to query local records for republishing", "error", err)

		return
	}

	republishedCount := 0
	errorCount := 0

	var orphanedCIDs []string

	for result := range results.Next() {
		// Extract CID from record key: /records/CID123 → CID123
		cid := path.Base(result.Key)
		if cid == "" {
			continue
		}

		// Get the record to extract its labels
		ref := &corev1.RecordRef{Cid: cid}

		record, err := r.storeAPI.Pull(ctx, ref)
		if err != nil {
			remoteLogger.Warn("Failed to pull record for republishing, marking as orphaned", "cid", cid, "error", err)

			// Track this CID for cleanup - the record no longer exists in storage
			orphanedCIDs = append(orphanedCIDs, cid)
			errorCount++

			continue
		}

		// Republish all label mappings for this record using enhanced format
		labels := getLabels(record)
		localPeerID := r.server.Host().ID().String()

		for _, label := range labels {
			// Use enhanced self-descriptive DHT key format
			enhancedKey := BuildEnhancedLabelKey(label, cid, localPeerID)

			// Republish label mapping to DHT network
			err = r.server.DHT().PutValue(ctx, enhancedKey, []byte(cid))
			if err != nil {
				remoteLogger.Warn("Failed to republish enhanced label mapping", "enhanced_key", enhancedKey, "error", err)

				errorCount++
			} else {
				remoteLogger.Debug("Successfully republished enhanced label mapping", "enhanced_key", enhancedKey)

				republishedCount++
			}
		}
	}

	// Clean up orphaned local records and their labels
	if len(orphanedCIDs) > 0 {
		cleanedCount := r.cleanupOrphanedLocalLabels(ctx, orphanedCIDs)
		remoteLogger.Info("Cleaned up orphaned local records", "count", cleanedCount)
	}

	remoteLogger.Info("Completed label republishing cycle",
		"republished", republishedCount, "errors", errorCount, "orphaned", len(orphanedCIDs))
}

// announceLabelToDHT announces a label mapping to the DHT network using enhanced key format.
func (r *routeRemote) announceLabelToDHT(ctx context.Context, label, cidStr string) {
	// Get local peer ID for enhanced key
	localPeerID := r.server.Host().ID().String()

	// Announce to DHT network using enhanced self-descriptive key format
	enhancedKey := BuildEnhancedLabelKey(label, cidStr, localPeerID)
	err := r.server.DHT().PutValue(ctx, enhancedKey, []byte(cidStr))

	if err != nil {
		remoteLogger.Warn("Failed to announce enhanced label to DHT", "enhanced_key", enhancedKey, "error", err)
	} else {
		remoteLogger.Debug("Successfully announced enhanced label to DHT", "enhanced_key", enhancedKey)
	}
}

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

// cleanupStaleRemoteLabels removes remote labels that haven't been seen recently.
func (r *routeRemote) cleanupStaleRemoteLabels(ctx context.Context) error {
	localPeerID := r.server.Host().ID().String()

	remoteLogger.Debug("Starting stale remote label cleanup")

	// Query all label keys with remote filter
	// We'll query each namespace separately and combine results
	var allResults []query.Result

	for _, namespace := range validators.AllNamespaces() {
		nsResults, err := r.dstore.Query(ctx, query.Query{
			Prefix: namespace.Prefix(),
			Filters: []query.Filter{
				&remoteLabelFilter{
					dstore:      r.dstore,
					ctx:         ctx,
					localPeerID: localPeerID,
				},
			},
		})
		if err != nil {
			remoteLogger.Warn("Failed to query namespace", "namespace", namespace, "error", err)

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
			remoteLogger.Warn("Error reading label entry", "key", result.Key, "error", result.Error)

			continue
		}

		// Parse enhanced key to get peer information
		_, _, keyPeerID, err := ParseEnhancedLabelKey(result.Key)
		if err != nil {
			remoteLogger.Warn("Failed to parse enhanced label key, marking for deletion",
				"key", result.Key, "error", err)

			staleKeys = append(staleKeys, datastore.NewKey(result.Key))

			continue
		}

		var metadata LabelMetadata
		if err := json.Unmarshal(result.Value, &metadata); err != nil {
			remoteLogger.Warn("Failed to parse label metadata, marking for deletion",
				"key", result.Key, "error", err)

			staleKeys = append(staleKeys, datastore.NewKey(result.Key))

			continue
		}

		// Validate metadata before checking staleness
		if err := metadata.Validate(); err != nil {
			remoteLogger.Warn("Invalid label metadata found during cleanup, marking for deletion",
				"key", result.Key, "error", err)

			staleKeys = append(staleKeys, datastore.NewKey(result.Key))

			continue
		}

		// Check if label is stale using the IsStale method
		if metadata.IsStale(MaxLabelAge) {
			remoteLogger.Debug("Found stale remote label",
				"key", result.Key, "age", metadata.Age(), "peer", keyPeerID)

			staleKeys = append(staleKeys, datastore.NewKey(result.Key))
		}
	}

	// Delete stale labels in batch
	if len(staleKeys) > 0 {
		batch, err := r.dstore.Batch(ctx)
		if err != nil {
			return fmt.Errorf("failed to create batch for cleanup: %w", err)
		}

		for _, key := range staleKeys {
			if err := batch.Delete(ctx, key); err != nil {
				remoteLogger.Warn("Failed to delete stale label", "key", key.String(), "error", err)
			}
		}

		if err := batch.Commit(ctx); err != nil {
			return fmt.Errorf("failed to commit stale label cleanup: %w", err)
		}

		remoteLogger.Info("Cleaned up stale remote labels", "count", len(staleKeys))
	} else {
		remoteLogger.Debug("No stale remote labels found")
	}

	return nil
}

// startRemoteLabelCleanupTask starts a background task that periodically cleans up stale remote labels.
func (r *routeRemote) startRemoteLabelCleanupTask(ctx context.Context) {
	remoteLogger.Info("Starting remote label cleanup task", "interval", CleanupInterval)

	ticker := time.NewTicker(CleanupInterval)

	go func() {
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				remoteLogger.Info("Remote label cleanup task stopped")

				return
			case <-ticker.C:
				if err := r.cleanupStaleRemoteLabels(ctx); err != nil {
					remoteLogger.Error("Failed to cleanup stale remote labels", "error", err)
				}
			}
		}
	}()
}

// cleanupOrphanedLocalLabels removes local records and labels for CIDs that no longer exist in storage.
func (r *routeRemote) cleanupOrphanedLocalLabels(ctx context.Context, orphanedCIDs []string) int {
	cleanedCount := 0

	for _, cid := range orphanedCIDs {
		if r.cleanupLabelsForCID(ctx, cid) {
			cleanedCount++
		}
	}

	return cleanedCount
}

// cleanupLabelsForCID removes all local records and labels associated with a specific CID.
func (r *routeRemote) cleanupLabelsForCID(ctx context.Context, cid string) bool {
	batch, err := r.dstore.Batch(ctx)
	if err != nil {
		remoteLogger.Error("Failed to create cleanup batch", "cid", cid, "error", err)

		return false
	}

	keysDeleted := 0

	// Remove the /records/ key
	recordKey := datastore.NewKey("/records/" + cid)
	if err := batch.Delete(ctx, recordKey); err != nil {
		remoteLogger.Warn("Failed to delete record key", "key", recordKey.String(), "error", err)
	} else {
		keysDeleted++
	}

	// Find and remove all label keys for this CID across all namespaces
	localPeerID := r.server.Host().ID().String()

	for _, namespace := range validators.AllNamespaces() {
		// Query labels in this namespace that match our CID
		labelResults, err := r.dstore.Query(ctx, query.Query{
			Prefix: namespace.Prefix(),
		})
		if err != nil {
			remoteLogger.Warn("Failed to query labels for cleanup", "namespace", namespace, "cid", cid, "error", err)

			continue
		}

		defer labelResults.Close()

		for result := range labelResults.Next() {
			// Parse enhanced key to get CID and PeerID
			_, keyCID, keyPeerID, err := ParseEnhancedLabelKey(result.Key)
			if err != nil {
				remoteLogger.Warn("Failed to parse enhanced label key during cleanup, deleting",
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
					remoteLogger.Warn("Failed to delete label key", "key", labelKey.String(), "error", err)
				} else {
					keysDeleted++

					remoteLogger.Debug("Scheduled orphaned label for deletion", "key", result.Key)
				}
			}
		}
	}

	// Commit the batch deletion
	if err := batch.Commit(ctx); err != nil {
		remoteLogger.Error("Failed to commit orphaned label cleanup", "cid", cid, "error", err)

		return false
	}

	if keysDeleted > 0 {
		remoteLogger.Debug("Successfully cleaned up orphaned labels", "cid", cid, "keysDeleted", keysDeleted)
	}

	return keysDeleted > 0
}
