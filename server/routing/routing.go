// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package routing makes the records a node has published discoverable across
// the network, and answers discovery queries from other nodes.
//
// Holding a record and publishing it are distinct: a held record is served to
// anyone who knows its CID, while publishing advertises its CID and labels as
// DHT provider keys so it can be found without one. Listing what this node has
// published is a local SQL query; searching the network is a DHT provider
// lookup followed by a query against the peers it names.
package routing

import (
	"context"
	"fmt"

	coretypes "github.com/agntcy/dir/api/core/types"
	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/agntcy/dir/server/datastore"
	"github.com/agntcy/dir/server/events"
	"github.com/agntcy/dir/server/types"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p/core/peer"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type route struct {
	local    *routeLocal
	remote   *routeRemote
	db       types.DatabaseAPI
	eventBus *events.SafeEventBus
	peerID   string
}

// hasPeersInRoutingTable checks if we have any peers in the DHT routing table.
// This determines whether we can perform network operations or should fall back to local-only mode.
func (r *route) hasPeersInRoutingTable() bool {
	if r.remote == nil || r.remote.server == nil {
		return false
	}

	return r.remote.server.DHT().RoutingTable().Size() > 0
}

func New(ctx context.Context, store types.StoreAPI, db types.DatabaseAPI, opts types.APIOptions) (types.RoutingAPI, error) {
	// Create main router
	mainRounter := &route{
		db:       db,
		eventBus: opts.EventBus(),
	}

	// Datastore for the DHT's own state: its routing table and the provider
	// records it holds for other peers. Nothing else writes to it.
	var dsOpts []datastore.Option
	if dstoreDir := opts.Config().Routing.DatastoreDir; dstoreDir != "" {
		dsOpts = append(dsOpts, datastore.WithFsProvider(dstoreDir))
	}

	dstore, err := datastore.New(dsOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create routing datastore: %w", err)
	}

	// Create remote router first to get the peer ID
	mainRounter.remote, err = newRemote(ctx, store, db, dstore, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create remote routing: %w", err)
	}

	// Get local peer ID from the remote server host
	mainRounter.peerID = mainRounter.remote.server.Host().ID().String()

	mainRounter.local = newLocal(db)

	return mainRounter, nil
}

// Publish marks a record as one this node announces, and announces it.
//
// Pushing a record only makes it servable to whoever already knows its CID.
// Publishing is what makes it discoverable, and the flag is what makes that
// durable: the reprovide cycle enumerates published records, so a node with no
// peers yet loses nothing by skipping the announcement here — it will announce
// as soon as a peer arrives.
func (r *route) Publish(ctx context.Context, record coretypes.Record) error {
	if record == nil {
		return status.Error(codes.InvalidArgument, "record is required") //nolint:wrapcheck
	}

	if err := r.setPublished(record.GetCid(), true); err != nil {
		return err
	}

	if r.hasPeersInRoutingTable() {
		if err := r.remote.Publish(ctx, record); err != nil {
			st := status.Convert(err)

			return status.Errorf(st.Code(), "failed to publish to the network: %s", st.Message())
		}
	} else {
		localLogger.Info("No DHT peers yet; record will be advertised once one is available", "cid", record.GetCid())
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

// Search returns records held by other peers. It asks the DHT which peers
// provide a queried label and then asks those peers what they hold, so results
// are best-effort: a peer unreachable within the search budget is missed.
//
// Records held by this node are not included; List covers those.
func (r *route) Search(ctx context.Context, req *routingv1.SearchRequest) (<-chan *routingv1.SearchResponse, error) {
	return r.remote.Search(ctx, req)
}

// Unpublish stops advertising a record. The node still holds it and still
// serves it to anyone who knows the CID; it just stops being discoverable.
//
// Nothing is withdrawn from the network, because Kademlia has no retraction:
// the provider records already at their custodians stay until they expire, up
// to RecordTTL. What this does is drop the record from every future reprovide
// cycle, which is the only durable way to stop announcing it.
//
// Labels need no special handling. The cycle recomputes the distinct label set
// from the advertised records each time, so a label stops being announced
// exactly when the last record carrying it does, with no refcount to maintain.
func (r *route) Unpublish(_ context.Context, record coretypes.Record) error {
	if record == nil {
		return status.Error(codes.InvalidArgument, "record is required") //nolint:wrapcheck
	}

	if err := r.setPublished(record.GetCid(), false); err != nil {
		return err
	}

	r.eventBus.RecordUnpublished(record.GetCid())

	return nil
}

// setPublished records whether the reprovide cycle should announce this CID.
func (r *route) setPublished(recordCID string, published bool) error {
	if recordCID == "" {
		return status.Error(codes.InvalidArgument, "record has no CID") //nolint:wrapcheck
	}

	if r.db == nil {
		return status.Error(codes.Unavailable, "routing has no database to record the published flag in") //nolint:wrapcheck
	}

	if err := r.db.SetRecordPublished(recordCID, published); err != nil {
		return status.Errorf(codes.Internal, "failed to set published flag for %s: %v", recordCID, err)
	}

	return nil
}

// Stop stops the routing services and releases resources.
// This should be called during server shutdown to clean up gracefully.
func (r *route) Stop() error {
	// Stop remote routing (includes the p2p server)
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
	limit := uint32(1)

	itemChan, err := r.local.List(ctx, &routingv1.ListRequest{Limit: &limit})
	if err != nil {
		localLogger.Debug("Routing not ready: local list request failed", "error", err)

		return false
	}

	for range itemChan {
	}

	if err := ctx.Err(); err != nil {
		localLogger.Debug("Routing not ready: local list canceled", "error", err)

		return false
	}

	if r.remote == nil {
		remoteLogger.Debug("Routing not ready: remote router is nil")

		return false
	}

	return r.remote.IsReady(ctx)
}

func (r *route) GetPeerID() string {
	return r.peerID
}

// GetProviderCount returns the number of distinct peers (including the local
// node) currently providing the given CID, via a DHT provider lookup.
//
// The local node is counted because Provide registers self in the local
// provider store, which FindProvidersAsync drains before going to the network.
// That also makes the count meaningful with an empty routing table.
//
// Best-effort: the result is whatever the lookup gathers within
// ProviderCountTimeout. Providers are deduplicated by peer ID because a peer
// can be emitted twice if the first sighting carried no addresses.
func (r *route) GetProviderCount(ctx context.Context, recordCID string) (int, error) {
	if r.remote == nil || r.remote.server == nil {
		return 0, fmt.Errorf("remote routing is not available")
	}

	decoded, err := cid.Decode(recordCID)
	if err != nil {
		return 0, fmt.Errorf("invalid CID %s: %w", recordCID, err)
	}

	lookupCtx, cancel := context.WithTimeout(ctx, ProviderCountTimeout)
	defer cancel()

	seen := make(map[peer.ID]struct{})

	// count=0 selects findAll. Any non-zero count lets the local provider store
	// short-circuit the network lookup once it holds that many entries, which
	// would make the count depend on what this node happens to have cached.
	for provider := range r.remote.server.DHT().FindProvidersAsync(lookupCtx, decoded, 0) {
		seen[provider.ID] = struct{}{}
	}

	return len(seen), nil
}
