// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"context"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
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
	Publish(context.Context, *coretypes.ObjectRef, *coretypes.Agent) error

	// List a given key.
	// This reads from content datastore.
	List(context.Context, string) ([]*coretypes.ObjectRef, error)

	// TODO: Resolve all the nodes that are providing this key.
	// This reads from peer datastore.
	// Resolve(context.Context, Key) (<-chan *Peer, error)

	// TODO: Lookup checks if a given node has this key.
	// This reads from content datastore.
	// Lookup(context.Context, Key) (*coretypes.ObjectRef, error)

	// TODO: maybe add method to Walk a given key.
	// This walks a content routing table and extracts sub-keys and their associated values.
	// Walker starts from the highest-level of the tree and can be optionally re-feed
	// returned results to continue traversal to the lowest-levels.
}
