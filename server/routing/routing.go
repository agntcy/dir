// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package routing provides distributed content routing capabilities for the dir system.
// It implements both local and remote routing strategies with automatic cleanup of stale data.
//
// The routing system consists of:
// - Local routing: Fast queries against local datastore
// - Remote routing: DHT-based discovery across the network
// - Cleanup service: Automatic removal of stale labels and orphaned records
//
// Label metadata is stored in JSON format with timestamps for lifecycle management.
package routing

import (
	"context"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/agntcy/dir/server/datastore"
	"github.com/agntcy/dir/server/types"
	"google.golang.org/grpc/status"
)

type route struct {
	local  *routeLocal
	remote *routeRemote
}

// hasPeersInRoutingTable checks if we have any peers in the DHT routing table.
// This determines whether we can perform network operations or should fall back to local-only mode.
func (r *route) hasPeersInRoutingTable() bool {
	if r.remote == nil || r.remote.server == nil {
		return false
	}

	return r.remote.server.DHT().RoutingTable().Size() > 0
}

func New(ctx context.Context, store types.StoreAPI, opts types.APIOptions) (types.RoutingAPI, error) {
	// Create main router
	mainRounter := &route{}

	// Create routing datastore
	var dsOpts []datastore.Option
	if dstoreDir := opts.Config().Routing.DatastoreDir; dstoreDir != "" {
		dsOpts = append(dsOpts, datastore.WithFsProvider(dstoreDir))
	}

	dstore, err := datastore.New(dsOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create routing datastore: %w", err)
	}

	// Create local router
	mainRounter.local = newLocal(store, dstore)

	// Create remote router
	mainRounter.remote, err = newRemote(ctx, mainRounter, store, dstore, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create remote routing: %w", err)
	}

	return mainRounter, nil
}

func (r *route) Publish(ctx context.Context, ref *corev1.RecordRef, record *corev1.Record) error {
	// Get local peer ID from the remote server host
	localPeerID := r.remote.server.Host().ID().String()

	// Always publish data locally for archival/querying
	err := r.local.Publish(ctx, ref, record, localPeerID)
	if err != nil {
		st := status.Convert(err)

		return status.Errorf(st.Code(), "failed to publish locally: %s", st.Message())
	}

	// Only publish to network if peers are available
	if r.hasPeersInRoutingTable() {
		err = r.remote.Publish(ctx, ref, record)
		if err != nil {
			st := status.Convert(err)

			return status.Errorf(st.Code(), "failed to publish to the network: %s", st.Message())
		}
	}

	return nil
}

func (r *route) List(ctx context.Context, req *routingv1.ListRequest) (<-chan *routingv1.LegacyListResponse_Item, error) {
	// Use remote routing when:
	// 1. Looking for a specific record (cid) - to find providers across the network
	// 2. MaxHops is set - indicates network traversal
	// But only if we have peers available for network operations
	if (req.GetLegacyListRequest().GetRef() != nil || req.GetLegacyListRequest().GetMaxHops() > 0) && r.hasPeersInRoutingTable() {
		return r.remote.List(ctx, req)
	}

	// Otherwise use local routing
	return r.local.List(ctx, req)
}

func (r *route) Unpublish(ctx context.Context, ref *corev1.RecordRef, record *corev1.Record) error {
	err := r.local.Unpublish(ctx, ref, record)
	if err != nil {
		st := status.Convert(err)

		return status.Errorf(st.Code(), "failed to unpublish locally: %s", st.Message())
	}

	// no need to explicitly handle unpublishing from the network
	// TODO clarify if network sync trigger is needed here
	return nil
}
