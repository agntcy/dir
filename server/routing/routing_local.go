// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	routingtypes "github.com/agntcy/dir/api/routing/v1alpha2"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	ocidigest "github.com/opencontainers/go-digest"
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

func (r *routeLocal) Publish(ctx context.Context, object *coretypes.Object) error {
	localLogger.Debug("Called local routing's Publish method", "object", object)

	ref := object.GetRef()
	if ref == nil {
		return status.Errorf(codes.InvalidArgument, "invalid object reference: %v", ref)
	}

	agent := object.GetAgent()
	if agent == nil {
		return status.Errorf(codes.InvalidArgument, "invalid agent object: %v", agent)
	}

	metrics, err := loadMetrics(ctx, r.dstore)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to load metrics: %v", err)
	}

	batch, err := r.dstore.Batch(ctx)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to create batch: %v", err)
	}

	// the key where we will save the agent
	agentKey := datastore.NewKey("/agents/" + ref.GetDigest())

	// check if we have the agent already
	// this is useful to avoid updating metrics and running the same operation multiple times
	agentExists, err := r.dstore.Has(ctx, agentKey)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to check if agent exists: %v", err)
	}

	if agentExists {
		localLogger.Info("Skipping republish as agent was already published", "ref", ref)

		return nil
	}

	// store agent for later lookup
	if err := batch.Put(ctx, agentKey, nil); err != nil {
		return status.Errorf(codes.Internal, "failed to put agent key: %v", err)
	}

	// keep track of all agent skills
	labels := getLabels(&coretypes.Agent{Agent: agent})
	for _, label := range labels {
		// Add key with digest
		agentLabelKey := fmt.Sprintf("%s/%s", label, ref.GetDigest())
		if err := batch.Put(ctx, datastore.NewKey(agentLabelKey), nil); err != nil {
			return status.Errorf(codes.Internal, "failed to put skill key: %v", err)
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

	localLogger.Info("Successfully published agent", "ref", ref)

	return nil
}

//nolint:cyclop
func (r *routeLocal) List(ctx context.Context, req *routingtypes.ListRequest) (<-chan *routingtypes.LegacyListResponse_Item, error) {
	localLogger.Debug("Called local routing's List method", "req", req)

	// dest to write the results on
	outCh := make(chan *routingtypes.LegacyListResponse_Item)

	// load metrics for the client
	metrics, err := loadMetrics(ctx, r.dstore)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to load metrics: %v", err)
	}

	// if we sent an empty request, return us stats for the current peer
	if req.GetLegacyListRequest().GetRecord() == nil && len(req.GetLegacyListRequest().GetLabels()) == 0 {
		go func(labels []string) {
			defer close(outCh)

			outCh <- &routingtypes.LegacyListResponse_Item{
				Labels: labels,
				Peer: &routingtypes.Peer{
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

		processedAgentDigests := make(map[string]struct{})

		for entry := range res.Next() {
			// read agent digest from datastore key
			digest, err := getAgentDigestFromKey(entry.Key)
			if err != nil {
				localLogger.Error("failed to get agent digest", "error", err)

				return
			}

			if _, ok := processedAgentDigests[digest]; ok {
				continue
			}

			processedAgentDigests[digest] = struct{}{}

			// lookup agent
			ref, err := r.store.Lookup(ctx, &coretypes.ObjectRef{
				Type:   coretypes.ObjectType_OBJECT_TYPE_AGENT.String(),
				Digest: digest,
			})
			if err != nil {
				localLogger.Error("failed to lookup agent", "error", err)

				continue
			}

			// get agent from peer
			object, err := r.store.Pull(ctx, ref)
			if err != nil {
				localLogger.Error("failed to pull agent", "error", err)

				continue
			}

			agent := &coretypes.Agent{}

			_, err = agent.LoadFromReader(object)
			if err != nil {
				localLogger.Error("failed to load agent", "error", err)

				continue
			}

			labels := getLabels(agent)

			// forward results back
			outCh <- &routingtypes.LegacyListResponse_Item{
				Labels: labels,
				Peer: &routingtypes.Peer{
					Id: "HOST",
				},
				Record: &coretypes.ObjectRef{
					Type:   coretypes.ObjectType_OBJECT_TYPE_AGENT.String(),
					Digest: digest,
				},
			}
		}
	}()

	return outCh, nil
}

func (r *routeLocal) Unpublish(ctx context.Context, object *coretypes.Object) error {
	localLogger.Debug("Called local routing's Unpublish method", "object", object)

	ref := object.GetRef()
	if ref == nil {
		return status.Errorf(codes.InvalidArgument, "invalid object reference: %v", ref)
	}

	agent := object.GetAgent()
	if agent == nil {
		return status.Errorf(codes.InvalidArgument, "invalid agent object: %v", agent)
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

	// get agent key and remove agent
	agentKey := datastore.NewKey("/agents/" + ref.GetDigest())
	if err := batch.Delete(ctx, agentKey); err != nil {
		return status.Errorf(codes.Internal, "failed to delete agent key: %v", err)
	}

	// keep track of all agent labels
	labels := getLabels(&coretypes.Agent{Agent: agent})

	for _, label := range labels {
		// Delete key with digest
		agentLabelKey := fmt.Sprintf("%s/%s", label, ref.GetDigest())
		if err := batch.Delete(ctx, datastore.NewKey(agentLabelKey)); err != nil {
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

	localLogger.Info("Successfully unpublished agent", "ref", ref)

	return nil
}

func getAgentDigestFromKey(k string) (string, error) {
	// Check if digest is valid
	digest := path.Base(k)
	if _, err := ocidigest.Parse(digest); err != nil {
		return "", fmt.Errorf("invalid digest: %s", digest)
	}

	return digest, nil
}

var _ query.Filter = (*labelFilter)(nil)

//nolint:containedctx
type labelFilter struct {
	dstore types.Datastore
	ctx    context.Context

	label string
}

func (s *labelFilter) Filter(e query.Entry) bool {
	digest := path.Base(e.Key)
	has, _ := s.dstore.Has(s.ctx, datastore.NewKey(fmt.Sprintf("%s/%s", s.label, digest)))

	return has
}

func getAgentSkills(agent *coretypes.Agent) []string {
	skills := make([]string, 0, len(agent.GetSkills()))
	for _, skill := range agent.GetSkills() {
		skills = append(skills, "/skills/"+skill.Key())
	}

	return skills
}

func getAgentDomains(agent *coretypes.Agent) []string {
	prefix := "schema.oasf.agntcy.org/domains/"

	var domains []string

	for _, ext := range agent.GetExtensions() {
		if strings.HasPrefix(ext.GetName(), prefix) {
			domain := ext.GetName()[len(prefix):]
			domains = append(domains, "/domains/"+domain)
		}
	}

	return domains
}

func getAgentFeatures(agent *coretypes.Agent) []string {
	prefix := "schema.oasf.agntcy.org/features/"

	var features []string

	for _, ext := range agent.GetExtensions() {
		if strings.HasPrefix(ext.GetName(), prefix) {
			feature := ext.GetName()[len(prefix):]
			features = append(features, "/features/"+feature)
		}
	}

	return features
}

func getLabels(agent *coretypes.Agent) []string {
	var labels []string

	// get agent skills
	skills := getAgentSkills(agent)
	labels = append(labels, skills...)

	// get agent domains
	domains := getAgentDomains(agent)
	labels = append(labels, domains...)

	// get agent features
	features := getAgentFeatures(agent)
	labels = append(labels, features...)

	return labels
}
