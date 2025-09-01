// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"context"

	corev1 "github.com/agntcy/dir/api/core/v1"
	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/libp2p/go-libp2p/core/peer"
)

type Peer = peer.AddrInfo

// RoutingAPI handles management of the routing layer.
type RoutingAPI interface {
	// Publish record to the network
	Publish(context.Context, *corev1.RecordRef, *corev1.Record) error

	// Search records from the network
	List(context.Context, *routingv1.ListRequest) (<-chan *routingv1.LegacyListResponse_Item, error)

	// Unpublish record from the network
	Unpublish(context.Context, *corev1.RecordRef, *corev1.Record) error
}

// PublicationAPI handles management of publication tasks.
type PublicationAPI interface {
	// CreatePublication creates a new publication task to be processed.
	CreatePublication(context.Context, *routingv1.PublishRequest) (string, error)
}
