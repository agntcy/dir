// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	routingtypes "github.com/agntcy/dir/api/routing/v1alpha1"
	"github.com/agntcy/dir/server/routing/internal/p2p"
	"github.com/agntcy/dir/server/routing/rpc"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
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
	// This is more accurate than checking peerstore as it verifies active DHT connections
	rt := r.server.DHT().RoutingTable()
	return rt.Size() > 0
}

func (r *routeRemote) Publish(ctx context.Context, object *coretypes.Object) error {
	remoteLogger.Debug("Called remote routing's Publish method", "object", object)

	// Skip if no peers are connected
	if !r.hasPeersInRoutingTable() {
		remoteLogger.Debug("No peers in DHT routing table, returning empty channel")

		return nil
	}

	ref := object.GetRef()

	// get object CID
	cid, err := ref.GetCID()
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "failed to get object CID: %v", err)
	}

	// announce to DHT
	err = r.server.DHT().Provide(ctx, cid, true)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to announce object %v, it will be retried in the background. Reason: %v", ref.GetDigest(), err)
	}

	remoteLogger.Debug("Successfully announced object to the network", "ref", ref)

	return nil
}

//nolint:mnd
func (r *routeRemote) List(ctx context.Context, req *routingtypes.ListRequest) (<-chan *routingtypes.ListResponse_Item, error) {
	remoteLogger.Debug("Called remote routing's List method", "req", req)

	// Check if we have peers connected for DHT operations
	if !r.hasPeersInRoutingTable() {
		remoteLogger.Debug("No peers in DHT routing table, returning empty channel")

		// Return empty channel
		emptyCh := make(chan *routingtypes.ListResponse_Item)
		close(emptyCh)
		return emptyCh, nil
	}

	// list data from remote for a given peer.
	// this returns all the records that fully match our query.
	if req.GetPeer() != nil {
		remoteLogger.Info("Listing data for peer", "req", req)

		resp, err := r.service.List(ctx, []peer.ID{peer.ID(req.GetPeer().GetId())}, &routingtypes.ListRequest{
			Labels: req.GetLabels(),
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to list data on remote: %v", err)
		}

		return resp, nil
	}

	// get specific agent from all remote peers hosting it
	// this returns all the peers that are holding requested agent
	if record := req.GetRecord(); record != nil {
		remoteLogger.Info("Listing data for record", "record", record)

		// get object CID
		cid, err := record.GetCID()
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "failed to get object CID: %v", err)
		}

		// find using the DHT
		provs, err := r.server.DHT().FindProviders(ctx, cid)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to find object providers: %v", err)
		}

		if len(provs) == 0 {
			return nil, status.Errorf(codes.NotFound, "no providers found for object: %s", record.GetDigest())
		}

		// stream results back
		resCh := make(chan *routingtypes.ListResponse_Item, 100)
		go func(provs []peer.AddrInfo, ref *coretypes.ObjectRef) {
			defer close(resCh)

			for _, prov := range provs {
				// pull agent from peer
				// TODO: this is not optional because we pull everything
				// just for the sake of showing the result
				object, err := r.service.Pull(ctx, prov.ID, ref)
				if err != nil {
					remoteLogger.Error("failed to pull agent", "error", err)

					continue
				}

				// get agent
				agent := object.GetAgent()
				labels := getLabels(&coretypes.Agent{Agent: agent})

				// peer addrs to string
				var addrs []string
				for _, addr := range prov.Addrs {
					addrs = append(addrs, addr.String())
				}

				remoteLogger.Info("Found an announced agent", "ref", ref, "peer", prov.ID, "labels", strings.Join(labels, ", "), "addrs", strings.Join(addrs, ", "))

				// send back to caller
				resCh <- &routingtypes.ListResponse_Item{
					Record: object.GetRef(),
					Labels: labels,
					Peer: &routingtypes.Peer{
						Id:    prov.ID.String(),
						Addrs: addrs,
					},
				}
			}
		}(provs, record)

		return resCh, nil
	}

	// run a query across peers, keep forwarding until we exhaust the hops
	// TODO: this is a naive implementation, reconsider better selection of peers and scheduling.
	remoteLogger.Info("Listing data for all peers", "req", req)

	// resolve hops
	if req.GetMaxHops() > 20 {
		return nil, errors.New("max hops exceeded")
	}

	//nolint:protogetter
	if req.MaxHops != nil && *req.MaxHops > 0 {
		*req.MaxHops--
	}

	// run in the background
	resCh := make(chan *routingtypes.ListResponse_Item, 100)
	go func(ctx context.Context, req *routingtypes.ListRequest) {
		defer close(resCh)

		// get data from peers (list what each of our connected peers has)
		resp, err := r.service.List(ctx, r.server.Host().Peerstore().Peers(), &routingtypes.ListRequest{
			Peer:    req.GetPeer(),
			Labels:  req.GetLabels(),
			Record:  req.GetRecord(),
			MaxHops: req.MaxHops, //nolint:protogetter
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

			// check if we have this agent locally
			_, err := r.storeAPI.Lookup(ctx, notif.Ref)
			if err != nil {
				remoteLogger.Error("failed to check if agent exists locally", "error", err)

				continue procLoop
			}

			// TODO: we should subscribe to some agents so we can create a local copy
			// of the agent and its skills.
			// for now, we are only testing if we can reach out and fetch it from the
			// broadcasting node

			// lookup from remote
			meta, err := r.service.Lookup(ctx, notif.Peer.ID, notif.Ref)
			if err != nil {
				remoteLogger.Error("failed to lookup agent", "error", err)

				continue procLoop
			}

			// fetch model directly from peer and drop it
			object, err := r.service.Pull(ctx, notif.Peer.ID, notif.Ref)
			if err != nil {
				remoteLogger.Error("failed to pull agent", "error", err)

				continue procLoop
			}
			agent := object.GetAgent()

			// extract labels
			labels := getLabels(&coretypes.Agent{Agent: agent})

			// TODO: we can perform validation and data synchronization here.
			// Depending on the server configuration, we can decide if we want to
			// pull this model into our own cache, rebroadcast it, or ignore it.

			remoteLogger.Info("Successfully processed agent", "meta", meta, "labels", strings.Join(labels, ", "), "peer", notif.Peer.ID)
		}
	}
}
