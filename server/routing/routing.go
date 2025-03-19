// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"context"
	"fmt"
	"log"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/server/routing/internal/service"
	"github.com/agntcy/dir/server/types"
	"github.com/ipfs/go-datastore/query"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

var (
	ProtocolID         = "dir"
	ProtocolRendezvous = ProtocolID + "connect"
)

type routing struct {
	ds types.Datastore
}

func New(opts types.APIOptions) (types.RoutingAPI, error) {
	return &routing{
		ds: opts.Datastore(),
	}, nil
}

func (r *routing) Publish(context.Context, *coretypes.ObjectRef, *coretypes.Agent) error {
	panic("unimplemented")
}

func (r *routing) List(context.Context, query.Query) ([]*coretypes.ObjectRef, error) {
	panic("unimplemented")
}

func (r *routing) Start(context.Context, query.Query) ([]*coretypes.ObjectRef, error) {
	panic("unimplemented")
}

// start starts all routing related services.
// This function runs until ctx is closed.
func start(ctx context.Context, listenAddr string, bootstrapAddrs []string, hostCh chan<- host.Host, listenCh chan<- string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Create host
	host, err := newHost(ctx, listenAddr)
	if err != nil {
		return err
	}
	hostCh <- host
	defer host.Close()

	log.Printf("Host ID: %s", host.ID())
	log.Printf("Host Addresses:")
	for _, addr := range host.Addrs() {
		log.Printf("  - %s/p2p/%s", addr, host.ID())
	}

	// Create DHT
	bootstrapPeers := make([]peer.AddrInfo, len(bootstrapAddrs))
	for i, addr := range bootstrapAddrs {
		peerinfo, err := peer.AddrInfoFromString(addr)
		if err != nil {
			return fmt.Errorf("invalid bootstrap addr: %w", err)
		}
		bootstrapPeers[i] = *peerinfo
	}

	kdht, err := newDHT(ctx, host, bootstrapPeers)
	if err != nil {
		return err
	}
	defer kdht.Close()

	// Start peer discovery
	go runDiscovery(ctx, host, kdht, ProtocolRendezvous)

	// Register service.
	// Directory services are available only on non-bootstrap nodes.
	if len(bootstrapPeers) > 0 {
		service := service.New(host, protocol.ID(ProtocolID), listenCh, bootstrapPeers)
		err = service.SetupRPC()
		if err != nil {
			log.Fatal(err)
		}
		go service.StartMessaging(ctx)
	}

	// Run until context expiry
	log.Print("Running routing services")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
