// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"context"
	"fmt"
	"sync"
	"time"

	coretypes "github.com/agntcy/dir/api/core/types"
	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/agntcy/dir/server/routing/internal/p2p"
	"github.com/agntcy/dir/server/routing/rpc"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	"github.com/ipfs/go-cid"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/protocol"
	ma "github.com/multiformats/go-multiaddr"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var remoteLogger = logging.Logger("routing/remote")

// routeRemote handles routing across the network. Records and their labels are
// advertised as DHT provider keys, and discovery is a DHT lookup followed by a
// query against the peers it names.
type routeRemote struct {
	storeAPI types.StoreAPI

	server  *p2p.Server
	service *rpc.Service

	// dstore persists the DHT's own state; nothing in this package reads it.
	dstore types.Datastore

	// db is the authority on what this node holds, and so on what it advertises.
	db types.DatabaseAPI

	isBootstrapNode bool // True if this node is a bootstrap node (no bootstrap peers configured)

	// reprovideInterval is how often published records are re-advertised so
	// their DHT provider records do not expire.
	reprovideInterval time.Duration

	// readvertise asks for an extra advertise pass before the next tick, for
	// when the set of published records changes underneath the loop. Buffered
	// and sent to without blocking, so a request that arrives mid-pass is
	// coalesced into the one already pending rather than dropped.
	readvertise chan struct{}

	// Lifecycle management
	//nolint:containedctx // Context needed for managing lifecycle of long-running cleanup tasks
	ctx    context.Context    // Routing subsystem context
	cancel context.CancelFunc // Cancel function for graceful shutdown
	wg     sync.WaitGroup     // Tracks all background goroutines
}

