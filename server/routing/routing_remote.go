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
	storeAPI       types.StoreAPI
	server         *p2p.Server
	service        *rpc.Service
	notifyCh       chan *handlerSync
	dstore         types.Datastore
	cleanupManager *CleanupManager
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

	// create cleanup manager for background tasks
	routeAPI.cleanupManager = NewCleanupManager(dstore, storeAPI, server)

	// run listener in background
	go routeAPI.handleNotify(ctx)

	// run label republishing task in background
	go routeAPI.cleanupManager.StartLabelRepublishTask(ctx)

	// run remote label cleanup task in background
	routeAPI.cleanupManager.StartRemoteLabelCleanupTask(ctx)

	return routeAPI, nil
}

func (r *routeRemote) Publish(ctx context.Context, ref *corev1.RecordRef, record *corev1.Record) error {
	remoteLogger.Debug("Called remote routing's Publish method for network operations", "ref", ref, "record", record)

	// Parse record CID
	decodedCID, err := cid.Decode(ref.GetCid())
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "failed to parse CID: %v", err)
	}

	// ✅ KEEP: Announce CID to DHT network (this triggers pull-based discovery)
	err = r.server.DHT().Provide(ctx, decodedCID, true)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to announce object %v: %v", ref.GetCid(), err)
	}

	// ❌ REMOVED: Label announcements via DHT.PutValue()
	// Labels are now discovered via pull-based mechanism when remote peers
	// receive the CID provider announcement and pull the content

	remoteLogger.Debug("Successfully announced CID to network for pull-based discovery",
		"ref", ref, "peers", r.server.DHT().RoutingTable().Size())

	return nil
}

