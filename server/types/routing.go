// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"context"

	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/libp2p/go-libp2p/core/peer"
)

type Peer = peer.AddrInfo

// RoutingAPI handles management of the routing layer.
type RoutingAPI interface {
	// Publish record to the network
	// The caller must wrap concrete record types (e.g. *corev1.Record) with adapters.NewRecordAdapter()
	Publish(context.Context, Record) error

	// List all records that this peer is currently providing (local-only operation)
	List(context.Context, *routingv1.ListRequest) (<-chan *routingv1.ListResponse, error)

	// Search for records across the network using cached remote announcements
	Search(context.Context, *routingv1.SearchRequest) (<-chan *routingv1.SearchResponse, error)

	// Unpublish record from the network
	// The caller must wrap concrete record types (e.g. *corev1.Record) with adapters.NewRecordAdapter()
	Unpublish(context.Context, Record) error

	// Stop stops the routing services and releases resources
	// Should be called during server shutdown for graceful cleanup
	Stop() error

	// IsReady checks if the routing subsystem is ready to serve traffic.
	IsReady(context.Context) bool
}

// PublicationAPI handles management of publication tasks.
type PublicationAPI interface {
	// CreatePublication creates a new publication task to be processed.
	CreatePublication(context.Context, *routingv1.PublishRequest) (string, error)
}
