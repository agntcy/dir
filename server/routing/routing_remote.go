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

	// Announce CID to DHT network
	err = r.server.DHT().Provide(ctx, decodedCID, true)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to announce object %v: %v", ref.GetCid(), err)
	}

	// Announce all label mappings to DHT network
	labels := GetLabels(record)
	for _, label := range labels {
		r.announceLabelToDHT(ctx, label, ref.GetCid())
	}

	remoteLogger.Debug("Successfully announced object and labels to network",
		"ref", ref, "labels", len(labels), "peers", r.server.DHT().RoutingTable().Size())

	return nil
}

// Search queries remote records using cached announcements from other peers.
// This mirrors the List implementation but filters for remote PeerIDs instead of local.
func (r *routeRemote) Search(ctx context.Context, req *routingv1.SearchRequest) (<-chan *routingv1.SearchResponse, error) {
	remoteLogger.Debug("Called remote routing's Search method", "req", req)

	// Output channel for results
	outCh := make(chan *routingv1.SearchResponse)

	// Process in background
	go func() {
		defer close(outCh)
		r.searchRemoteRecords(ctx, req.GetQueries(), req.GetLimit(), req.GetMinMatchScore(), outCh)
	}()

	return outCh, nil
}

// searchRemoteRecords searches for remote records using cached label announcements.
// Uses the same logic as List but filters for remote records (peerID != localPeerID).
//
//nolint:gocognit // Core search algorithm requires complex logic for namespace iteration, filtering, and scoring
func (r *routeRemote) searchRemoteRecords(ctx context.Context, queries []*routingv1.RecordQuery, limit uint32, minMatchScore uint32, outCh chan<- *routingv1.SearchResponse) {
	localPeerID := r.server.Host().ID().String()
	processedCIDs := make(map[string]bool) // Avoid duplicates
	processedCount := 0
	limitInt := int(limit)

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

			// Check if this record matches ALL queries (AND relationship - same as List)
			if r.matchesAllQueriesForSearch(ctx, keyCID, queries, keyPeerID) {
				// Calculate match score safely
				matchQueries := r.getMatchingQueries(result.Key, queries)
				score := safeIntToUint32(len(matchQueries))

				// Apply minimum match score filter
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

					if limitInt > 0 && processedCount >= limitInt {
						break
					}
				}
			}
		}
	}

	remoteLogger.Debug("Completed Search operation", "processed", processedCount, "queries", len(queries))
}

// matchesAllQueriesForSearch checks if a remote record matches ALL provided queries (AND relationship).
// Uses shared query matching logic with remote label retrieval strategy.
func (r *routeRemote) matchesAllQueriesForSearch(ctx context.Context, cid string, queries []*routingv1.RecordQuery, peerID string) bool {
	// Create closure that captures peerID for remote label retrieval
	labelRetriever := func(ctx context.Context, cid string) []string {
		return r.getRemoteRecordLabels(ctx, cid, peerID)
	}

	// Inject remote label retrieval strategy into shared query matching logic
	return MatchesAllQueries(ctx, cid, queries, labelRetriever)
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

// getMatchingQueries returns the queries that match against a specific label key.
// Uses shared query matching logic for consistency.
func (r *routeRemote) getMatchingQueries(labelKey string, queries []*routingv1.RecordQuery) []*routingv1.RecordQuery {
	// Use shared logic from query_matching.go
	return GetMatchingQueries(labelKey, queries)
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