func newRemote(parentCtx context.Context,
	storeAPI types.StoreAPI,
	db types.DatabaseAPI,
	dstore types.Datastore,
	opts types.APIOptions,
) (*routeRemote, error) {
	// Create routing subsystem context for lifecycle management of background tasks
	routingCtx, cancel := context.WithCancel(parentCtx)

	// Determine if this is a bootstrap node (no bootstrap peers configured)
	isBootstrapNode := len(opts.Config().Routing.BootstrapPeers) == 0

	// Create routing
	reprovideInterval := opts.Config().Routing.RepublishInterval
	if reprovideInterval <= 0 {
		reprovideInterval = RepublishInterval
	}

	routeAPI := &routeRemote{
		storeAPI:          storeAPI,
		dstore:            dstore,
		db:                db,
		ctx:               routingCtx,
		cancel:            cancel,
		isBootstrapNode:   isBootstrapNode,
		reprovideInterval: reprovideInterval,
		readvertise:       make(chan struct{}, 1),
	}

	refreshInterval := RefreshInterval
	if opts.Config().Routing.RefreshInterval > 0 {
		refreshInterval = opts.Config().Routing.RefreshInterval
	}

	// Use parent context for p2p server (should live as long as the server)
	server, err := p2p.New(parentCtx,
		p2p.WithListenAddress(opts.Config().Routing.ListenAddress),
		p2p.WithDirectoryAPIAddress(opts.Config().Routing.DirectoryAPIAddress),
		p2p.WithDirectoryOCIAddress(opts.Config().AdvertisedOCIAddress()),
		p2p.WithBootstrapAddrs(opts.Config().Routing.BootstrapPeers),
		p2p.WithRefreshInterval(refreshInterval),
		p2p.WithRandevous(ProtocolRendezvous), // enable libp2p auto-discovery
		p2p.WithIdentityKeyPath(opts.Config().Routing.KeyPath),
		p2p.WithRelayService(opts.Config().Routing.RelayService),
		p2p.WithStaticRelays(opts.Config().Routing.StaticRelays),
		p2p.WithForceReachabilityPrivate(opts.Config().Routing.ForceReachabilityPrivate),
		p2p.WithForceReachabilityPublic(opts.Config().Routing.ForceReachabilityPublic),
		p2p.WithCustomDHTOpts(
			func(_ host.Host) ([]dht.Option, error) {
				return []dht.Option{
					dht.Datastore(dstore),                           // custom DHT datastore
					dht.ProtocolPrefix(protocol.ID(ProtocolPrefix)), // custom DHT protocol prefix
					dht.MaxRecordAge(RecordTTL),                     // set consistent TTL for all DHT records
					dht.Mode(dht.ModeServer),
				}, nil
			},
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create p2p: %w", err)
	}

	routeAPI.server = server

	rpcService, err := rpc.New(server.Host(), storeAPI, db)
	if err != nil {
		defer server.Close()

		return nil, fmt.Errorf("failed to create RPC service: %w", err)
	}

	routeAPI.service = rpcService

	//nolint:contextcheck // Intentionally passing routing context to child goroutine for lifecycle management
	routeAPI.startAdvertiseTask()

	//nolint:contextcheck // Intentionally passing routing context to child goroutine for lifecycle management
	routeAPI.startPublishedMigrationTask(opts.Config().Routing.DatastoreDir)

	return routeAPI, nil
}

// Publish announces a record to the DHT: the CID so the record can be fetched,
// and its labels so it can be found.
//
// Flow:
//  1. Validate and extract CID from record
//  2. Announce CID to DHT (critical - returns error if fails)
//  3. Announce the record's labels and their ancestors (best-effort)
//
// Parameters:
//   - ctx: Operation context
//   - record: Record interface (caller must wrap corev1.Record with adapter)
//
// Returns:
//   - error: If critical operations fail (validation, CID parsing, DHT announcement)
func (r *routeRemote) Publish(ctx context.Context, record coretypes.Record) error {
	// Validation
	if record == nil {
		return status.Error(codes.InvalidArgument, "record is required") //nolint:wrapcheck
	}

	// Extract and validate CID
	cidStr := record.GetCid()
	if cidStr == "" {
		return status.Error(codes.InvalidArgument, "record has no CID") //nolint:wrapcheck
	}

	remoteLogger.Debug("Publishing record to network", "cid", cidStr)

	// Parse CID
	decodedCID, err := cid.Decode(cidStr)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid CID %q: %v", cidStr, err)
	}

	// 1. Announce CID to DHT network (content discovery)
	err = r.server.DHT().Provide(ctx, decodedCID, true)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to announce CID to DHT: %v", err)
	}

	// 2. Announce the record's labels, and their ancestors, as DHT keys.
	// This is what lets a peer searching for /skills/A reach a record tagged
	// /skills/A/B without any node holding a global index.
	labels := expandLabels(types.GetLabelsFromRecord(record))
	if failed := r.provideLabels(ctx, labels); failed > 0 {
		remoteLogger.Warn("Some label announcements failed",
			"cid", cidStr,
			"labels", len(labels),
			"failed", failed)
	}

	remoteLogger.Debug("Successfully announced record to network",
		"cid", cidStr,
		"labelKeys", len(labels),
		"dhtPeers", r.server.DHT().RoutingTable().Size())

	return nil
}

// Search normalises the request and streams the matches on a channel. Queries
// are OR'd: a record is returned once it matches minMatchScore of them.
func (r *routeRemote) Search(ctx context.Context, req *routingv1.SearchRequest) (<-chan *routingv1.SearchResponse, error) {
	remoteLogger.Debug("Called remote routing's Search method", "req", req)

	// Deduplicate queries to ensure consistent scoring regardless of client behavior
	originalQueries := req.GetQueries()
	deduplicatedQueries := deduplicateQueries(originalQueries)

	if len(originalQueries) != len(deduplicatedQueries) {
		remoteLogger.Info("Deduplicated search queries for consistent scoring",
			"originalCount", len(originalQueries), "deduplicatedCount", len(deduplicatedQueries))
	}

	// Enforce minimum match score for proto compliance
	// Proto: "If not set, it will return records that match at least one query"
	minMatchScore := req.GetMinMatchScore()
	if minMatchScore < DefaultMinMatchScore {
		minMatchScore = DefaultMinMatchScore
		remoteLogger.Debug("Applied minimum match score for production safety", "original", req.GetMinMatchScore(), "applied", minMatchScore)
	}

	outCh := make(chan *routingv1.SearchResponse)

	go func() {
		defer close(outCh)

		r.searchRemoteRecords(ctx, deduplicatedQueries, req.GetLimit(), minMatchScore, outCh)
	}()

	return outCh, nil
}

