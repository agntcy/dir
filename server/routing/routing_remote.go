// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/agntcy/dir/server/routing/internal/p2p"
	"github.com/agntcy/dir/server/routing/rpc"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	"github.com/ipfs/go-cid"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/providers"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ProtocolPrefix     = "dir"
	ProtocolRendezvous = "dir/connect"

	// refresh interval for DHT routing tables.
	refreshInterval = 30 * time.Second

	remoteLogger = logging.Logger("routing/remote")
)

// this interface handles routing across the network.
// TODO: we shoud add caching here.
type routeRemote struct {
	storeAPI types.StoreAPI
	server   *p2p.Server
	service  *rpc.Service
	notifyCh chan *handlerSync
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
		notifyCh: make(chan *handlerSync, 1000),
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

				// return custom opts for DHT
				return []dht.Option{
					dht.Datastore(dstore),                           // custom DHT datastore
					dht.ProtocolPrefix(protocol.ID(ProtocolPrefix)), // custom DHT protocol prefix
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

	return routeAPI, nil
}

func (r *routeRemote) hasPeersInRoutingTable() bool {
	// Check if we have any peers in the DHT routing table
	rt := r.server.DHT().RoutingTable()

	return rt.Size() > 0
}

func (r *routeRemote) Publish(ctx context.Context, ref *corev1.RecordRef, record *corev1.Record) error {
	remoteLogger.Debug("Called remote routing's Publish method", "ref", ref, "record", record)

	// Check if we have peers connected for DHT operations i.e. if directory running in network mode.
	if !r.hasPeersInRoutingTable() {
		remoteLogger.Debug("No peers in DHT routing table, returning empty channel")

		return nil
	}

	// get record CID
	decodedCID, err := cid.Decode(ref.GetCid())
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "failed to parse CID: %v", err)
	}

	// announce to DHT
	err = r.server.DHT().Provide(ctx, decodedCID, true)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to announce object %v, it will be retried in the background. Reason: %v", ref.GetCid(), err)
	}

	remoteLogger.Debug("Successfully announced object to the network", "ref", ref)

	return nil
}

//nolint:mnd,cyclop
func (r *routeRemote) List(ctx context.Context, req *routingv1.ListRequest) (<-chan *routingv1.LegacyListResponse_Item, error) {
	remoteLogger.Debug("Called remote routing's List method", "req", req)

	// Check if we have peers connected for DHT operations i.e. if directory running in network mode.
	if !r.hasPeersInRoutingTable() {
		remoteLogger.Debug("No peers in DHT routing table, returning empty channel")

		// Return empty channel
		emptyCh := make(chan *routingv1.LegacyListResponse_Item)
		close(emptyCh)

		return emptyCh, nil
	}

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
	if req.GetLegacyListRequest().GetMaxHops() > 20 {
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
procLoop:
	for {
		select {
		case <-ctx.Done():
			return
		case notif := <-r.notifyCh:

			// check if we have this record locally
			_, err := r.storeAPI.Lookup(ctx, notif.Ref)
			if err != nil {
				// Check if this is a "record not found" error, which is expected behavior
				// for fresh deployments or when we haven't synced this record yet
				if st := status.Convert(err); st.Code() == codes.NotFound {
					remoteLogger.Debug("record not found locally, which is expected for unsynced content", "ref", notif.Ref, "peer", notif.Peer.ID)
				} else {
					// Log other lookup failures as errors since they indicate actual problems
					remoteLogger.Error("failed to check if record exists locally", "error", err, "ref", notif.Ref)
				}

				continue procLoop
			}

			// TODO: we should subscribe to some records so we can create a local copy
			// of the record and its skills.
			// for now, we are only testing if we can reach out and fetch it from the
			// broadcasting node

			// lookup from remote
			meta, err := r.service.Lookup(ctx, notif.Peer.ID, notif.Ref)
			if err != nil {
				remoteLogger.Error("failed to lookup record", "error", err)

				continue procLoop
			}

			// fetch model directly from peer and drop it
			record, err := r.service.Pull(ctx, notif.Peer.ID, notif.Ref)
			if err != nil {
				remoteLogger.Error("failed to pull record", "error", err)

				continue procLoop
			}

			// extract labels
			labels := getLabels(record)

			// TODO: we can perform validation and data synchronization here.
			// Depending on the server configuration, we can decide if we want to
			// pull this model into our own cache, rebroadcast it, or ignore it.

			remoteLogger.Info("Successfully processed record", "meta", meta, "labels", strings.Join(labels, ", "), "peer", notif.Peer.ID)
		}
	}
}
