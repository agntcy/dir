// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package p2p

import (
	"context"
	"fmt"

	"github.com/agntcy/dir/utils/logging"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	discovery "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/libp2p/go-libp2p/p2p/host/autorelay"
)

var logger = logging.Logger("p2p")

type Server struct {
	opts    *options
	host    host.Host
	dht     *dht.IpfsDHT
	closeFn func()
}

// New constructs a new p2p server.
func New(ctx context.Context, opts ...Option) (*Server, error) {
	logger.Debug("Creating new p2p server", "opts", opts)

	// Load options
	options := &options{}
	for _, opt := range append(opts, withRandomIdentity()) {
		if err := opt(options); err != nil {
			return nil, err
		}
	}

	// Start in the background.
	// Wait for ready status message before returning.
	status := <-start(ctx, options)
	if status.Err != nil {
		return nil, fmt.Errorf("failed while starting services: %w", status.Err)
	}

	server := &Server{
		opts:    options,
		host:    status.Host,
		dht:     status.DHT,
		closeFn: status.Close,
	}

	logger.Debug("P2P server created", "host", server.host.ID(), "addresses", server.P2pAddrs())

	return server, nil
}

// Info returns the addresses at which we can reach this server.
func (s *Server) Info() *peer.AddrInfo {
	return &peer.AddrInfo{
		ID:    s.host.ID(),
		Addrs: s.host.Addrs(),
	}
}

// Returns p2p specific addresses as addrinfos.
func (s *Server) P2pInfo() []peer.AddrInfo {
	var p2pInfos []peer.AddrInfo //nolint:prealloc

	for _, addr := range s.P2pAddrs() {
		p2pInfo, _ := peer.AddrInfoFromString(addr)
		p2pInfos = append(p2pInfos, *p2pInfo)
	}

	return p2pInfos
}

// Returns p2p specific addresses as strings.
func (s *Server) P2pAddrs() []string {
	var p2pAddrs []string //nolint:prealloc
	for _, addr := range s.host.Addrs() {
		p2pAddrs = append(p2pAddrs, fmt.Sprintf("%s/p2p/%s", addr.String(), s.host.ID().String()))
	}

	return p2pAddrs
}

func (s *Server) Host() host.Host {
	return s.host
}

func (s *Server) DHT() *dht.IpfsDHT {
	return s.dht
}

func (s *Server) Key() crypto.PrivKey {
	return s.host.Peerstore().PrivKey(s.host.ID())
}

// Close stops running services.
func (s *Server) Close() {
	s.closeFn()
}

type status struct {
	Err   error
	Host  host.Host
	DHT   *dht.IpfsDHT
	Close func()
}

// start starts all routing related services.
// This function runs until ctx is closed.
//
// TODO: maybe limit how long we should wait for status channel
// via contexts.
func start(ctx context.Context, opts *options) <-chan status {
	statusCh := make(chan status)

	go func() {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		// Create host
		host, err := newHost(opts.ListenAddress, opts.DirectoryAPIAddress, opts.Key)
		if err != nil {
			statusCh <- status{Err: err}

			return
		}

		defer host.Close()

		logger.Debug("Host created", "id", host.ID(), "addresses", host.Addrs())

		// Enable mDNS for local network peer discovery
		setupMDNS(host)

		// Create DHT
		var customDhtOpts []dht.Option
		if opts.DHTCustomOpts != nil {
			customDhtOpts, err = opts.DHTCustomOpts(host)
			if err != nil {
				statusCh <- status{Err: err}

				return
			}
		}

		kdht, err := newDHT(ctx, host, opts.BootstrapPeers, opts.RefreshInterval, customDhtOpts...)
		if err != nil {
			statusCh <- status{Err: err}

			return
		}
		defer kdht.Close()

		// Enable AutoRelay with DHT as peer source for finding relay candidates.
		// AutoRelay makes NAT'd peers reachable by establishing relay circuits.
		// The DHT routing table is queried to find potential relay peers.
		if err := setupAutoRelay(host, kdht); err != nil {
			logger.Warn("Failed to setup AutoRelay", "error", err)
		}

		// Advertise to rendezvous for initial peer discovery.
		// Peer discovery is now handled automatically by:
		//   - DHT: Bootstrap() connects to bootstrap peers at startup
		//   - DHT: RoutingTableRefreshPeriod() maintains routing table (every 30s)
		//   - GossipSub: Mesh maintenance with peer exchange (if enabled)
		//   - Connection Manager: Maintains healthy connection count (50-200)
		//
		// The custom discover() polling loop has been removed as it was redundant
		// with DHT's built-in peer discovery and caused excessive polling (60/min).
		if opts.Randevous != "" {
			routingDiscovery := discovery.NewRoutingDiscovery(kdht)

			_, err := routingDiscovery.Advertise(ctx, opts.Randevous)
			if err != nil {
				logger.Warn("Failed to advertise to rendezvous",
					"rendezvous", opts.Randevous,
					"error", err)
			} else {
				logger.Info("Advertised to rendezvous (discovery handled by DHT)",
					"rendezvous", opts.Randevous)
			}
		}
		// Register services. Only available on non-bootstrap nodes.
		if opts.APIRegistrer != nil && len(opts.BootstrapPeers) > 0 {
			err := opts.APIRegistrer(host)
			if err != nil {
				statusCh <- status{Err: err}

				return
			}
		}

		// Run until context expiry
		logger.Debug("Host and DHT created, running routing services", "host", host.ID(), "addresses", host.Addrs())

		for _, peer := range opts.BootstrapPeers {
			for _, addr := range peer.Addrs {
				host.Peerstore().AddAddr(peer.ID, addr, 0)
			}
		}

		<-kdht.RefreshRoutingTable()

		// At this point, we are done.
		// Notify listener that we are ready.
		statusCh <- status{
			Host: host,
			DHT:  kdht,
			Close: func() {
				cancel()
				host.Close()
				kdht.Close()
			},
		}

		// Wait for context to close
		<-ctx.Done()
	}()

	return statusCh
}

