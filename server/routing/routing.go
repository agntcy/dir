// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:revive,unused
package routing

import (
	"context"
	"fmt"
	"path"
	"time"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	routingtypes "github.com/agntcy/dir/api/routing/v1alpha1"
	"github.com/agntcy/dir/server/routing/internal/p2p"
	"github.com/agntcy/dir/server/types"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/protocol"
	ocidigest "github.com/opencontainers/go-digest"
)

var (
	ProtocolPrefix     = "dir"
	ProtocolRendezvous = ProtocolPrefix + "/connect"
)

type routing struct {
	dstore types.Datastore
	server *p2p.Server
}

func New(ctx context.Context, opts types.APIOptions) (types.RoutingAPI, error) {
	// Create P2P server
	server, err := p2p.New(ctx,
		p2p.WithListenAddress(opts.Config().Routing.ListenAddress),
		p2p.WithBootstrapAddrs(opts.Config().Routing.BootstrapPeers),
		p2p.WithRefreshInterval(1*time.Second), // quick refresh, TODO: make configurable
		p2p.WithRandevous(ProtocolRendezvous),  // enable libp2p auto-discovery
		p2p.WithIdentityKeyPath(opts.Config().Routing.KeyPath),
		p2p.WithCustomDHTOpts(
			dht.Datastore(opts.Datastore()), // custom DHT datastore
			// dht.Validator(&validator{}),
			dht.NamespacedValidator("dir", &validator{}),    // custom namespace validator
			dht.ProtocolPrefix(protocol.ID(ProtocolPrefix)), // custom DHT protocol
			dht.ProviderStore(&peerstore{}),                 // provider store
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create p2p: %w", err)
	}

	return &routing{
		dstore: opts.Datastore(),
		server: server,
	}, nil
}

func (r *routing) Publish(ctx context.Context, object *coretypes.Object, local bool) error {
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

func (r *routing) List(ctx context.Context, req *routingtypes.ListRequest) (<-chan *routingtypes.ListResponse_Item, error) {
	ch := make(chan *routingtypes.ListResponse_Item)
	errCh := make(chan error, 1)

	metrics := make(Metrics)
	if err := metrics.load(ctx, r.dstore); err != nil {
		return nil, fmt.Errorf("failed to load metrics: %w", err)
	}

	// Get least common label
	leastCommonLabel := req.GetLabels()[0]
	for _, label := range req.GetLabels() {
		if metrics[label].Total < metrics[leastCommonLabel].Total {
			leastCommonLabel = label
		}
	}

	// Get filters for not least common labels
	var filters []query.Filter

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

			ch <- &routingtypes.ListResponse_Item{
				Record: &coretypes.ObjectRef{
					Type:   coretypes.ObjectType_OBJECT_TYPE_AGENT.String(),
					Digest: digest,
				},
			}
		}
	}()

	// TODO: Fetch items from the network via libp2p RPC if not found in the local datastore

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
