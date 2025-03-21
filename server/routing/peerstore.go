package routing

import (
	"context"

	"github.com/libp2p/go-libp2p-kad-dht/providers"
	"github.com/libp2p/go-libp2p/core/peer"
)

type peerstore struct{}

// provider store is used by the DHT to respond to announce and discover data.
// this is where the magic happens.
func newPeerstore() (providers.ProviderStore, error) {
	return &peerstore{}, nil
}

func (p *peerstore) AddProvider(ctx context.Context, key []byte, prov peer.AddrInfo) error {
	// TODO: efficiently store set membership details
	return nil
}

func (p *peerstore) GetProviders(ctx context.Context, key []byte) ([]peer.AddrInfo, error) {
	// return set membership details
	// TODO: decide the algorithm later, ie. greedy, best, etc.
	return nil, nil
}

func (p *peerstore) Close() error {
	// close the underlying provider store
	return nil
}
