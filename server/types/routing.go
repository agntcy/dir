package types

import (
	"context"
	storetypes "github.com/agntcy/dir/api/store/v1alpha1"
	"github.com/dep2p/libp2p/core/peer"
)

type Key interface {
	String() string
	Namespace() string
	Path() []string
}

// RoutingAPI handles management of the routing layer.
type RoutingAPI interface {
	// Bootstrap rebuilds the routing tables to be served for the RoutingAPI.
	// This is a local operation that should read storage layer and rebuild.
	Bootstrap(context.Context) error

	// Publish announces to the network that you are providing given object.
	Publish(context.Context, *storetypes.ObjectRef) error

	// Resolve all the nodes that are providing this key.
	Resolve(context.Context, Key) (<-chan *peer.AddrInfo, error)

	// Lookup checks if a given node has this key.
	Lookup(context.Context, Key) (bool, error)

	// List a given key.
	//
	// This walks a routing table and extracts sub-keys and their associated values.
	// Walker starts from the highest-level of the tree and can be optionally re-feed
	// returned results to continue traversal to the lowest-levels. Returned keys may
	// optionally have associated ref on the StoreAPI, ie. Key[i] => ObjectRef[i] or
	// nil. Accepts parent key and regexp filters for sub-keys.
	List(context.Context, Key, []string) ([]Key, []storetypes.ObjectRef, error)
}
