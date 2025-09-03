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
	"github.com/agntcy/dir/server/routing/validators"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/server/types/adapters"
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

func (r *routeLocal) Publish(ctx context.Context, ref *corev1.RecordRef, record *corev1.Record) error {
	localLogger.Debug("Called local routing's Publish method", "ref", ref, "record", record)

	// Validate input parameters
	if ref == nil {
		return status.Error(codes.InvalidArgument, "record reference is required") //nolint:wrapcheck // Mock should return exact error without wrapping
	}

	if record == nil {
		return status.Error(codes.InvalidArgument, "record is required") //nolint:wrapcheck // Mock should return exact error without wrapping
	}

	metrics, err := loadMetrics(ctx, r.dstore)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to load metrics: %v", err)
	}

	batch, err := r.dstore.Batch(ctx)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to create batch: %v", err)
	}

	// the key where we will save the record
	recordKey := datastore.NewKey("/records/" + ref.GetCid())

	// check if we have the record already
	// this is useful to avoid updating metrics and running the same operation multiple times
	recordExists, err := r.dstore.Has(ctx, recordKey)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to check if record exists: %v", err)
	}

	if recordExists {
		localLogger.Info("Skipping republish as record was already published", "ref", ref)

		return nil
	}

	// store record for later lookup
	if err := batch.Put(ctx, recordKey, nil); err != nil {
		return status.Errorf(codes.Internal, "failed to put record key: %v", err)
	}

	// Update metrics for all record labels and store them locally for queries
	// Note: This handles ALL local storage for both local-only and network scenarios
	// Network announcements are handled separately by routing_remote when peers are available
	labels := getLabels(record)
	for _, label := range labels {
		// Create minimal metadata (PeerID and CID now in key)
		metadata := &LabelMetadata{
			Timestamp: time.Now(),
			LastSeen:  time.Now(),
		}

		// Serialize metadata to JSON
		metadataBytes, err := json.Marshal(metadata)
		if err != nil {
			return status.Errorf(codes.Internal, "failed to serialize label metadata: %v", err)
		}

		// Store with enhanced self-descriptive key: /skills/AI/CID123/Peer1
		enhancedKey := BuildEnhancedLabelKey(label, ref.GetCid(), r.localPeerID)

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

	localLogger.Info("Successfully published record", "ref", ref)

	return nil
}

