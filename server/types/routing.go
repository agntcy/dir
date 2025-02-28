package types

import (
	"context"
	"github.com/dep2p/libp2p/cid"
	"github.com/dep2p/libp2p/core/peer"
)

type RoutingEntry struct {
	Name string
	Cid  cid.Cid
	Path ImmutablePath
}

// RoutingAPI specifies the interface to the routing layer.
type RoutingAPI interface {
	// Provide announces to the network that you are providing given values
	Provide(context.Context, ImmutablePath) error

	// Resolve all the nodes that are providing this path.
	Resolve(context.Context, ImmutablePath) (<-chan peer.AddrInfo, error)

	// Lookup checks if a given node has this path.
	Lookup(context.Context, ImmutablePath) (bool, error)

	// List a given path. Only first children are returned.
	List(context.Context, ImmutablePath, chan<- RoutingEntry) error
}
