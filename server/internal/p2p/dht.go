// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package p2p

import (
	"context"
	"log"
	"sync"
	"time"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
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
				log.Printf("Error while connecting to node %v: %-v", p.ID, err)

				return
			}

			log.Printf("Successfully connected to bootstrap node: %v", p.ID)
		}(p)
	}

	wg.Wait()

	return kdht, nil
}
