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

	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/agntcy/dir/server/datastore"
	"github.com/agntcy/dir/server/events"
	"github.com/agntcy/dir/server/types"
	"google.golang.org/grpc/status"
)

type route struct {
	local    *routeLocal
	remote   *routeRemote
	eventBus *events.SafeEventBus
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
	mainRounter := &route{
		eventBus: opts.EventBus(),
	}

	// Create routing datastore
	var dsOpts []datastore.Option
	if dstoreDir := opts.Config().Routing.DatastoreDir; dstoreDir != "" {
		dsOpts = append(dsOpts, datastore.WithFsProvider(dstoreDir))
	}

	dstore, err := datastore.New(dsOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create routing datastore: %w", err)
	}

	// Create remote router first to get the peer ID
	mainRounter.remote, err = newRemote(ctx, store, dstore, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create remote routing: %w", err)
	}

	// Get local peer ID from the remote server host
	localPeerID := mainRounter.remote.server.Host().ID().String()

	// Create local router with peer ID
	mainRounter.local = newLocal(store, dstore, localPeerID)

	return mainRounter, nil
}

func (r *route) Publish(ctx context.Context, record types.Record) error {
	// Always publish data locally for archival/querying
	err := r.local.Publish(ctx, record)
	if err != nil {
		st := status.Convert(err)

		return status.Errorf(st.Code(), "failed to publish locally: %s", st.Message())
	}

	// Only publish to network if peers are available
	if r.hasPeersInRoutingTable() {
		err = r.remote.Publish(ctx, record)
		if err != nil {
			st := status.Convert(err)

			return status.Errorf(st.Code(), "failed to publish to the network: %s", st.Message())
		}
	}

	// Emit RECORD_PUBLISHED event after successful publication
	labels := types.GetLabelsFromRecord(record)
	labelStrings := make([]string, len(labels))

	for i, label := range labels {
		labelStrings[i] = label.String()
	}

	r.eventBus.RecordPublished(record.GetCid(), labelStrings)

	return nil
}

func (r *route) List(ctx context.Context, req *routingv1.ListRequest) (<-chan *routingv1.ListResponse, error) {
	// List is always local-only - it returns records that this peer is currently providing
	// This operation does not interact with the network (per proto comment)
	return r.local.List(ctx, req)
}

func (r *route) Search(ctx context.Context, req *routingv1.SearchRequest) (<-chan *routingv1.SearchResponse, error) {
	// Search is always remote-only - it returns records from other peers using cached announcements
	// This operation queries locally cached remote announcements from DHT
	return r.remote.Search(ctx, req)
}

func (r *route) Unpublish(ctx context.Context, record types.Record) error {
	err := r.local.Unpublish(ctx, record)
	if err != nil {
		st := status.Convert(err)

		return status.Errorf(st.Code(), "failed to unpublish locally: %s", st.Message())
	}

	// Emit RECORD_UNPUBLISHED event after successful unpublication
	r.eventBus.RecordUnpublished(record.GetCid())

	// no need to explicitly handle unpublishing from the network
	// TODO clarify if network sync trigger is needed here
	return nil
}

// Stop stops the routing services and releases resources.
// This should be called during server shutdown to clean up gracefully.
func (r *route) Stop() error {
	// Stop remote routing (includes GossipSub and p2p server)
	if r.remote != nil {
		if err := r.remote.Stop(); err != nil {
			return fmt.Errorf("failed to stop remote routing: %w", err)
		}
	}

	return nil
}

// IsReady checks if the routing subsystem is ready to serve traffic.
func (r *route) IsReady(ctx context.Context) bool {
	// Check if local list request is successful
	_, err := r.local.List(ctx, &routingv1.ListRequest{})
	if err != nil {
		localLogger.Debug("Routing not ready: local list request failed", "error", err)

		return false
	}

	if r.remote == nil {
		remoteLogger.Debug("Routing not ready: remote router is nil")

		return false
	}

	return r.remote.IsReady(ctx)
}
