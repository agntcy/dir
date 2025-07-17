// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"context"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	routingtypes "github.com/agntcy/dir/api/routing/v1alpha2"
	"github.com/libp2p/go-libp2p/core/peer"
)

type Peer = peer.AddrInfo

// RoutingAPI handles management of the routing layer.
type RoutingAPI interface {
	// Publish announces to the network that you are providing given object.
	// This writes to peer and content datastore.
	// It can perform sync to store the data to other nodes.
	// For now, we try sync on every publish.
	// TODO: find a better sync mechanism (buffered sync).
	// Request can be assumed to be validated.
	// We are only intersted in agent objects. Data on this object should be empty.
	Publish(ctx context.Context, object *coretypes.Object) error

	// Search to network with a given request.
	// This reads from content datastore.
	// Request can be assumed to be validated.
	List(context.Context, *routingtypes.ListRequest) (<-chan *routingtypes.LegacyListResponse_Item, error)

	// Unpublish removes the object from the network.
	// This removes the object from peer and content datastore.
	Unpublish(ctx context.Context, object *coretypes.Object) error
}
