// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package autosync implements DHT-based record synchronization: it pulls records
// (and their referrers) announced by trusted peers over libp2p and ingests them
// locally with full parity to a normal push.
//
// Trust model (zero-trust): the allow-set is matched against the authenticated
// peer.ID (never a payload field), and all pulled content is verified before
// ingest — record CID integrity + OASF schema validation, and per-referrer
// belongs-to-record + type allow-list checks. Any failure is fail-closed
// (the offending item is skipped, never partially trusted).
package autosync

import (
	"context"
	"fmt"
	"sync"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/server/ingest"
	"github.com/agntcy/dir/server/routing/internal/p2p"
	"github.com/agntcy/dir/server/routing/rpc"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	ma "github.com/multiformats/go-multiaddr"
)

var logger = logging.Logger("routing/autosync")

// Worker pool configuration.
const (
	// workerCount is the number of concurrent workers pulling+ingesting records.
	workerCount = 4

	// queueSize bounds the pending autosync jobs. When full, new announcements
	// are dropped (a later re-announcement/republish re-triggers), so autosync
	// never blocks the DHT notification handler.
	queueSize = 256

	// jobTimeout bounds a single record's pull + ingest (record + referrers).
	jobTimeout = 60 * time.Second
)

// allowedReferrerTypes is the deny-by-default set of referrer types the autosync
// worker will pull and ingest from a peer. Anything else is rejected.
var allowedReferrerTypes = map[string]struct{}{
	corev1.SignatureReferrerType:  {},
	corev1.PublicKeyReferrerType:  {},
	corev1.ScanReportReferrerType: {},
}

// job is a unit of work: pull+ingest the announced record from a peer.
type job struct {
	ref  *corev1.RecordRef
	peer peer.AddrInfo
}

// recordFetcher is the subset of the libp2p RPC service the autosync worker
// needs. It is satisfied by *rpc.Service and mocked in tests.
type recordFetcher interface {
	Pull(ctx context.Context, peerID peer.ID, ref *corev1.RecordRef) (*corev1.Record, error)
	ListReferrers(ctx context.Context, peerID peer.ID, ref *corev1.RecordRef) ([]rpc.ReferrerDescriptor, error)
	PullReferrer(ctx context.Context, peerID peer.ID, ref *corev1.RecordRef, desc rpc.ReferrerDescriptor) (*corev1.RecordReferrer, error)
}

// peerRouter abstracts the reachability operations needed to dial a source peer
// (peerstore address hints + DHT re-resolution). Satisfied by serverPeerRouter.
type peerRouter interface {
	AddAddrs(peerID peer.ID, addrs []ma.Multiaddr)
	FindPeer(ctx context.Context, peerID peer.ID) (peer.AddrInfo, error)
}

// serverPeerRouter adapts *p2p.Server to peerRouter.
type serverPeerRouter struct {
	server *p2p.Server
}

func (s serverPeerRouter) AddAddrs(peerID peer.ID, addrs []ma.Multiaddr) {
	s.server.Host().Peerstore().AddAddrs(peerID, addrs, peerstore.TempAddrTTL)
}

func (s serverPeerRouter) FindPeer(ctx context.Context, peerID peer.ID) (peer.AddrInfo, error) {
	return s.server.DHT().FindPeer(ctx, peerID) //nolint:wrapcheck
}

// Manager pulls records (and their referrers) announced by trusted peers over
// libp2p and ingests them locally with full parity to a normal push.
type Manager struct {
	allowSet  map[peer.ID]struct{}
	transport recordFetcher
	router    peerRouter
	ingestor  ingest.Ingestor
	store     types.StoreAPI
	validator corev1.Validator

	queue    chan job
	inFlight map[string]struct{}
	mu       sync.Mutex
}

// NewManager creates an autosync Manager wired to the live libp2p RPC service
// and p2p server.
func NewManager(
	allowSet map[peer.ID]struct{},
	service *rpc.Service,
	server *p2p.Server,
	ingestor ingest.Ingestor,
	store types.StoreAPI,
	validator corev1.Validator,
) *Manager {
	return newManager(allowSet, service, serverPeerRouter{server: server}, ingestor, store, validator)
}

