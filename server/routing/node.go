// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"context"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	libp2prouting "github.com/libp2p/go-libp2p/core/routing"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	libp2ptls "github.com/libp2p/go-libp2p/p2p/security/tls"
)

// TODO: connect p2p interface to serve the Routing API.
type Node struct {
	node host.Host
}

// TODO: make ctor more configurable with options.
func NewNode() (*Node, func() error, error) {
	// Select keypair
	priv, _, err := crypto.GenerateKeyPair(
		crypto.Ed25519, // Select your key type. Ed25519 are nice short
		-1,             // Select key length when possible (i.e. RSA).
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create p2p host keypair: %w", err)
	}

	// Select connection manager
	connMgr, err := connmgr.NewConnManager(
		100, //nolint:mnd
		400, //nolint:mnd
		connmgr.WithGracePeriod(time.Minute),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create p2p host connection manager: %w", err)
	}

	// Create host node
	node, err := libp2p.New(
		// Use the keypair we generated
		libp2p.Identity(priv),
		// Multiple listen addresses
		libp2p.ListenAddrStrings(
			"/ip4/0.0.0.0/tcp/9000",         // regular tcp connections
			"/ip4/0.0.0.0/udp/9000/quic-v1", // a UDP endpoint for the QUIC transport
		),
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
		libp2p.Routing(func(h host.Host) (libp2prouting.PeerRouting, error) {
			idht, err := dht.New(context.Background(), h)

			return idht, err //nolint:wrapcheck
		}),
		// If you want to help other peers to figure out if they are behind
		// NATs, you can launch the server-side of AutoNAT too (AutoRelay
		// already runs the client)
		//
		// This service is highly rate-limited and should not cause any
		// performance issues.
		libp2p.EnableNATService(),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create p2p host: %w", err)
	}

	return &Node{
		node: node,
	}, node.Close, nil
}