// extractProtocolValue returns the value of the given multiaddr protocol code
// from the first address that carries it, or "" if none do. The stored value is
// percent-encoded (see p2p.EncodeAppAddr), so it is decoded back to its original
// URL form before returning.
func extractProtocolValue(multiaddrs []ma.Multiaddr, code int) string {
	for _, addr := range multiaddrs {
		if value, err := addr.ValueForProtocol(code); err == nil && value != "" {
			return p2p.DecodeAppAddr(value)
		}
	}

	return ""
}

// Stop stops the remote routing services and releases resources.
// This should be called during server shutdown to clean up gracefully.
func (r *routeRemote) Stop() error {
	remoteLogger.Info("Stopping routing subsystem")

	// Cancel routing context to stop all background goroutines.
	r.cancel()

	// Wait for all goroutines to finish gracefully
	r.wg.Wait()
	remoteLogger.Debug("All routing background tasks stopped")

	// Close p2p server (host and DHT)
	r.server.Close()
	remoteLogger.Debug("P2P server closed")

	remoteLogger.Info("Routing subsystem stopped successfully")

	return nil
}

// IsReady checks if the remote routing subsystem is ready to serve traffic.
// For bootstrap nodes (first peer in network):
//   - Ready when DHT, host and datastore are initialized (0 peers is expected)
//
// For regular nodes (connecting to existing network):
//   - DHT must have peers in routing table
//   - Must have connected peers
func (r *routeRemote) IsReady(ctx context.Context) bool {
	if r.server == nil {
		remoteLogger.Debug("Routing not ready: server is nil")

		return false
	}

	// Check if host is initialized
	host := r.server.Host()
	if host == nil {
		remoteLogger.Debug("Routing not ready: host is nil")

		return false
	}

	// Check if DHT is initialized
	dht := r.server.DHT()
	if dht == nil {
		remoteLogger.Debug("Routing not ready: DHT is nil")

		return false
	}

	// Check if datastore is initialized
	if r.dstore == nil {
		remoteLogger.Debug("Routing not ready: datastore is nil")

		return false
	}

	// Verify host is listening on addresses
	// This ensures the libp2p transport layer is properly initialized
	addrs := host.Addrs()
	if len(addrs) == 0 {
		remoteLogger.Debug("Routing not ready: host has no listen addresses")

		return false
	}

	// Bootstrap nodes are ready when DHT is initialized, even with 0 peers
	// They serve as entry points for the network and will accept incoming connections
	if r.isBootstrapNode {
		remoteLogger.Debug("Routing ready (bootstrap node)", "listenAddrs", len(addrs))

		return true
	}

	// For regular nodes, require peers in routing table (successful bootstrap)
	routingTableSize := dht.RoutingTable().Size()
	if routingTableSize == 0 {
		remoteLogger.Debug("Routing not ready: DHT routing table is empty")

		return false
	}

	// Require at least one connected peer for regular nodes
	connectedPeers := len(host.Network().Peers())
	if connectedPeers == 0 {
		remoteLogger.Debug("Routing not ready: no connected peers")

		return false
	}

	remoteLogger.Debug("Routing ready", "routingTableSize", routingTableSize, "connectedPeers", connectedPeers)

	return true
}
