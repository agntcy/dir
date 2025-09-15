// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/agntcy/dir/server/routing/internal/p2p"
	"github.com/agntcy/dir/server/routing/rpc"
	validators "github.com/agntcy/dir/server/routing/validators"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/server/types/labels"
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

// NamespaceEntry contains processed namespace query data.
// This is used by namespace iteration functions for routing operations.
type NamespaceEntry struct {
	Namespace string
	Key       string
	Value     []byte
}

// QueryAllNamespaces queries all supported label namespaces and returns processed entries.
// This centralizes namespace iteration and datastore querying, eliminating code duplication
// between local and remote routing operations. All resource management is handled internally.
func QueryAllNamespaces(ctx context.Context, dstore types.Datastore, includeLocators bool) ([]NamespaceEntry, error) {
	var entries []NamespaceEntry

	// Define which namespaces to query
	namespaces := []string{
		labels.LabelTypeSkill.Prefix(),
		labels.LabelTypeDomain.Prefix(),
		labels.LabelTypeFeature.Prefix(),
	}

	// Include locators namespace if requested (local routing needs it, remote might not)
	if includeLocators {
		namespaces = append(namespaces, labels.LabelTypeLocator.Prefix())
	}

	for _, namespace := range namespaces {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("namespace query canceled: %w", ctx.Err())
		default:
		}

		results, err := dstore.Query(ctx, query.Query{Prefix: namespace})
		if err != nil {
			remoteLogger.Warn("Failed to query namespace", "namespace", namespace, "error", err)

			continue
		}

		// Process results and handle cleanup
		func() {
			defer results.Close()

			for result := range results.Next() {
				if result.Error != nil {
					continue
				}

				entries = append(entries, NamespaceEntry{
					Namespace: namespace,
					Key:       result.Key,
					Value:     result.Value,
				})
			}
		}()
	}

	return entries, nil
}

// routeRemote handles routing across the network with pull-based label caching.
type routeRemote struct {
	storeAPI       types.StoreAPI
	server         *p2p.Server
	service        *rpc.Service
	notifyCh       chan *handlerSync
	dstore         types.Datastore
	cleanupManager *CleanupManager
}

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

	refreshInterval := RefreshInterval
	if opts.Config().Routing.RefreshInterval > 0 {
		refreshInterval = opts.Config().Routing.RefreshInterval
	}

	server, err := p2p.New(ctx,
		p2p.WithListenAddress(opts.Config().Routing.ListenAddress),
		p2p.WithBootstrapAddrs(opts.Config().Routing.BootstrapPeers),
		p2p.WithRefreshInterval(refreshInterval),
		p2p.WithRandevous(ProtocolRendezvous), // enable libp2p auto-discovery
		p2p.WithIdentityKeyPath(opts.Config().Routing.KeyPath),
		p2p.WithCustomDHTOpts(
			func(h host.Host) ([]dht.Option, error) {
				providerMgr, err := providers.NewProviderManager(h.ID(), h.Peerstore(), dstore)
				if err != nil {
					return nil, fmt.Errorf("failed to create provider manager: %w", err)
				}

				labelValidators := validators.CreateLabelValidators()
				validator := record.NamespacedValidator{
					labels.LabelTypeSkill.String():   labelValidators[labels.LabelTypeSkill.String()],
					labels.LabelTypeDomain.String():  labelValidators[labels.LabelTypeDomain.String()],
					labels.LabelTypeFeature.String(): labelValidators[labels.LabelTypeFeature.String()],
				}

				return []dht.Option{
					dht.Datastore(dstore),                           // custom DHT datastore
					dht.ProtocolPrefix(protocol.ID(ProtocolPrefix)), // custom DHT protocol prefix
					dht.Validator(validator),                        // custom validators for label namespaces
					dht.MaxRecordAge(RecordTTL),                     // set consistent TTL for all DHT records
					dht.Mode(dht.ModeServer),
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

	routeAPI.server = server

	rpcService, err := rpc.New(server.Host(), storeAPI)
	if err != nil {
		defer server.Close()

		return nil, fmt.Errorf("failed to create RPC service: %w", err)
	}

	routeAPI.service = rpcService

	routeAPI.cleanupManager = NewCleanupManager(dstore, storeAPI, server)

	go routeAPI.handleNotify(ctx)

	go routeAPI.cleanupManager.StartLabelRepublishTask(ctx)

	routeAPI.cleanupManager.StartRemoteLabelCleanupTask(ctx)

	return routeAPI, nil
}

func (r *routeRemote) Publish(ctx context.Context, ref *corev1.RecordRef, record *corev1.Record) error {
	remoteLogger.Debug("Called remote routing's Publish method for network operations", "ref", ref, "record", record)

	decodedCID, err := cid.Decode(ref.GetCid())
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "failed to parse CID: %v", err)
	}

	// Announce CID to DHT network (triggers pull-based discovery)
	err = r.server.DHT().Provide(ctx, decodedCID, true)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to announce object %v: %v", ref.GetCid(), err)
	}

	// Note: Label announcements via DHT.PutValue() have been removed.
	// Labels are now discovered via pull-based mechanism when remote peers
	// receive the CID provider announcement and pull the content.

	remoteLogger.Debug("Successfully announced CID to network for pull-based discovery",
		"ref", ref, "peers", r.server.DHT().RoutingTable().Size())

	return nil
}