// newManager is the interface-based constructor used by NewManager and tests.
func newManager(
	allowSet map[peer.ID]struct{},
	transport recordFetcher,
	router peerRouter,
	ingestor ingest.Ingestor,
	store types.StoreAPI,
	validator corev1.Validator,
) *Manager {
	return &Manager{
		allowSet:  allowSet,
		transport: transport,
		router:    router,
		ingestor:  ingestor,
		store:     store,
		validator: validator,
		queue:     make(chan job, queueSize),
		inFlight:  make(map[string]struct{}),
	}
}

// Start launches the bounded worker pool. Workers stop when ctx is cancelled.
func (m *Manager) Start(ctx context.Context, wg *sync.WaitGroup) {
	for range workerCount {
		wg.Go(func() {
			m.worker(ctx)
		})
	}

	logger.Info("Autosync workers started",
		"workers", workerCount,
		"trusted_peers", len(m.allowSet),
	)
}

// MaybeEnqueue schedules an autosync job if the announcing peer is trusted and
// the CID is not already in flight. Non-blocking: if the queue is full the
// announcement is dropped (a later re-announcement re-triggers), so the DHT
// notification handler is never blocked.
//
// addrInfo is the announcing peer's authenticated identity + addresses from the
// DHT provider record.
func (m *Manager) MaybeEnqueue(ref *corev1.RecordRef, addrInfo peer.AddrInfo) {
	if ref == nil {
		return
	}

	// Authorization: only sync from peers on the allow-set, matched against the
	// authenticated peer.ID provided by the DHT provider record.
	if _, ok := m.allowSet[addrInfo.ID]; !ok {
		return
	}

	cid := ref.GetCid()
	if cid == "" {
		return
	}

	// Dedupe: skip if this CID is already queued or being processed.
	m.mu.Lock()
	if _, ok := m.inFlight[cid]; ok {
		m.mu.Unlock()

		return
	}

	m.inFlight[cid] = struct{}{}
	m.mu.Unlock()

	select {
	case m.queue <- job{ref: ref, peer: addrInfo}:
		logger.Debug("Autosync job enqueued", "cid", cid, "peer", addrInfo.ID)
	default:
		// Queue full: release the in-flight marker and drop.
		m.clearInFlight(cid)
		logger.Warn("Autosync queue full, dropping announcement", "cid", cid, "peer", addrInfo.ID)
	}
}

func (m *Manager) clearInFlight(cid string) {
	m.mu.Lock()
	delete(m.inFlight, cid)
	m.mu.Unlock()
}

func (m *Manager) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case j := <-m.queue:
			m.process(ctx, j)
			m.clearInFlight(j.ref.GetCid())
		}
	}
}

// process pulls, verifies, and ingests a single announced record + referrers.
// It is fail-closed: any verification failure aborts ingestion of the offending
// item.
func (m *Manager) process(parentCtx context.Context, j job) {
	cid := j.ref.GetCid()
	peerID := j.peer.ID

	// Dedupe against local storage: if we already have the record, skip the pull.
	if _, err := m.store.Lookup(parentCtx, j.ref); err == nil {
		logger.Debug("Record already present locally, skipping autosync", "cid", cid, "peer", peerID)

		return
	}

	// Bound the whole job so a slow/unreachable peer cannot tie up the worker.
	ctx, cancel := context.WithTimeout(parentCtx, jobTimeout)
	defer cancel()

	record, err := m.pullRecord(ctx, peerID, j.ref, j.peer.Addrs)
	if err != nil {
		logger.Error("Autosync failed to pull record", "cid", cid, "peer", peerID, "error", err)

		return
	}

	// Integrity: the pulled content must hash to the announced CID. This prevents
	// a peer from substituting different content for an announced CID.
	if actual := record.GetCid(); actual != cid {
		logger.Warn("Autosync rejected record: CID mismatch",
			"announced_cid", cid, "actual_cid", actual, "peer", peerID)

		return
	}

	// Validity: run the same OASF schema gate as a normal push before ingest.
	valid, validationErrors, err := record.ValidateWith(ctx, m.validator)
	if err != nil {
		logger.Error("Autosync record validation error", "cid", cid, "peer", peerID, "error", err)

		return
	}

	if !valid {
		logger.Warn("Autosync rejected record: OASF validation failed",
			"cid", cid, "peer", peerID, "errors", validationErrors)

		return
	}

	if _, err := m.ingestor.ImportRecord(ctx, record); err != nil {
		logger.Error("Autosync failed to ingest record", "cid", cid, "peer", peerID, "error", err)

		return
	}

	logger.Info("Autosync ingested record", "cid", cid, "peer", peerID)

	m.syncReferrers(ctx, peerID, j.ref)
}

