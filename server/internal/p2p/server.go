// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package p2p

import (
	"context"
	"fmt"
	"log"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

type Server struct {
	opts    *options
	host    host.Host
	dht     *dht.IpfsDHT
	closeFn func()
}

// New constructs a new p2p server.
func New(ctx context.Context, opts ...Option) (*Server, error) {
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

	return &Server{
		opts:    options,
		host:    status.Host,
		dht:     status.DHT,
		closeFn: status.Close,
	}, nil
}

// Addresses returns the addresses at which we can reach this server.
func (s *Server) Info() *peer.AddrInfo {
	return &peer.AddrInfo{
		ID:    s.host.ID(),
		Addrs: s.host.Addrs(),
	}
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
		host, err := newHost(opts.ListenAddress, opts.Key)
		if err != nil {
			statusCh <- status{Err: err}

			return
		}

		defer host.Close()
		log.Printf("Host: %v %v", host.ID(), host.Addrs())

		// Create DHT
		kdht, err := newDHT(ctx, host, opts.BootstrapPeers, opts.RefreshInterval)
		if err != nil {
			statusCh <- status{Err: err}

			return
		}
		defer kdht.Close()

		// Start peer discovery if requested
		if opts.Randevous != "" {
			go discover(ctx, host, kdht, opts.Randevous)
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
		log.Print("Running routing services")

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