// Search queries remote records using cached labels with OR logic and minimum threshold.
// Records are returned if they match at least minMatchScore queries (OR relationship).
func (r *routeRemote) Search(ctx context.Context, req *routingv1.SearchRequest) (<-chan *routingv1.SearchResponse, error) {
	remoteLogger.Debug("Called remote routing's Search method", "req", req)

	// Deduplicate queries to ensure consistent scoring regardless of client behavior
	originalQueries := req.GetQueries()
	deduplicatedQueries := deduplicateQueries(originalQueries)

	if len(originalQueries) != len(deduplicatedQueries) {
		remoteLogger.Info("Deduplicated search queries for consistent scoring",
			"originalCount", len(originalQueries), "deduplicatedCount", len(deduplicatedQueries))
	}

	// Enforce minimum match score for proto compliance
	// Proto: "If not set, it will return records that match at least one query"
	minMatchScore := req.GetMinMatchScore()
	if minMatchScore < DefaultMinMatchScore {
		minMatchScore = DefaultMinMatchScore
		remoteLogger.Debug("Applied minimum match score for production safety", "original", req.GetMinMatchScore(), "applied", minMatchScore)
	}

	outCh := make(chan *routingv1.SearchResponse)

	go func() {
		defer close(outCh)
		r.searchRemoteRecords(ctx, deduplicatedQueries, req.GetLimit(), minMatchScore, outCh)
	}()

	return outCh, nil
}

// searchRemoteRecords searches for remote records using cached labels with OR logic.
// Records are returned if they match at least minMatchScore queries.
//
//nolint:gocognit // Core search algorithm requires complex logic for namespace iteration, filtering, and scoring
func (r *routeRemote) searchRemoteRecords(ctx context.Context, queries []*routingv1.RecordQuery, limit uint32, minMatchScore uint32, outCh chan<- *routingv1.SearchResponse) {
	localPeerID := r.server.Host().ID().String()
	processedCIDs := make(map[string]bool) // Avoid duplicates
	processedCount := 0
	limitInt := int(limit)

	remoteLogger.Debug("Starting remote search with OR logic and minimum threshold", "queries", len(queries), "minMatchScore", minMatchScore, "localPeerID", localPeerID)

	// Query all namespaces to find remote records
	entries, err := QueryAllNamespaces(ctx, r.dstore, false) // Remote doesn't need locators namespace
	if err != nil {
		remoteLogger.Error("Failed to get namespace entries for search", "error", err)

		return
	}

	for _, entry := range entries {
		if limitInt > 0 && processedCount >= limitInt {
			break
		}

		_, keyCID, keyPeerID, err := ParseEnhancedLabelKey(entry.Key)
		if err != nil {
			remoteLogger.Warn("Failed to parse enhanced label key", "key", entry.Key, "error", err)

			continue
		}

		// Filter for remote records only (exclude local records)
		if keyPeerID == localPeerID {
			continue // Skip local records
		}

		// Avoid duplicate CIDs (same record might have multiple matching labels)
		if processedCIDs[keyCID] {
			continue
		}

		// Calculate match score using OR logic (how many queries match this record)
		matchQueries, score := r.calculateMatchScore(ctx, keyCID, queries, keyPeerID)

		remoteLogger.Debug("Calculated match score for remote record", "cid", keyCID, "score", score, "minMatchScore", minMatchScore, "matchingQueries", len(matchQueries))

		// Apply minimum match score filter (record included if score ≥ threshold)
		if score >= minMatchScore {
			peer := r.createPeerInfo(keyPeerID)

			outCh <- &routingv1.SearchResponse{
				RecordRef:    &corev1.RecordRef{Cid: keyCID},
				Peer:         peer,
				MatchQueries: matchQueries,
				MatchScore:   score,
			}

			processedCIDs[keyCID] = true
			processedCount++

			remoteLogger.Debug("Record meets minimum threshold, including in results", "cid", keyCID, "score", score)

			if limitInt > 0 && processedCount >= limitInt {
				break
			}
		} else {
			remoteLogger.Debug("Record does not meet minimum threshold, excluding from results", "cid", keyCID, "score", score, "minMatchScore", minMatchScore)
		}
	}

	remoteLogger.Debug("Completed Search operation", "processed", processedCount, "queries", len(queries))
}

