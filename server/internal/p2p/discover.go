// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package p2p

import (
	"context"
	"log"
	"time"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	discovery "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	duitls "github.com/libp2p/go-libp2p/p2p/discovery/util"
)

// Discover starts the discovery process in a blocking mode.
// This should be started as a goroutine.
// Returns on context expiry.
func discover(ctx context.Context, h host.Host, dht *dht.IpfsDHT, rendezvous string) {
	routingDiscovery := discovery.NewRoutingDiscovery(dht)
	duitls.Advertise(ctx, routingDiscovery, rendezvous)

	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()

	log.Print("Peer discovery process running")

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Search for peers
			peers, err := duitls.FindPeers(ctx, routingDiscovery, rendezvous)
			if err != nil {
				log.Printf("Error while searching for peers: %v", err)

				continue
			}

			// Connect to discovered peers
			for _, p := range peers {
				if p.ID == h.ID() { // skip self
					continue
				}

				if h.Network().Connectedness(p.ID) == network.Connected { // skip connected
					continue
				}

				_, err = h.Network().DialPeer(ctx, p.ID)
				if err != nil {
					log.Printf("Error while connecting to peer %v: %v", p.ID, err)

					continue
				}

				log.Printf("Successfully connected to peer %s", p.ID.String())
			}
		}
	}
}