// Search queries remote records using cached labels with OR logic and minimum threshold.
// Records are returned if they match at least minMatchScore queries (OR relationship).
func (r *routeRemote) Search(ctx context.Context, req *routingv1.SearchRequest) (<-chan *routingv1.SearchResponse, error) {
	remoteLogger.Debug("Called remote routing's Search method", "req", req)

	// ✅ DEFENSIVE: Deduplicate queries to ensure consistent scoring regardless of client behavior
	originalQueries := req.GetQueries()
	deduplicatedQueries := deduplicateQueries(originalQueries)

	if len(originalQueries) != len(deduplicatedQueries) {
		remoteLogger.Info("Deduplicated search queries for consistent scoring",
			"originalCount", len(originalQueries), "deduplicatedCount", len(deduplicatedQueries))
	}

	// ✅ PRODUCTION SAFETY: Enforce minimum match score for proto compliance
	// Proto: "If not set, it will return records that match at least one query"
	minMatchScore := req.GetMinMatchScore()
	if minMatchScore < DefaultMinMatchScore {
		minMatchScore = DefaultMinMatchScore
		remoteLogger.Debug("Applied minimum match score for production safety", "original", req.GetMinMatchScore(), "applied", minMatchScore)
	}

	// Output channel for results
	outCh := make(chan *routingv1.SearchResponse)

	// Process in background with deduplicated queries and corrected minMatchScore
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
	namespaces := []string{
		validators.NamespaceSkills.Prefix(),
		validators.NamespaceDomains.Prefix(),
		validators.NamespaceFeatures.Prefix(),
	}

	for _, namespace := range namespaces {
		if limitInt > 0 && processedCount >= limitInt {
			break
		}

		// Query all labels in this namespace
		labelResults, err := r.dstore.Query(ctx, query.Query{
			Prefix: namespace,
		})
		if err != nil {
			remoteLogger.Warn("Failed to query namespace for remote records", "namespace", namespace, "error", err)

			continue
		}
		defer labelResults.Close()

		// Process each label entry
		for result := range labelResults.Next() {
			if result.Error != nil {
				remoteLogger.Warn("Error reading label entry", "key", result.Key, "error", result.Error)

				continue
			}

			// Parse enhanced key to get components
			_, keyCID, keyPeerID, err := ParseEnhancedLabelKey(result.Key)
			if err != nil {
				remoteLogger.Warn("Failed to parse enhanced label key", "key", result.Key, "error", err)

				continue
			}

			// ✅ KEY DIFFERENCE: Filter for REMOTE records only
			if keyPeerID == localPeerID {
				continue // Skip local records
			}

			// Avoid duplicate CIDs (same record might have multiple matching labels)
			if processedCIDs[keyCID] {
				continue
			}

			// ✅ OR LOGIC: Calculate match score (how many queries match this record)
			matchQueries, score := r.calculateMatchScore(ctx, keyCID, queries, keyPeerID)

			remoteLogger.Debug("Calculated match score for remote record", "cid", keyCID, "score", score, "minMatchScore", minMatchScore, "matchingQueries", len(matchQueries))

			// ✅ OR LOGIC: Apply minimum match score filter (record included if score ≥ threshold)
			if score >= minMatchScore {
				// Get peer information
				peer := r.createPeerInfo(keyPeerID)

				// Send the response
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
	}

	remoteLogger.Debug("Completed Search operation", "processed", processedCount, "queries", len(queries))
}

// calculateMatchScore calculates how many queries match a remote record (OR logic).
// Returns the matching queries and the match score for minimum threshold filtering.
func (r *routeRemote) calculateMatchScore(ctx context.Context, cid string, queries []*routingv1.RecordQuery, peerID string) ([]*routingv1.RecordQuery, uint32) {
	if len(queries) == 0 {
		return nil, 0
	}

	// Get all labels for this remote record
	labels := r.getRemoteRecordLabels(ctx, cid, peerID)
	if len(labels) == 0 {
		return nil, 0
	}

	var matchingQueries []*routingv1.RecordQuery

	// OR LOGIC: Check each query against all labels - any match counts toward the score
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
func (r *routeRemote) getRemoteRecordLabels(ctx context.Context, cid, peerID string) []string {
	var labels []string

	// Query each namespace to find labels for this CID/PeerID combination
	namespaces := []string{
		validators.NamespaceSkills.Prefix(),
		validators.NamespaceDomains.Prefix(),
		validators.NamespaceFeatures.Prefix(),
	}

	for _, namespace := range namespaces {
		results, err := r.dstore.Query(ctx, query.Query{
			Prefix: namespace,
		})
		if err != nil {
			remoteLogger.Warn("Failed to query namespace for remote labels", "namespace", namespace, "cid", cid, "error", err)

			continue
		}

		for result := range results.Next() {
			if result.Error != nil {
				continue
			}

			// Parse enhanced key
			label, keyCID, keyPeerID, err := ParseEnhancedLabelKey(result.Key)
			if err != nil {
				continue
			}

			// Check if this key matches our target CID and PeerID
			if keyCID == cid && keyPeerID == peerID {
				labels = append(labels, label)
			}
		}

		results.Close()
	}

	return labels
}

// createPeerInfo creates a Peer message from a PeerID string.
func (r *routeRemote) createPeerInfo(peerID string) *routingv1.Peer {
	// TODO: Could be enhanced to include actual peer addresses if available
	return &routingv1.Peer{
		Id: peerID,
		// Addresses could be populated from DHT peerstore if needed
	}
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

	// Skip local announcements
	if peerIDStr == r.server.Host().ID().String() {
		remoteLogger.Debug("Ignoring self announcement", "cid", notif.Ref.GetCid())

		return
	}

	// Check if we already have cached labels for this remote record
	if r.hasRemoteRecordCached(ctx, notif.Ref.GetCid(), peerIDStr) {
		// ✅ This is a reannouncement - update lastSeen timestamps
		remoteLogger.Debug("Received reannouncement for cached record, updating lastSeen",
			"cid", notif.Ref.GetCid(), "peer", peerIDStr)

		r.updateRemoteRecordLastSeen(ctx, notif.Ref.GetCid(), peerIDStr)

		return
	}

	// New record - pull content and cache labels
	remoteLogger.Debug("New remote record announced, pulling content for label extraction",
		"cid", notif.Ref.GetCid(), "peer", peerIDStr)

	// Pull the actual content from the announcing peer
	record, err := r.service.Pull(ctx, notif.Peer.ID, notif.Ref)
	if err != nil {
		remoteLogger.Error("Failed to pull remote content for label caching",
			"cid", notif.Ref.GetCid(), "peer", peerIDStr, "error", err)

		return
	}

	// Extract all labels from the record content
	labels := GetLabels(record)
	if len(labels) == 0 {
		remoteLogger.Warn("No labels found in remote record", "cid", notif.Ref.GetCid(), "peer", peerIDStr)

		return
	}

	// Cache each label locally using enhanced key format
	now := time.Now()
	cachedCount := 0

	for _, label := range labels {
		enhancedKey := BuildEnhancedLabelKey(label, notif.Ref.GetCid(), peerIDStr)

		metadata := &LabelMetadata{
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
		"cid", notif.Ref.GetCid(), "peer", peerIDStr, "totalLabels", len(labels), "cached", cachedCount)
}

// hasRemoteRecordCached checks if we already have cached labels for this remote record.
// This helps avoid duplicate work and identifies reannouncement events.
func (r *routeRemote) hasRemoteRecordCached(ctx context.Context, cid, peerID string) bool {
	namespaces := []string{
		validators.NamespaceSkills.Prefix(),
		validators.NamespaceDomains.Prefix(),
		validators.NamespaceFeatures.Prefix(),
	}

	for _, namespace := range namespaces {
		results, err := r.dstore.Query(ctx, query.Query{Prefix: namespace})
		if err != nil {
			remoteLogger.Warn("Failed to query namespace for cache check", "namespace", namespace, "error", err)

			continue
		}
		defer results.Close()

		for result := range results.Next() {
			if result.Error != nil {
				continue
			}

			// Parse enhanced key to check if it matches our CID/PeerID
			_, keyCID, keyPeerID, err := ParseEnhancedLabelKey(result.Key)
			if err != nil {
				continue
			}

			if keyCID == cid && keyPeerID == peerID {
				return true // Found cached labels for this record
			}
		}
	}

	return false
}

// updateRemoteRecordLastSeen updates the lastSeen timestamp for all cached labels
// from a specific remote peer/CID combination (for reannouncement handling).
//
//nolint:gocognit // Complex but necessary logic for iterating multiple namespaces and updating cached labels
func (r *routeRemote) updateRemoteRecordLastSeen(ctx context.Context, cid, peerID string) {
	namespaces := []string{
		validators.NamespaceSkills.Prefix(),
		validators.NamespaceDomains.Prefix(),
		validators.NamespaceFeatures.Prefix(),
	}

	now := time.Now()
	updatedCount := 0

	for _, namespace := range namespaces {
		results, err := r.dstore.Query(ctx, query.Query{Prefix: namespace})
		if err != nil {
			remoteLogger.Warn("Failed to query namespace for lastSeen update", "namespace", namespace, "error", err)

			continue
		}
		defer results.Close()

		for result := range results.Next() {
			if result.Error != nil {
				continue
			}

			// Parse enhanced key to check if it matches our CID/PeerID
			_, keyCID, keyPeerID, err := ParseEnhancedLabelKey(result.Key)
			if err != nil {
				continue
			}

			//nolint:nestif // Complex nested structure necessary for error-safe metadata update
			if keyCID == cid && keyPeerID == peerID {
				// This is a matching cached label - update its lastSeen
				var metadata LabelMetadata
				if err := json.Unmarshal(result.Value, &metadata); err == nil {
					metadata.LastSeen = now // ✅ Update lastSeen timestamp

					if metadataBytes, err := json.Marshal(metadata); err == nil {
						err = r.dstore.Put(ctx, datastore.NewKey(result.Key), metadataBytes)
						if err == nil {
							updatedCount++

							remoteLogger.Debug("Updated lastSeen for cached label", "key", result.Key)
						} else {
							remoteLogger.Warn("Failed to update lastSeen for cached label", "key", result.Key, "error", err)
						}
					}
				} else {
					remoteLogger.Warn("Failed to unmarshal label metadata for lastSeen update", "key", result.Key, "error", err)
				}
			}
		}
	}

	remoteLogger.Debug("Updated lastSeen timestamps for reannounced record",
		"cid", cid, "peer", peerID, "updatedLabels", updatedCount)
}
