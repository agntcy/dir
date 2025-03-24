package routing

import (
	"context"

	"github.com/libp2p/go-libp2p-kad-dht/providers"
	"github.com/libp2p/go-libp2p/core/peer"
)

var _ providers.ProviderStore = &peerstore{}

// peerstore is used by the DHT to respond to content announcement and discovery data.
// TODO: decide what to do here based on routing primitives
type peerstore struct{}

func (p *peerstore) AddProvider(ctx context.Context, key []byte, prov peer.AddrInfo) error {
	return nil
}

func (p *peerstore) GetProviders(ctx context.Context, key []byte) ([]peer.AddrInfo, error) {
	return nil, nil
}

func (p *peerstore) Close() error {
	return nil
}