// pullRecord pulls the record over libp2p RPC. Addresses from the announcement
// are added to the peerstore first; on failure it re-resolves the peer via the
// DHT (FindPeer) and retries once. Fuller backoff/relay handling is a follow-up.
func (m *Manager) pullRecord(ctx context.Context, peerID peer.ID, ref *corev1.RecordRef, addrs []ma.Multiaddr) (*corev1.Record, error) {
	if len(addrs) > 0 {
		m.router.AddAddrs(peerID, addrs)
	}

	record, err := m.transport.Pull(ctx, peerID, ref)
	if err == nil {
		return record, nil
	}

	logger.Debug("Autosync pull failed, re-resolving peer via DHT", "peer", peerID, "error", err)

	addrInfo, findErr := m.router.FindPeer(ctx, peerID)
	if findErr != nil {
		return nil, fmt.Errorf("failed to pull record from peer %s: %w", peerID, err)
	}

	if len(addrInfo.Addrs) > 0 {
		m.router.AddAddrs(peerID, addrInfo.Addrs)
	}

	return m.transport.Pull(ctx, peerID, ref) //nolint:wrapcheck
}

// syncReferrers lists, pulls, verifies, and ingests the record's referrers.
// Each referrer is verified independently (belongs-to-record + type allow-list);
// a failing referrer is skipped without affecting the already-ingested record.
func (m *Manager) syncReferrers(ctx context.Context, peerID peer.ID, recordRef *corev1.RecordRef) {
	descriptors, err := m.transport.ListReferrers(ctx, peerID, recordRef)
	if err != nil {
		logger.Warn("Autosync failed to list referrers", "cid", recordRef.GetCid(), "peer", peerID, "error", err)

		return
	}

	for _, desc := range descriptors {
		if _, ok := allowedReferrerTypes[desc.Type]; !ok {
			logger.Warn("Autosync skipped referrer: disallowed type",
				"cid", recordRef.GetCid(), "peer", peerID, "type", desc.Type)

			continue
		}

		referrer, err := m.transport.PullReferrer(ctx, peerID, recordRef, desc)
		if err != nil {
			logger.Warn("Autosync failed to pull referrer",
				"cid", recordRef.GetCid(), "peer", peerID, "referrer_cid", desc.Cid, "error", err)

			continue
		}

		if !m.verifyReferrer(referrer, recordRef.GetCid(), desc) {
			logger.Warn("Autosync rejected referrer: verification failed",
				"cid", recordRef.GetCid(), "peer", peerID, "referrer_cid", desc.Cid, "type", desc.Type)

			continue
		}

		if _, err := m.ingestor.ImportReferrer(ctx, recordRef.GetCid(), referrer); err != nil {
			logger.Warn("Autosync failed to ingest referrer",
				"cid", recordRef.GetCid(), "peer", peerID, "referrer_cid", desc.Cid, "error", err)

			continue
		}

		logger.Debug("Autosync ingested referrer",
			"cid", recordRef.GetCid(), "peer", peerID, "referrer_cid", desc.Cid, "type", desc.Type)
	}
}

// verifyReferrer checks that a pulled referrer belongs to the record and is
// self-consistent with the requested descriptor (type + referrer CID). This is
// fail-closed: any mismatch rejects the referrer.
func (m *Manager) verifyReferrer(referrer *corev1.RecordReferrer, recordCID string, desc rpc.ReferrerDescriptor) bool {
	if referrer == nil {
		return false
	}

	// Must belong to the record we are syncing.
	if referrer.GetRecordRef().GetCid() != recordCID {
		return false
	}

	// Must match the requested descriptor (guards against a peer relabeling).
	if referrer.GetType() != desc.Type {
		return false
	}

	if referrer.GetReferrerRef().GetCid() != desc.Cid {
		return false
	}

	return true
}
