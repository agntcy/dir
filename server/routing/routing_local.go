// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var localLogger = logging.Logger("routing/local")

// operations performed locally.
type routeLocal struct {
	store       types.StoreAPI
	dstore      types.Datastore
	localPeerID string // Cached local peer ID for efficient filtering
}

func newLocal(store types.StoreAPI, dstore types.Datastore, localPeerID string) *routeLocal {
	return &routeLocal{
		store:       store,
		dstore:      dstore,
		localPeerID: localPeerID,
	}
}

func (r *routeLocal) Publish(ctx context.Context, record types.Record) error {
	if record == nil {
		return status.Error(codes.InvalidArgument, "record is required") //nolint:wrapcheck // Mock should return exact error without wrapping
	}

	cid := record.GetCid()
	if cid == "" {
		return status.Error(codes.InvalidArgument, "record has no CID") //nolint:wrapcheck
	}

	localLogger.Debug("Called local routing's Publish method", "cid", cid)

	metrics, err := loadMetrics(ctx, r.dstore)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to load metrics: %v", err)
	}

	batch, err := r.dstore.Batch(ctx)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to create batch: %v", err)
	}

	// the key where we will save the record
	recordKey := datastore.NewKey("/records/" + cid)

	// check if we have the record already
	// this is useful to avoid updating metrics and running the same operation multiple times
	recordExists, err := r.dstore.Has(ctx, recordKey)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to check if record exists: %v", err)
	}

	if recordExists {
		localLogger.Info("Skipping republish as record was already published", "cid", cid)

		return nil
	}

	// store record for later lookup
	if err := batch.Put(ctx, recordKey, nil); err != nil {
		return status.Errorf(codes.Internal, "failed to put record key: %v", err)
	}

	// Update metrics for all record labels and store them locally for queries
	// Note: This handles ALL local storage for both local-only and network scenarios
	// Network announcements are handled separately by routing_remote when peers are available
	labelList := types.GetLabelsFromRecord(record)
	for _, label := range labelList {
		// Create minimal metadata (PeerID and CID now in key)
		metadata := &types.LabelMetadata{
			Timestamp: time.Now(),
			LastSeen:  time.Now(),
		}

		// Serialize metadata to JSON
		metadataBytes, err := json.Marshal(metadata)
		if err != nil {
			return status.Errorf(codes.Internal, "failed to serialize label metadata: %v", err)
		}

		// Store with enhanced self-descriptive key: /skills/AI/CID123/Peer1
		enhancedKey := BuildEnhancedLabelKey(label, cid, r.localPeerID)

		labelKey := datastore.NewKey(enhancedKey)
		if err := batch.Put(ctx, labelKey, metadataBytes); err != nil {
			return status.Errorf(codes.Internal, "failed to put label key: %v", err)
		}

		metrics.increment(label)
	}

	err = batch.Commit(ctx)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to commit batch: %v", err)
	}

	// sync metrics
	err = metrics.update(ctx, r.dstore)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to update metrics: %v", err)
	}

	localLogger.Info("Successfully published record", "cid", cid)

	return nil
}

//nolint:cyclop
func (r *routeLocal) List(ctx context.Context, req *routingv1.ListRequest) (<-chan *routingv1.ListResponse, error) {
	localLogger.Debug("Called local routing's List method", "req", req)

	// ✅ DEFENSIVE: Deduplicate queries for consistent behavior (same as remote Search)
	originalQueries := req.GetQueries()
	deduplicatedQueries := deduplicateQueries(originalQueries)

	if len(originalQueries) != len(deduplicatedQueries) {
		localLogger.Info("Deduplicated list queries for consistent filtering",
			"originalCount", len(originalQueries), "deduplicatedCount", len(deduplicatedQueries))
	}

	// Output channel for results
	outCh := make(chan *routingv1.ListResponse)

	// Process in background with deduplicated queries
	go func() {
		defer close(outCh)

		r.listLocalRecords(ctx, deduplicatedQueries, req.GetLimit(), outCh)
	}()

	return outCh, nil
}

