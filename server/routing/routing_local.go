// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"

	corev1 "github.com/agntcy/dir/api/core/v1"
	routingv1 "github.com/agntcy/dir/api/routing/v1"
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
	store  types.StoreAPI
	dstore types.Datastore
}

func newLocal(store types.StoreAPI, dstore types.Datastore) *routeLocal {
	return &routeLocal{
		store:  store,
		dstore: dstore,
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

	// keep track of all record skills
	labels := getLabels(record)
	for _, label := range labels {
		// Add key with cid
		labelKey := fmt.Sprintf("%s/%s", label, ref.GetCid())
		if err := batch.Put(ctx, datastore.NewKey(labelKey), nil); err != nil {
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
func (r *routeLocal) List(ctx context.Context, req *routingv1.ListRequest) (<-chan *routingv1.LegacyListResponse_Item, error) {
	localLogger.Debug("Called local routing's List method", "req", req)

	// dest to write the results on
	outCh := make(chan *routingv1.LegacyListResponse_Item)

	// load metrics for the client
	metrics, err := loadMetrics(ctx, r.dstore)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to load metrics: %v", err)
	}

	// if we sent an empty request, return us stats for the current peer
	if req.GetLegacyListRequest().GetRef() == nil && len(req.GetLegacyListRequest().GetLabels()) == 0 {
		go func(labels []string) {
			defer close(outCh)

			outCh <- &routingv1.LegacyListResponse_Item{
				Labels: labels,
				Peer: &routingv1.Peer{
					Id: "HOST",
				},
				LabelCounts: metrics.counts(),
			}
		}(metrics.labels())

		return outCh, nil
	}

	// validate request
	if len(req.GetLegacyListRequest().GetLabels()) == 0 {
		return nil, errors.New("no labels provided")
	}

	// get filters for not least common labels
	var filters []query.Filter

	leastCommonLabel := req.GetLegacyListRequest().GetLabels()[0]
	for _, label := range req.GetLegacyListRequest().GetLabels() {
		if metrics.Data[label].Total < metrics.Data[leastCommonLabel].Total {
			leastCommonLabel = label
		}
	}

	for _, label := range req.GetLegacyListRequest().GetLabels() {
		if label != leastCommonLabel {
			filters = append(filters, &labelFilter{
				dstore: r.dstore,
				ctx:    ctx,
				label:  label,
			})
		}
	}

	// start query
	res, err := r.dstore.Query(ctx, query.Query{
		Prefix:  leastCommonLabel,
		Filters: filters,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query datastore: %v", err)
	}

	// process items in the background, done in best effort mode
	go func() {
		defer close(outCh)

		processedRecordCids := make(map[string]struct{})

		for entry := range res.Next() {
			// read record cid from datastore key
			cid := path.Base(entry.Key)

			if _, ok := processedRecordCids[cid]; ok {
				continue
			}

			processedRecordCids[cid] = struct{}{}

			ref := &corev1.RecordRef{
				Cid: cid,
			}

			// lookup record
			_, err := r.store.Lookup(ctx, ref)
			if err != nil {
				localLogger.Error("failed to lookup record", "error", err)

				continue
			}

			// get record from peer
			record, err := r.store.Pull(ctx, ref)
			if err != nil {
				localLogger.Error("failed to pull record", "error", err)

				continue
			}

			labels := getLabels(record)

			// forward results back
			outCh <- &routingv1.LegacyListResponse_Item{
				Labels: labels,
				Peer: &routingv1.Peer{
					Id: "HOST",
				},
				Ref: ref,
			}
		}
	}()

	return outCh, nil
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
		// Delete key with cid
		labelKey := fmt.Sprintf("%s/%s", label, ref.GetCid())
		if err := batch.Delete(ctx, datastore.NewKey(labelKey)); err != nil {
			return status.Errorf(codes.Internal, "failed to delete skill key: %v", err)
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

var _ query.Filter = (*labelFilter)(nil)

//nolint:containedctx
type labelFilter struct {
	dstore types.Datastore
	ctx    context.Context

	label string
}

func (s *labelFilter) Filter(e query.Entry) bool {
	cid := path.Base(e.Key)
	has, _ := s.dstore.Has(s.ctx, datastore.NewKey(fmt.Sprintf("%s/%s", s.label, cid)))

	return has
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
		skills = append(skills, "/skills/"+skill.GetName())
	}

	labels = append(labels, skills...)

	// get record domains
	domainPrefix := "schema.oasf.agntcy.org/domains/"

	var domains []string

	for _, ext := range recordData.GetExtensions() {
		if strings.HasPrefix(ext.GetName(), domainPrefix) {
			domain := ext.GetName()[len(domainPrefix):]
			domains = append(domains, "/domains/"+domain)
		}
	}

	labels = append(labels, domains...)

	// get record features
	featuresPrefix := "schema.oasf.agntcy.org/features/"

	var features []string

	for _, ext := range recordData.GetExtensions() {
		if strings.HasPrefix(ext.GetName(), featuresPrefix) {
			feature := ext.GetName()[len(featuresPrefix):]
			features = append(features, "/features/"+feature)
		}
	}

	labels = append(labels, features...)

	return labels
}
