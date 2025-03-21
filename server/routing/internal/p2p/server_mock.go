package p2p

import (
	"context"
	"fmt"
	"time"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/protocol"
)

// NewMockServer creates a bootstrap and server p2p nodes.
// This is used for testing only!
func NewMockServer(ctx context.Context, protocolExt protocol.ID) (*Server, error) {
	// bootstrap node
	bootstrap, err := newMockServer(ctx, protocolExt, "/ip4/0.0.0.0/tcp/0", nil)
	if err != nil {
		return nil, err
	}

	var bootAddrs []string //nolint:prealloc
	for _, addr := range bootstrap.Info().Addrs {
		bootAddrs = append(bootAddrs, fmt.Sprintf("%s/p2p/%s", addr.String(), bootstrap.Info().ID.String()))
	}

	// our connected node
	server, err := newMockServer(ctx, protocolExt, "/ip4/0.0.0.0/tcp/0", bootAddrs)
	if err != nil {
		return nil, err
	}

	return server, nil
}

func newMockServer(ctx context.Context, protocolExt protocol.ID, addr string, bootstrapAddrs []string) (*Server, error) {
	return New(ctx,
		WithListenAddress(addr),
		WithBootstrapAddrs(bootstrapAddrs),
		WithRefreshInterval(1*time.Second),
		WithCustomDHTOpts(
			dht.ProtocolExtension(protocolExt),
		),
	)
}
