// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// nolint
package routing

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"path"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	routingtypes "github.com/agntcy/dir/api/routing/v1alpha1"
	"github.com/agntcy/dir/server/types"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	ocidigest "github.com/opencontainers/go-digest"
)

// operations performed locally
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

func (r *routeLocal) Publish(ctx context.Context, object *coretypes.Object, local bool) error {
	ref := object.GetRef()
	if ref == nil {
		return fmt.Errorf("invalid object reference: %v", ref)
	}

	agent := object.GetAgent()
	if agent == nil {
		return fmt.Errorf("invalid agent object: %v", agent)
	}

	metrics := make(Metrics)
	if err := metrics.load(ctx, r.dstore); err != nil {
		return fmt.Errorf("failed to load metrics: %w", err)
	}

	batch, err := r.dstore.Batch(ctx)
	if err != nil {
		return fmt.Errorf("failed to create batch: %w", err)
	}

	for _, skill := range agent.GetSkills() {
		key := "/skills/" + skill.Key()

		// Add key with digest
		agentSkillKey := fmt.Sprintf("%s/%s", key, ref.GetDigest())
		if err := batch.Put(ctx, datastore.NewKey(agentSkillKey), nil); err != nil {
			return fmt.Errorf("failed to put skill key: %w", err)
		}

		metrics.increment(key)
	}

	err = batch.Commit(ctx)
	if err != nil {
		return fmt.Errorf("failed to commit batch: %w", err)
	}

	err = metrics.update(ctx, r.dstore)
	if err != nil {
		return fmt.Errorf("failed to update metrics: %w", err)
	}

	// TODO: Publish items to the network via libp2p RPC

	return nil
}

func (r *routeLocal) List(ctx context.Context, req *routingtypes.ListRequest) (<-chan *routingtypes.ListResponse_Item, error) {
	ch := make(chan *routingtypes.ListResponse_Item)
	errCh := make(chan error, 1)

	metrics := make(Metrics)
	if err := metrics.load(ctx, r.dstore); err != nil {
		return nil, fmt.Errorf("failed to load metrics: %w", err)
	}

	if len(req.GetLabels()) == 0 {
		return nil, fmt.Errorf("no labels provided")
	}

	// Get filters for not least common labels
	var filters []query.Filter
	leastCommonLabel := req.GetLabels()[0]
	for _, label := range req.GetLabels() {
		if metrics[label].Total < metrics[leastCommonLabel].Total {
			leastCommonLabel = label
		}
	}
	for _, label := range req.GetLabels() {
		if label != leastCommonLabel {
			filters = append(filters, &labelFilter{
				dstore: r.dstore,
				ctx:    ctx,
				label:  label,
			})
		}
	}

	go func() {
		defer close(ch)
		defer close(errCh)

		res, err := r.dstore.Query(ctx, query.Query{
			Prefix:  leastCommonLabel,
			Filters: filters,
		})
		if err != nil {
			errCh <- err

			return
		}

		for entry := range res.Next() {
			digest, err := getAgentDigestFromKey(entry.Key)
			if err != nil {
				errCh <- err

				return
			}

			// get agent from peer
			object, err := r.store.Pull(ctx, &coretypes.ObjectRef{
				Type:   coretypes.ObjectType_OBJECT_TYPE_AGENT.String(),
				Digest: digest,
			})
			if err != nil {
				log.Printf("failed to pull agent: %v", err)

				continue
			}

			// read all
			data, _ := io.ReadAll(object)

			var agent *coretypes.Agent
			if err := json.Unmarshal(data, &agent); err != nil {
				log.Printf("failed to unmarshal agent: %v", err)
				continue
			}

			// get agent skills
			var skills []string
			for _, skill := range agent.GetSkills() {
				skills = append(skills, skill.Key())
			}

			ch <- &routingtypes.ListResponse_Item{
				Labels: skills,
				Peer: &routingtypes.Peer{
					Id: "local",
				},
				Record: &coretypes.ObjectRef{
					Type:   coretypes.ObjectType_OBJECT_TYPE_AGENT.String(),
					Digest: digest,
				},
			}
		}
	}()

	select {
	case err := <-errCh:
		return nil, err
	default:
		return ch, nil
	}
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
