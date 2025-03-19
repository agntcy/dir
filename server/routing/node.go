// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"bufio"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/agntcy/dir/server/types"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	libp2ptls "github.com/libp2p/go-libp2p/p2p/security/tls"
)

const protocolID = "/dir/1.0.0"

// TODO: connect p2p interface to serve the Routing API.
type Node struct {
	host host.Host
	dht  *dht.IpfsDHT
}

// TODO: make ctor more configurable with options.
func NewNode(ctx context.Context, listenAddr string, bootstrapAddrs []string, datastore types.Datastore) (*Node, error) {
	// Select keypair
	priv, _, err := crypto.GenerateKeyPairWithReader(
		crypto.Ed25519, // Select your key type. Ed25519 are nice short
		-1,             // Select key length when possible (i.e. RSA).
		rand.Reader,    // Always generate a random note ID
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create p2p host keypair: %w", err)
	}

	// Select connection manager
	connMgr, err := connmgr.NewConnManager(
		100, //nolint:mnd
		400, //nolint:mnd
		connmgr.WithGracePeriod(time.Minute),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create p2p host connection manager: %w", err)
	}

	// Prepare DHT linkage for host
	var idht *dht.IpfsDHT

	// Create host node
	host, err := libp2p.New(
		// Use the keypair we generated
		libp2p.Identity(priv),
		// Multiple listen addresses
		libp2p.ListenAddrStrings(listenAddr),
		// support TLS connections
		libp2p.Security(libp2ptls.ID, libp2ptls.New),
		// support noise connections
		libp2p.Security(noise.ID, noise.New),
		// support any other default transports (TCP)
		libp2p.DefaultTransports,
		// Let's prevent our peer from having too many
		// connections by attaching a connection manager.
		libp2p.ConnectionManager(connMgr),
		// Attempt to open ports using uPNP for NATed hosts.
		libp2p.NATPortMap(),
		// Let this host use the DHT to find other hosts
		// libp2p.Routing(func(h host.Host) (libp2prouting.PeerRouting, error) {
		// 	return idht, nil
		// }),
		// If you want to help other peers to figure out if they are behind
		// NATs, you can launch the server-side of AutoNAT too (AutoRelay
		// already runs the client)
		//
		// This service is highly rate-limited and should not cause any
		// performance issues.
		libp2p.EnableNATService(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create p2p host: %w", err)
	}

	// Register Directory protocols
	host.SetStreamHandler(protocolID, handleStream)

	// Bootstrap the DHT
	dhtPeers := make([]peer.AddrInfo, len(bootstrapAddrs))
	for i, addr := range bootstrapAddrs {
		peerinfo, err := peer.AddrInfoFromString(addr)
		if err != nil {
			return nil, fmt.Errorf("invalid bootstrap addr: %w", err)
		}

		dhtPeers[i] = *peerinfo
	}
	idht, err = dht.New(ctx, host,
		// dht.ProtocolPrefix(protocol.ID(protocolID)),
		dht.BootstrapPeers(dhtPeers...),
		dht.Datastore(datastore),
		dht.Mode(dht.ModeServer),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create dht: %w", err)
	}
	if err := idht.Bootstrap(ctx); err != nil {
		return nil, fmt.Errorf("failed to bootstrap dht: %w", err)
	}

	// Advertise host with Dir protocol to the network
	routingDiscovery := drouting.NewRoutingDiscovery(idht)
	dutil.Advertise(ctx, routingDiscovery, protocolID)

	// Register readers
	peerChan, err := routingDiscovery.FindPeers(ctx, protocolID)
	if err != nil {
		return nil, fmt.Errorf("failed to find peers: %w", err)
	}
	go func() {
		for peer := range peerChan {
			if peer.ID == host.ID() {
				continue
			}
			fmt.Println("Found peer:", peer)

			fmt.Println("Connecting to:", peer)
			stream, err := host.NewStream(ctx, peer.ID, protocol.ID(protocolID))

			if err != nil {
				fmt.Println("Connection failed:", err)
				continue
			} else {
				rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

				go writeData(rw)
				go readData(rw)
			}

			fmt.Println("Connected to:", peer)
		}
	}()

	return &Node{
		host: host,
		dht:  idht,
	}, nil
}

func (n *Node) ID() string {
	return n.host.ID().String()
}

func (n *Node) Close() error {
	return errors.Join(
		n.host.Close(),
		n.dht.Close(),
	)
}
