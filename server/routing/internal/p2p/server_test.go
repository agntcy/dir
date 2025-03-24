// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package p2p_test

import (
	"context"
	"testing"
	"time"

	p2p "github.com/agntcy/dir/server/routing/internal/p2p"
	"github.com/agntcy/dir/server/routing/internal/p2p/mockstream"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
)

func TestServer(t *testing.T) {
	// set context
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	// create bootstrap node
	bootstrap, _ := startTestNode(ctx, t, "/ip4/0.0.0.0/tcp/0", nil)

	// connect some nodes
	_, aliceCh := startTestNode(ctx, t, "/ip4/0.0.0.0/tcp/0", bootstrap.P2pInfo())
	_, bobCh := startTestNode(ctx, t, "/ip4/0.0.0.0/tcp/0", bootstrap.P2pInfo())

	// wait to exchanged messages
	<-aliceCh
	<-bobCh
}

func startTestNode(ctx context.Context, t *testing.T, addr string, bootstrapAddrs []peer.AddrInfo) (*p2p.Server, <-chan string) {
	t.Helper()

	listenCh := make(chan string, 1)
	server, err := p2p.New(
		ctx,
		p2p.WithListenAddress(addr),
		p2p.WithBootstrapPeers(bootstrapAddrs),
		p2p.WithRefreshInterval(1*time.Second),
		p2p.WithRandevous("connect"),
		// // Both protocol API registration works.
		// // Any of them can be used to forward gRPC requests over.
		// // We chose socket approach as its faster and more stable for testing.
		// // This is okay as we only want to check if the networking works.
		// p2p.WithAPIRegistrer(func(host host.Host) error {
		// 	return mockrpc.Start(ctx, host, protocol.ID("/featX/v1.0.0"), listenCh, bootstrapAddrs)
		// }),
		p2p.WithAPIRegistrer(func(host host.Host) error {
			host.SetStreamHandler("/featY/v1.0.0", mockstream.HandleStream(ctx, listenCh))
			go mockstream.StartDataStream(ctx, host, "/featY/v1.0.0", listenCh)

			return nil
		}),
	)
	assert.NoError(t, err)

	return server, listenCh
}
