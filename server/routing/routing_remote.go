// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/agntcy/dir/server/routing/internal/p2p"
	"github.com/agntcy/dir/server/routing/rpc"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/providers"
	record "github.com/libp2p/go-libp2p-record"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
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
	parentRouter types.RoutingAPI,
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
				labelValidators := CreateLabelValidators()
				validator := record.NamespacedValidator{
					"skills":   labelValidators["skills"],
					"domains":  labelValidators["domains"],
					"features": labelValidators["features"],
				}

				// return custom opts for DHT
				return []dht.Option{
					dht.Datastore(dstore),                           // custom DHT datastore
					dht.ProtocolPrefix(protocol.ID(ProtocolPrefix)), // custom DHT protocol prefix
					dht.Validator(validator),                        // custom validators for label namespaces
					dht.MaxRecordAge(DHTRecordTTL),                  // set consistent TTL for all DHT records
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
	rpcService, err := rpc.New(server.Host(), storeAPI, parentRouter)
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

//nolint:mnd,cyclop
func (r *routeRemote) List(ctx context.Context, req *routingv1.ListRequest) (<-chan *routingv1.LegacyListResponse_Item, error) {
	remoteLogger.Debug("Called remote routing's List method for network operations", "req", req)

	// list data from remote for a given peer.
	// this returns all the records that fully match our query.
	if req.GetLegacyListRequest().GetPeer() != nil {
		remoteLogger.Info("Listing data for peer", "req", req)

		resp, err := r.service.List(ctx, []peer.ID{peer.ID(req.GetLegacyListRequest().GetPeer().GetId())}, &routingv1.ListRequest{
			LegacyListRequest: &routingv1.LegacyListRequest{
				Labels: req.GetLegacyListRequest().GetLabels(),
			},
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to list data on remote: %v", err)
		}

		return resp, nil
	}

	// get specific record from all remote peers hosting it
	// this returns all the peers that are holding requested record
	if ref := req.GetLegacyListRequest().GetRef(); ref != nil {
		remoteLogger.Info("Listing data for record", "ref", ref)

		// get record CID
		decodedCID, err := cid.Decode(ref.GetCid())
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "failed to parse CID: %v", err)
		}

		// find using the DHT
		provs, err := r.server.DHT().FindProviders(ctx, decodedCID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to find object providers: %v", err)
		}

		if len(provs) == 0 {
			return nil, status.Errorf(codes.NotFound, "no providers found for object: %s", ref.GetCid())
		}

		// stream results back
		resCh := make(chan *routingv1.LegacyListResponse_Item, 100)
		go func(provs []peer.AddrInfo, ref *corev1.RecordRef) {
			defer close(resCh)

			for _, prov := range provs {
				// pull record from peer
				// TODO: this is not optional because we pull everything
				// just for the sake of showing the result
				record, err := r.service.Pull(ctx, prov.ID, ref)
				if err != nil {
					remoteLogger.Error("failed to pull record", "error", err)

					continue
				}

				// get record
				labels := getLabels(record)

				// peer addrs to string
				var addrs []string
				for _, addr := range prov.Addrs {
					addrs = append(addrs, addr.String())
				}

				remoteLogger.Info("Found an announced record", "ref", ref, "peer", prov.ID, "labels", strings.Join(labels, ", "), "addrs", strings.Join(addrs, ", "))

				// send back to caller
				resCh <- &routingv1.LegacyListResponse_Item{
					Ref:    ref,
					Labels: labels,
					Peer: &routingv1.Peer{
						Id:    prov.ID.String(),
						Addrs: addrs,
					},
				}
			}
		}(provs, ref)

		return resCh, nil
	}

	// run a query across peers, keep forwarding until we exhaust the hops
	// TODO: this is a naive implementation, reconsider better selection of peers and scheduling.
	remoteLogger.Info("Listing data for all peers", "req", req)

	// resolve hops
	if req.GetLegacyListRequest().GetMaxHops() > MaxHops {
		return nil, errors.New("max hops exceeded")
	}

	//nolint:protogetter
	if req.LegacyListRequest.MaxHops != nil && *req.LegacyListRequest.MaxHops > 0 {
		*req.LegacyListRequest.MaxHops--
	}

	// run in the background
	resCh := make(chan *routingv1.LegacyListResponse_Item, 100)
	go func(ctx context.Context, req *routingv1.ListRequest) {
		defer close(resCh)

		// get data from peers (list what each of our connected peers has)
		resp, err := r.service.List(ctx, r.server.Host().Peerstore().Peers(), &routingv1.ListRequest{
			LegacyListRequest: &routingv1.LegacyListRequest{
				Peer:    req.GetLegacyListRequest().GetPeer(),
				Labels:  req.GetLegacyListRequest().GetLabels(),
				Ref:     req.GetLegacyListRequest().GetRef(),
				MaxHops: req.LegacyListRequest.MaxHops, //nolint:protogetter
			},
		})
		if err != nil {
			remoteLogger.Error("failed to list from peer over the network", "error", err)

			return
		}

		// TODO: crawl by continuing the walk based on hop count
		// IMPORTANT: do we really want to use other nodes as hops or our peers are enough?

		// pass the results back
		for item := range resp {
			resCh <- item
		}
	}(ctx, req)

	return resCh, nil
}

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
			case "LABEL":
				r.handleLabelNotification(ctx, notif)
			case "CID":
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
	remoteLogger.Info("Processing label announcement",
		"label", notif.LabelKey, "cid", notif.Ref.GetCid(), "peer", notif.Peer.ID)

	// Store the remote label mapping locally in our shared datastore
	// This allows us to discover this remote content via label searches
	err := r.dstore.Put(ctx, datastore.NewKey(notif.LabelKey), []byte(""))
	if err != nil {
		remoteLogger.Error("Failed to store remote label announcement",
			"label", notif.LabelKey, "error", err)

		return
	}

	remoteLogger.Info("Successfully stored remote label announcement",
		"label", notif.LabelKey, "peer", notif.Peer.ID)
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

	// FRAUD DETECTION: Validate that the announcing peer actually has the content
	// Step 1: Try to lookup metadata from the announcing peer
	_, err = r.service.Lookup(ctx, notif.Peer.ID, notif.Ref)
	if err != nil {
		remoteLogger.Error("FRAUD DETECTED: Peer announced CID but failed metadata lookup",
			"peer", notif.Peer.ID, "cid", notif.Ref.GetCid(), "error", err)

		return
	}

	// Step 2: Try to actually fetch the content from the announcing peer
	_, err = r.service.Pull(ctx, notif.Peer.ID, notif.Ref)
	if err != nil {
		remoteLogger.Error("FRAUD DETECTED: Peer announced CID but failed content delivery",
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
	ticker := time.NewTicker(LabelRepublishInterval)
	defer ticker.Stop()

	remoteLogger.Info("Started label republishing task", "interval", LabelRepublishInterval)

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
			remoteLogger.Warn("Failed to pull record for republishing", "cid", cid, "error", err)

			errorCount++

			continue
		}

		// Republish all label mappings for this record
		labels := getLabels(record)
		for _, label := range labels {
			// Use proper validator-compatible DHT key format
			dhtKey := FormatLabelKey(label, cid)

			// Republish label mapping to DHT network
			err = r.server.DHT().PutValue(ctx, dhtKey, []byte(cid))
			if err != nil {
				remoteLogger.Warn("Failed to republish label mapping", "label", dhtKey, "error", err)

				errorCount++
			} else {
				remoteLogger.Debug("Successfully republished label mapping", "label", dhtKey)

				republishedCount++
			}
		}
	}

	remoteLogger.Info("Completed label republishing cycle",
		"republished", republishedCount, "errors", errorCount)
}

// announceLabelToDHT announces a label mapping to the DHT network.
func (r *routeRemote) announceLabelToDHT(ctx context.Context, label, cidStr string) {
	// Announce to DHT network using proper validator-compatible key format
	// This automatically stores in the shared datastore AND announces to the network
	dhtKey := FormatLabelKey(label, cidStr)
	err := r.server.DHT().PutValue(ctx, dhtKey, []byte(cidStr))

	if err != nil {
		remoteLogger.Warn("Failed to announce label to DHT", "label", dhtKey, "error", err)
	} else {
		remoteLogger.Debug("Successfully announced label to DHT", "label", dhtKey)
	}
}