// calculateMatchScore calculates how many queries match a remote record (OR logic).
// Returns the matching queries and the match score for minimum threshold filtering.
func (r *routeRemote) calculateMatchScore(ctx context.Context, cid string, queries []*routingv1.RecordQuery, peerID string) ([]*routingv1.RecordQuery, uint32) {
	if len(queries) == 0 {
		return nil, 0
	}

	labels := r.getRemoteRecordLabels(ctx, cid, peerID)
	if len(labels) == 0 {
		return nil, 0
	}

	var matchingQueries []*routingv1.RecordQuery

	// Check each query against all labels - any match counts toward the score (OR logic)
	for _, query := range queries {
		if QueryMatchesLabels(query, labels) {
			matchingQueries = append(matchingQueries, query)
		}
	}

	score := safeIntToUint32(len(matchingQueries))

	remoteLogger.Debug("OR logic match score calculated", "cid", cid, "total_queries", len(queries), "matching_queries", len(matchingQueries), "score", score)

	return matchingQueries, score
}

// getRemoteRecordLabels gets labels for a remote record by finding all enhanced keys for this CID/PeerID.
func (r *routeRemote) getRemoteRecordLabels(ctx context.Context, cid, peerID string) []labels.Label {
	var labelList []labels.Label

	entries, err := QueryAllNamespaces(ctx, r.dstore, false) // Remote doesn't need locators namespace
	if err != nil {
		remoteLogger.Error("Failed to get namespace entries for labels", "error", err)

		return nil
	}

	for _, entry := range entries {
		label, keyCID, keyPeerID, err := ParseEnhancedLabelKey(entry.Key)
		if err != nil {
			continue
		}

		if keyCID == cid && keyPeerID == peerID {
			labelList = append(labelList, label)
		}
	}

	return labelList
}

// createPeerInfo creates a Peer message from a PeerID string.
func (r *routeRemote) createPeerInfo(peerID string) *routingv1.Peer {
	// TODO: Could be enhanced to include actual peer addresses if available
	return &routingv1.Peer{
		Id: peerID,
		// Addresses could be populated from DHT peerstore if needed
	}
}

func (r *routeRemote) handleNotify(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Process DHT provider notifications and handle pull-based label discovery
	for {
		select {
		case <-ctx.Done():
			return
		case notif := <-r.notifyCh:
			// All announcements are now CID provider announcements
			// Labels are discovered via pull-based mechanism
			r.handleCIDProviderNotification(ctx, notif)
		}
	}
}