// setupAutoRelay enables AutoRelay with DHT as the peer source for finding relay candidates.
// AutoRelay provides guaranteed connectivity for NAT'd peers by establishing relay circuits.
// The DHT routing table is used to discover potential relay peers (public nodes).
func setupAutoRelay(h host.Host, kdht *dht.IpfsDHT) error {
	// Create a peer source function that queries DHT for relay candidates
	peerSource := func(ctx context.Context, numPeers int) <-chan peer.AddrInfo {
		peerChan := make(chan peer.AddrInfo)

		go func() {
			defer close(peerChan)

			// Get peers from DHT routing table
			// These are likely good relay candidates (public, well-connected)
			routingTable := kdht.RoutingTable()
			peers := routingTable.ListPeers()

			// Send up to numPeers candidates
			count := 0
			for _, p := range peers {
				if count >= numPeers {
					break
				}

				// Get peer's address info from peerstore
				addrs := h.Peerstore().Addrs(p)
				if len(addrs) > 0 {
					select {
					case peerChan <- peer.AddrInfo{ID: p, Addrs: addrs}:
						count++
					case <-ctx.Done():
						return
					}
				}
			}

			logger.Debug("Provided relay candidates from DHT",
				"requested", numPeers,
				"provided", count)
		}()

		return peerChan
	}

	// Enable AutoRelay with DHT-based peer source
	_, err := autorelay.NewAutoRelay(h, autorelay.WithPeerSource(peerSource))
	if err != nil {
		return fmt.Errorf("failed to enable AutoRelay: %w", err)
	}

	logger.Info("AutoRelay enabled with DHT peer source")

	return nil
}

// mdnsNotifee handles mDNS peer discovery events.
type mdnsNotifee struct {
	host host.Host
}

// HandlePeerFound is called when mDNS discovers a peer on the local network.
func (n *mdnsNotifee) HandlePeerFound(pi peer.AddrInfo) {
	// Connect to discovered local peer
	if err := n.host.Connect(context.Background(), pi); err != nil {
		logger.Debug("Failed to connect to mDNS discovered peer",
			"peer", pi.ID,
			"error", err)

		return
	}

	logger.Info("Connected to local peer via mDNS",
		"peer", pi.ID,
		"addrs", pi.Addrs)
}

// setupMDNS enables mDNS discovery for local network peers.
// Peers on the same LAN will discover each other in < 1 second without bootstrap nodes.
// This is useful for development, testing, and enterprise LAN deployments.
func setupMDNS(h host.Host) {
	notifee := &mdnsNotifee{host: h}

	service := mdns.NewMdnsService(h, MDNSServiceName, notifee)
	if err := service.Start(); err != nil {
		logger.Warn("Failed to start mDNS discovery",
			"service", MDNSServiceName,
			"error", err)

		return
	}

	logger.Info("mDNS local discovery enabled",
		"service", MDNSServiceName)
}
