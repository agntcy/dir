package types

import (
	"context"
	ipld "github.com/ipfs/go-ipld-format"
)

// API defines a unified interface to Dir
type API interface {
	// Store returns an implementation of Store API
	Store() StoreAPI

	// Dag returns an implementation of Dag API
	Dag() DagAPI

	// Routing returns an implementation of Routing API
	Routing() RoutingAPI

	// ResolvePath resolves the path (if not resolved already) into a path using Store API
	ResolvePath(context.Context, ImmutablePath) (ImmutablePath, error)

	// ResolveNode resolves the path (if not resolved already) into a node using Store API
	ResolveNode(context.Context, ImmutablePath) (ipld.Node, error)
}