// handleCIDProviderNotification implements pull-based label discovery and caching.
// When a remote peer announces they have content, we pull it and cache the labels locally.
func (r *routeRemote) handleCIDProviderNotification(ctx context.Context, notif *handlerSync) {
	peerIDStr := notif.Peer.ID.String()

	if peerIDStr == r.server.Host().ID().String() {
		remoteLogger.Debug("Ignoring self announcement", "cid", notif.Ref.GetCid())

		return
	}

	if r.hasRemoteRecordCached(ctx, notif.Ref.GetCid(), peerIDStr) {
		// This is a reannouncement - update lastSeen timestamps
		remoteLogger.Debug("Received reannouncement for cached record, updating lastSeen",
			"cid", notif.Ref.GetCid(), "peer", peerIDStr)

		r.updateRemoteRecordLastSeen(ctx, notif.Ref.GetCid(), peerIDStr)

		return
	}

	remoteLogger.Debug("New remote record announced, pulling content for label extraction",
		"cid", notif.Ref.GetCid(), "peer", peerIDStr)

	record, err := r.service.Pull(ctx, notif.Peer.ID, notif.Ref)
	if err != nil {
		remoteLogger.Error("Failed to pull remote content for label caching",
			"cid", notif.Ref.GetCid(), "peer", peerIDStr, "error", err)

		return
	}

	labelList := GetLabelsFromRecord(record)
	if len(labelList) == 0 {
		remoteLogger.Warn("No labels found in remote record", "cid", notif.Ref.GetCid(), "peer", peerIDStr)

		return
	}

	now := time.Now()
	cachedCount := 0

	for _, label := range labelList {
		enhancedKey := BuildEnhancedLabelKey(label, notif.Ref.GetCid(), peerIDStr)

		metadata := &labels.LabelMetadata{
			Timestamp: now,
			LastSeen:  now,
		}

		metadataBytes, err := json.Marshal(metadata)
		if err != nil {
			remoteLogger.Warn("Failed to marshal label metadata", "enhanced_key", enhancedKey, "error", err)

			continue
		}

		err = r.dstore.Put(ctx, datastore.NewKey(enhancedKey), metadataBytes)
		if err != nil {
			remoteLogger.Warn("Failed to cache remote label", "enhanced_key", enhancedKey, "error", err)
		} else {
			cachedCount++
		}
	}

	remoteLogger.Info("Successfully cached remote record labels via pull-based discovery",
		"cid", notif.Ref.GetCid(), "peer", peerIDStr, "totalLabels", len(labelList), "cached", cachedCount)
}

// hasRemoteRecordCached checks if we already have cached labels for this remote record.
// This helps avoid duplicate work and identifies reannouncement events.
func (r *routeRemote) hasRemoteRecordCached(ctx context.Context, cid, peerID string) bool {
	entries, err := QueryAllNamespaces(ctx, r.dstore, false) // Remote doesn't need locators namespace
	if err != nil {
		remoteLogger.Error("Failed to get namespace entries for cache check", "error", err)

		return false
	}

	for _, entry := range entries {
		// Parse enhanced key to check if it matches our CID/PeerID
		_, keyCID, keyPeerID, err := ParseEnhancedLabelKey(entry.Key)
		if err != nil {
			continue
		}

		if keyCID == cid && keyPeerID == peerID {
			return true
		}
	}

	return false
}

// updateLabelMetadataTimestamp updates the lastSeen timestamp for a single cached label entry.
func (r *routeRemote) updateLabelMetadataTimestamp(ctx context.Context, key string, value []byte, timestamp time.Time) error {
	var metadata labels.LabelMetadata
	if err := json.Unmarshal(value, &metadata); err != nil {
		return fmt.Errorf("failed to unmarshal label metadata: %w", err)
	}

	metadata.LastSeen = timestamp

	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal label metadata: %w", err)
	}

	err = r.dstore.Put(ctx, datastore.NewKey(key), metadataBytes)
	if err != nil {
		return fmt.Errorf("failed to save label metadata: %w", err)
	}

	return nil
}

// updateRemoteRecordLastSeen updates the lastSeen timestamp for all cached labels
// from a specific remote peer/CID combination (for reannouncement handling).
func (r *routeRemote) updateRemoteRecordLastSeen(ctx context.Context, cid, peerID string) {
	now := time.Now()
	updatedCount := 0

	entries, err := QueryAllNamespaces(ctx, r.dstore, false) // Remote doesn't need locators namespace
	if err != nil {
		remoteLogger.Error("Failed to get namespace entries for lastSeen update", "error", err)

		return
	}

	for _, entry := range entries {
		// Parse enhanced key to check if it matches our CID/PeerID
		_, keyCID, keyPeerID, err := ParseEnhancedLabelKey(entry.Key)
		if err != nil {
			continue
		}

		if keyCID == cid && keyPeerID == peerID {
			if err := r.updateLabelMetadataTimestamp(ctx, entry.Key, entry.Value, now); err != nil {
				remoteLogger.Warn("Failed to update lastSeen for cached label", "key", entry.Key, "error", err)
			} else {
				updatedCount++

				remoteLogger.Debug("Updated lastSeen for cached label", "key", entry.Key)
			}
		}
	}

	remoteLogger.Debug("Updated lastSeen timestamps for reannounced record",
		"cid", cid, "peer", peerID, "updatedLabels", updatedCount)
}
