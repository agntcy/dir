// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package p2p

import (
	"context"
	"sync"
	"time"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

// newDHT creates a DHT to be served over libp2p host.
// DHT will serve as a bootstrap peer if no bootstrap peers provided.
func newDHT(ctx context.Context, host host.Host, bootstrapPeers []peer.AddrInfo, refreshPeriod time.Duration, options ...dht.Option) (*dht.IpfsDHT, error) {
	// If no bootstrap nodes provided, we are the bootstrap node.
	if len(bootstrapPeers) == 0 {
		options = append(options, dht.Mode(dht.ModeServer))
	} else {
		options = append(options, dht.BootstrapPeers(bootstrapPeers...))
	}

	// Set refresh period
	if refreshPeriod > 0 {
		options = append(options, dht.RoutingTableRefreshPeriod(refreshPeriod))
	}

	// Create DHT
	kdht, err := dht.New(ctx, host, options...)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	// Bootstrap DHT
	if err = kdht.Bootstrap(ctx); err != nil {
		return nil, err //nolint:wrapcheck
	}

	// Sync with bootstrap nodes
	var wg sync.WaitGroup
	for _, p := range bootstrapPeers {
		wg.Add(1)

		go func(p peer.AddrInfo) {
			defer wg.Done()

			if err := host.Connect(ctx, p); err != nil {
				logger.Error("Error while connecting to node", "node", p.ID, "error", err)

				return
			}

			logger.Info("Successfully connected to bootstrap node", "node", p.ID)
		}(p)
	}

	wg.Wait()

	// Tag and protect bootstrap peers to prevent Connection Manager from pruning them.
	// Bootstrap peers are critical for network entry and should never be disconnected.
	if host.ConnManager() != nil {
		for _, p := range bootstrapPeers {
			// Check if we're actually connected (connection might have failed)
			if host.Network().Connectedness(p.ID) == network.Connected {
				// Tag with high priority
				host.ConnManager().TagPeer(p.ID, "bootstrap", PeerPriorityBootstrap)

				// Protect (never disconnect)
				host.ConnManager().Protect(p.ID, "bootstrap")

				logger.Info("Protected bootstrap peer",
					"peer", p.ID.String(),
					"tag", "bootstrap",
					"priority", PeerPriorityBootstrap)
			}
		}
	}

	return kdht, nil
}