//nolint:cyclop
func (r *routeLocal) List(ctx context.Context, req *routingv1.ListRequest) (<-chan *routingv1.ListResponse, error) {
	localLogger.Debug("Called local routing's List method", "req", req)

	// Output channel for results
	outCh := make(chan *routingv1.ListResponse)

	// Process in background
	go func() {
		defer close(outCh)
		r.listLocalRecords(ctx, req.GetQueries(), req.GetLimit(), outCh)
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
			labels := r.getRecordLabelsEfficiently(ctx, cid)

			// Send the response
			outCh <- &routingv1.ListResponse{
				RecordRef: &corev1.RecordRef{Cid: cid},
				Labels:    labels,
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
func (r *routeLocal) matchesAllQueries(ctx context.Context, cid string, queries []*routingv1.RecordQuery) bool {
	if len(queries) == 0 {
		return true // No filters = match everything
	}

	// Get all labels for this record
	recordLabels := r.getRecordLabelsEfficiently(ctx, cid)

	// ALL queries must match (AND relationship)
	for _, query := range queries {
		if !r.queryMatchesLabels(query, recordLabels) {
			return false
		}
	}

	return true
}

// queryMatchesLabels checks if a query matches against the record's labels.
func (r *routeLocal) queryMatchesLabels(query *routingv1.RecordQuery, labels []string) bool {
	switch query.GetType() {
	case routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL:
		// Check if any skill label matches the query
		skillPrefix := validators.NamespaceSkills.Prefix()
		targetSkill := skillPrefix + query.GetValue()

		for _, label := range labels {
			// Exact match: /skills/category1/class1 matches "category1/class1"
			if label == targetSkill {
				return true
			}
			// Prefix match: /skills/category2/class2 matches "category2"
			if strings.HasPrefix(label, targetSkill+"/") {
				return true
			}
		}

		return false

	case routingv1.RecordQueryType_RECORD_QUERY_TYPE_LOCATOR:
		// Check if any locator label matches the query (consistent with skills)
		locatorPrefix := validators.NamespaceLocators.Prefix()
		targetLocator := locatorPrefix + query.GetValue()

		for _, label := range labels {
			// Exact match: /locators/docker-image matches "docker-image"
			if label == targetLocator {
				return true
			}
		}

		return false

	case routingv1.RecordQueryType_RECORD_QUERY_TYPE_UNSPECIFIED:
		// Unspecified queries match everything
		return true

	default:
		localLogger.Warn("Unknown query type", "type", query.GetType())

		return false
	}
}

// getRecordLabelsEfficiently gets labels for a record by extracting them from datastore keys.
// This completely avoids expensive Pull operations by using the fact that labels are stored as keys.
// This function is designed to be resilient - it never returns an error, only logs warnings.
func (r *routeLocal) getRecordLabelsEfficiently(ctx context.Context, cid string) []string {
	var labels []string

	// Query each namespace to find labels for this CID
	namespaces := []string{
		validators.NamespaceSkills.Prefix(),
		validators.NamespaceDomains.Prefix(),
		validators.NamespaceFeatures.Prefix(),
		validators.NamespaceLocators.Prefix(),
	}

	for _, namespace := range namespaces {
		// Query all keys in this namespace
		results, err := r.dstore.Query(ctx, query.Query{
			Prefix: namespace,
		})
		if err != nil {
			localLogger.Warn("Failed to query namespace for labels", "namespace", namespace, "cid", cid, "error", err)

			continue
		}

		// Find keys for this CID and local peer: "/skills/AI/ML/CID123/Peer1"
		for result := range results.Next() {
			if result.Error != nil {
				localLogger.Warn("Error reading label key", "key", result.Key, "error", result.Error)

				continue
			}

			// Parse the enhanced key to get components
			label, keyCID, keyPeerID, err := ParseEnhancedLabelKey(result.Key)
			if err != nil {
				localLogger.Warn("Failed to parse enhanced label key", "key", result.Key, "error", err)

				continue
			}

			// Check if this key matches our CID and is from local peer
			if keyCID == cid && keyPeerID == r.localPeerID {
				labels = append(labels, label)
			}
		}

		results.Close()
	}

	return labels
}

func (r *routeLocal) Unpublish(ctx context.Context, ref *corev1.RecordRef, record *corev1.Record) error {
	localLogger.Debug("Called local routing's Unpublish method", "ref", ref, "record", record)

	// Validate input parameters
	if ref == nil {
		return status.Error(codes.InvalidArgument, "record reference is required") //nolint:wrapcheck // Mock should return exact error without wrapping
	}

	if record == nil {
		return status.Error(codes.InvalidArgument, "record is required") //nolint:wrapcheck // Mock should return exact error without wrapping
	}

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
	recordKey := datastore.NewKey("/records/" + ref.GetCid())
	if err := batch.Delete(ctx, recordKey); err != nil {
		return status.Errorf(codes.Internal, "failed to delete record key: %v", err)
	}

	// keep track of all record labels
	labels := getLabels(record)

	for _, label := range labels {
		// Delete enhanced key with CID and PeerID
		enhancedKey := BuildEnhancedLabelKey(label, ref.GetCid(), r.localPeerID)

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

	localLogger.Info("Successfully unpublished record", "ref", ref)

	return nil
}

func getLabels(record *corev1.Record) []string {
	// Use adapter pattern to get version-agnostic access to record data
	adapter := adapters.NewRecordAdapter(record)

	recordData := adapter.GetRecordData()
	if recordData == nil {
		localLogger.Error("failed to get record data")

		return nil
	}

	var labels []string

	// get record skills
	skills := make([]string, 0, len(recordData.GetSkills()))
	for _, skill := range recordData.GetSkills() {
		skills = append(skills, validators.NamespaceSkills.Prefix()+skill.GetName())
	}

	labels = append(labels, skills...)

	// get record domains
	var domains []string

	for _, ext := range recordData.GetExtensions() {
		if strings.HasPrefix(ext.GetName(), validators.DomainSchemaPrefix) {
			domain := ext.GetName()[len(validators.DomainSchemaPrefix):]
			domains = append(domains, validators.NamespaceDomains.Prefix()+domain)
		}
	}

	labels = append(labels, domains...)

	// get record features
	var features []string

	for _, ext := range recordData.GetExtensions() {
		if strings.HasPrefix(ext.GetName(), validators.FeaturesSchemaPrefix) {
			feature := ext.GetName()[len(validators.FeaturesSchemaPrefix):]
			features = append(features, validators.NamespaceFeatures.Prefix()+feature)
		}
	}

	labels = append(labels, features...)

	// get record locators
	locators := make([]string, 0, len(recordData.GetLocators()))

	for _, locator := range recordData.GetLocators() {
		locators = append(locators, validators.NamespaceLocators.Prefix()+locator.GetType())
	}

	labels = append(labels, locators...)

	return labels
}