// listLocalRecords lists all local records with optional query filtering.
// Uses the simple and efficient approach: start with /records/ index, then filter by queries.
func (r *routeLocal) listLocalRecords(ctx context.Context, queries []*routingv1.RecordQuery, limit uint32, outCh chan<- *routingv1.ListResponse) {
	processedCount := 0
	limitInt := int(limit)

	// Step 1: Get all local record CIDs from /records/ index
	recordResults, err := r.dstore.Query(ctx, query.Query{
		Prefix: "/records/",
	})
	if err != nil {
		localLogger.Error("Failed to query local records", "error", err)

		return
	}
	defer recordResults.Close()

	// Step 2: For each local record, check if it matches ALL queries
	for result := range recordResults.Next() {
		if result.Error != nil {
			localLogger.Warn("Error reading record entry", "key", result.Key, "error", result.Error)

			continue
		}

		// Extract CID from record key: /records/CID123 → CID123
		cid := strings.TrimPrefix(result.Key, "/records/")
		if cid == "" {
			continue
		}

		// Check if this record matches all queries (AND relationship)
		if r.matchesAllQueries(ctx, cid, queries) {
			// Get labels for this record
			internalLabels := r.getRecordLabelsEfficiently(ctx, cid)

			// Convert []Label to []string for gRPC API boundary
			apiLabels := make([]string, len(internalLabels))
			for i, label := range internalLabels {
				apiLabels[i] = label.String()
			}

			// Send the response
			outCh <- &routingv1.ListResponse{
				RecordRef: &corev1.RecordRef{Cid: cid},
				Labels:    apiLabels,
			}

			processedCount++
			if limitInt > 0 && processedCount >= limitInt {
				break
			}
		}
	}

	localLogger.Debug("Completed List operation", "processed", processedCount, "queries", len(queries))
}

// matchesAllQueries checks if a record matches ALL provided queries (AND relationship).
// Uses shared query matching logic with local label retrieval strategy.
func (r *routeLocal) matchesAllQueries(ctx context.Context, cid string, queries []*routingv1.RecordQuery) bool {
	// Inject local label retrieval strategy into shared query matching logic
	return MatchesAllQueries(ctx, cid, queries, r.getRecordLabelsEfficiently)
}

// getRecordLabelsEfficiently gets labels for a record by extracting them from datastore keys.
// This completely avoids expensive Pull operations by using the fact that labels are stored as keys.
// This function is designed to be resilient - it never returns an error, only logs warnings.
func (r *routeLocal) getRecordLabelsEfficiently(ctx context.Context, cid string) []types.Label {
	var labelList []types.Label

	// Use shared namespace iteration function
	entries, err := QueryAllNamespaces(ctx, r.dstore)
	if err != nil {
		localLogger.Error("Failed to get namespace entries for labels", "cid", cid, "error", err)

		return labelList
	}

	// Find keys for this CID and local peer: "/skills/AI/ML/CID123/Peer1"
	for _, entry := range entries {
		// Parse the enhanced key to get components
		label, keyCID, keyPeerID, err := ParseEnhancedLabelKey(entry.Key)
		if err != nil {
			localLogger.Warn("Failed to parse enhanced label key", "key", entry.Key, "error", err)

			continue
		}

		// Check if this key matches our CID and is from local peer
		if keyCID == cid && keyPeerID == r.localPeerID {
			labelList = append(labelList, label)
		}
	}

	return labelList
}

func (r *routeLocal) Unpublish(ctx context.Context, record types.Record) error {
	if record == nil {
		return status.Error(codes.InvalidArgument, "record is required") //nolint:wrapcheck // Mock should return exact error without wrapping
	}

	cid := record.GetCid()
	if cid == "" {
		return status.Error(codes.InvalidArgument, "record has no CID") //nolint:wrapcheck
	}

	localLogger.Debug("Called local routing's Unpublish method", "cid", cid)

	// load metrics for the client
	metrics, err := loadMetrics(ctx, r.dstore)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to load metrics: %v", err)
	}

	batch, err := r.dstore.Batch(ctx)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to create batch: %v", err)
	}

	// get record key and remove record
	recordKey := datastore.NewKey("/records/" + cid)
	if err := batch.Delete(ctx, recordKey); err != nil {
		return status.Errorf(codes.Internal, "failed to delete record key: %v", err)
	}

	// keep track of all record labels
	labelList := types.GetLabelsFromRecord(record)

	for _, label := range labelList {
		// Delete enhanced key with CID and PeerID
		enhancedKey := BuildEnhancedLabelKey(label, cid, r.localPeerID)

		labelKey := datastore.NewKey(enhancedKey)
		if err := batch.Delete(ctx, labelKey); err != nil {
			return status.Errorf(codes.Internal, "failed to delete label key: %v", err)
		}

		metrics.decrement(label)
	}

	err = batch.Commit(ctx)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to commit batch: %v", err)
	}

	// sync metrics
	err = metrics.update(ctx, r.dstore)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to update metrics: %v", err)
	}

	localLogger.Info("Successfully unpublished record", "cid", cid)

	return nil
}
