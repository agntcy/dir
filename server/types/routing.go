// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"context"

	corev1 "github.com/agntcy/dir/api/core/v1"
	routingtypes "github.com/agntcy/dir/api/routing/v1"
	"github.com/libp2p/go-libp2p/core/peer"
)

type Peer = peer.AddrInfo

// RoutingAPI handles management of the routing layer.
type RoutingAPI interface {
	// Publish record to the network
	Publish(context.Context, *corev1.RecordRef, *corev1.Record) error

	// Search records from the network
	List(context.Context, *routingtypes.ListRequest) (<-chan *routingtypes.LegacyListResponse_Item, error)

	// Unpublish record from the network
	Unpublish(context.Context, *corev1.RecordRef, *corev1.Record) error
}
