// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// nolint
package routing

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	routingtypes "github.com/agntcy/dir/api/routing/v1alpha1"
	"github.com/agntcy/dir/server/routing/internal/p2p"
	"github.com/agntcy/dir/server/routing/rpc"
	"github.com/agntcy/dir/server/types"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/providers"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

var (
	ProtocolPrefix     = "dir"
	ProtocolRendezvous = "dir/connect"

	// refresh interval for DHT routing tables.
	refreshInterval = 5 * time.Second
)

// this interface handles routing across the network.
// TODO: we shoud add caching here
type routeRemote struct {
	storeAPI types.StoreAPI
	server   *p2p.Server
	service  *rpc.Service
	notifyCh chan *handlerSync
}

func newRemote(ctx context.Context, storeAPI types.StoreAPI, opts types.APIOptions) (*routeRemote, error) {
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
				providerMgr, err := providers.NewProviderManager(h.ID(), h.Peerstore(), opts.Datastore())
				if err != nil {
					return nil, err
				}

				// return custom opts for DHT
				return []dht.Option{
					dht.Datastore(opts.Datastore()),                 // custom DHT datastore
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
	rpcService, err := rpc.New(ctx, server.Host(), storeAPI, routeAPI)
	if err != nil {
		defer server.Close()

		return nil, err
	}

	// update service
	routeAPI.service = rpcService

	// run listener in background
	go routeAPI.handleNotify(ctx)

	return routeAPI, nil
}

func (r *routeRemote) Publish(ctx context.Context, object *coretypes.Object, local bool) error {
	ref := object.GetRef()

	// get object CID
	cid, err := ref.GetCID()
	if err != nil {
		return fmt.Errorf("failed to get object CID: %w", err)
	}

	// announce to DHT
	err = r.server.DHT().Provide(ctx, cid, true)
	if err != nil {
		return fmt.Errorf("failed to announce object: %w", err)
	}

	return nil
}

func (r *routeRemote) List(ctx context.Context, req *routingtypes.ListRequest) (<-chan *routingtypes.ListResponse_Item, error) {
	// list data from remote for a given peer
	if req.GetPeer() != nil {
		// force the peer to return its local data
		req.Local = new(bool)
		*req.Local = true

		// TODO: handle error
		// we dont do anythin with the error for now, it can only time out
		resp, _ := r.service.List(ctx, []peer.ID{peer.ID(req.GetPeer().GetId())}, req)

		return resp, nil
	}

	// get specific agent from all remote peers hosting it
	if req.GetRecord() != nil {
		// get object CID
		cid, err := req.GetRecord().GetCID()
		if err != nil {
			return nil, fmt.Errorf("failed to get object CID: %w", err)
		}

		// find using the DHT
		provs, err := r.server.DHT().FindProviders(ctx, cid)
		if err != nil {
			return nil, fmt.Errorf("failed to find object providers: %w", err)
		}

		if len(provs) == 0 {
			return nil, fmt.Errorf("no providers found for object: %s", cid)
		}

		// stream results back
		// NOTE: we list from each provider
		resCh := make(chan *routingtypes.ListResponse_Item, 100)
		go func(provs []peer.AddrInfo, ref *coretypes.ObjectRef) {
			defer close(resCh)

			for _, prov := range provs {
				// get agent from peer
				object, err := r.service.Pull(ctx, prov.ID, ref)
				if err != nil {
					log.Printf("failed to pull agent: %v", err)

					continue
				}

				agent := object.GetAgent()

				// get agent skills
				var skills []string
				for _, skill := range agent.GetSkills() {
					skills = append(skills, skill.Key())
				}

				// peer addrs to string
				var addrs []string
				for _, addr := range prov.Addrs {
					addrs = append(addrs, addr.String())
				}

				// send back to caller
				resCh <- &routingtypes.ListResponse_Item{
					Record: object.GetRef(),
					Labels: skills,
					Peer: &routingtypes.Peer{
						Id:    prov.ID.String(),
						Addrs: addrs,
					},
				}
			}
		}(provs, req.GetRecord())

		return resCh, nil
	}

	// run a query across peers, keep forwarding until we exhaust the hops
	// TODO: this is a naive implementation, reconsider better selection of peers,
	// and scheduling.
	// fix number of hops
	if req.MaxHops == nil {
		req.MaxHops = new(uint32)
		*req.MaxHops = 5
	}

	if req.GetMaxHops() > 20 {
		return nil, errors.New("max hops exceeded")
	}

	resCh := make(chan *routingtypes.ListResponse_Item, 100)
	go func(ctx context.Context, req *routingtypes.ListRequest) {
		defer close(resCh)

		peers := r.server.Host().Peerstore().Peers()

		// get data from peers (list what each peer has)
		localReq := &routingtypes.ListRequest{
			Peer:    req.GetPeer(),
			Labels:  req.GetLabels(),
			Record:  req.GetRecord(),
			MaxHops: req.MaxHops,
			Local:   toPtr(true),
		}

		resp, err := r.service.List(ctx, peers, localReq)
		if err != nil {
			log.Printf("failed to list: %v", err)

			return
		}

		// stream local data results from each peer.
		// we need to drop some peers from querying here!
		for item := range resp {
			// TODO: filter what to return back
			resCh <- item // forward results back
		}

		//  check forwarding
		*req.MaxHops = req.GetMaxHops() - 1
		if req.MaxHops != nil && req.GetMaxHops() == 0 {
			// done
			return
		}

		// forward requests further
		resp, err = r.service.List(ctx, peers, req)
		if err != nil {
			log.Printf("failed to list: %v", err)

			return
		}

		// stream sub-query results
		for item := range resp {
			// TODO: filter what to return back
			resCh <- item // forward results back
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
				log.Printf("failed to check if agent exists locally: %v", err)

				continue procLoop
			}

			// TODO: we should subscribe to some agents so we can create a local copy
			// of the agent and its skills.
			// for now, we are only testing if we can reach out and fetch it from the
			// broadcasting node

			// lookup from remote
			meta, err := r.service.Lookup(ctx, notif.Peer.ID, notif.Ref)
			if err != nil {
				log.Printf("failed to lookup: %v", err)

				continue procLoop
			}

			// fetch model directly from peer and drop it
			object, err := r.service.Pull(ctx, notif.Peer.ID, notif.Ref)
			if err != nil {
				log.Printf("failed to pull: %v", err)

				continue procLoop
			}
			agent := object.GetAgent()

			// extract skills
			var skills []string
			for _, skill := range agent.GetSkills() {
				skills = append(skills, skill.Key())
			}

			// TODO: we can validate the agent here
			// for now, we just log the agent and its skills

			log.Printf("successfully processed agent %v with skills %s", meta, skills)
		}
	}
}
