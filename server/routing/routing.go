// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:revive,unused
package routing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"path"
	"time"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	routingtypes "github.com/agntcy/dir/api/routing/v1alpha1"
	"github.com/agntcy/dir/server/routing/internal/p2p"
	"github.com/agntcy/dir/server/routing/rpc"
	"github.com/agntcy/dir/server/types"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	"github.com/libp2p/go-libp2p/core/peer"
	ocidigest "github.com/opencontainers/go-digest"
)

var (
	ProtocolPrefix     = "dir"
	ProtocolRendezvous = "dir/connect"

	// refresh interval for DHT routing tables
	refreshInterval = 5 * time.Minute
)

// routing only implements local operations for the routing layer (ie what i have locally stored).
// the networking operation is handled by the handler which runs this across
type routing struct {
	storeAPI types.StoreAPI
	dstore   types.Datastore
	server   *p2p.Server
	service  *rpc.Service
	notifyCh chan *notification
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

	select {
	case err := <-errCh:
		return nil, err
	default:
		return ch, nil
	}
}

func (r *routing) List(ctx context.Context, req *routingtypes.ListRequest) (<-chan *routingtypes.ListResponse_Item, error) {
	// figure out the request params
	if req.Local != nil && *req.Local {
		return r.listLocal(ctx, req)
	}

	// list data from remote for a given peer
	if req.Peer != nil {
		// force the peer to return its local data
		req.Local = new(bool)
		*req.Local = true

		// TODO: handle error
		// we dont do anythin with the error for now, it can only time out
		resp, _ := r.service.List(ctx, []peer.ID{peer.ID(req.Peer.Id)}, req)
		return resp, nil
	}

	// get specific agent from all remote peers hosting it
	if req.GetRecord() != nil {
		// get object CID
		cid, err := req.GetRecord().GetCID()
		if err != nil {
			return nil, fmt.Errorf("failed to get object CID: %w", err)
		}

		// announce to DHT
		provs, err := r.server.DHT().FindProviders(ctx, cid)
		if err != nil {
			return nil, fmt.Errorf("failed to find object providers: %w", err)
		}
		if len(provs) == 0 {
			return nil, fmt.Errorf("no providers found for object: %s", cid)
		}

		// stream results back
		// NOTE: we list from each provider
		resCh := make(chan *routingtypes.ListResponse_Item, 100)
		go func(provs []peer.AddrInfo, ref coretypes.ObjectRef) {
			defer close(resCh)
			for _, prov := range provs {
				// get agent from peer
				object, err := r.service.Pull(ctx, prov.ID, &ref)
				if err != nil {
					log.Printf("failed to pull agent: %v", err)
					continue
				}
				agent := object.Agent

				// get agent skills
				var skills []string
				for _, skill := range agent.Skills {
					skills = append(skills, skill.Key())
				}

				// peer addrs to string
				var addrs []string
				for _, addr := range prov.Addrs {
					addrs = append(addrs, addr.String())
				}

				// send back to caller
				resCh <- &routingtypes.ListResponse_Item{
					Record: object.Ref,
					Labels: skills,
					Peer: &routingtypes.Peer{
						Id:    prov.ID.String(),
						Addrs: addrs,
					},
				}
			}
		}(provs, *req.GetRecord())
		return resCh, nil
	}

	// run a query across peers, keep forwarding until we exhaust the hops
	// TODO: this is a naive implementation, reconsider better selection of peers,
	// and scheduling.
	// fix number of hops
	if req.MaxHops == nil {
		req.MaxHops = new(uint32)
		*req.MaxHops = 5
	}
	if *req.MaxHops > 20 {
		return nil, errors.New("max hops exceeded")
	}

	resCh := make(chan *routingtypes.ListResponse_Item, 100)
	go func(ctx context.Context, req *routingtypes.ListRequest) {
		defer close(resCh)
		peers := r.server.Host().Peerstore().Peers()

		// get data from peers (list what each peer has)
		localReq := &routingtypes.ListRequest{
			Peer:    req.Peer,
			Labels:  req.Labels,
			Record:  req.Record,
			MaxHops: req.MaxHops,
			Local:   toPtr(true),
		}
		resp, err := r.service.List(ctx, peers, localReq)
		if err != nil {
			log.Printf("failed to list: %v", err)
			return
		}

		// stream local data results from each peer.
		// we need to drop some peers from querying here!
		for item := range resp {
			// TODO: filter what to return back
			resCh <- item // forward results back
		}

		//  check forwarding
		*req.MaxHops = *req.MaxHops - 1
		if req.MaxHops != nil && *req.MaxHops == 0 {
			// done
			return
		}

		// forward requests further
		resp, err = r.service.List(ctx, peers, req)
		if err != nil {
			log.Printf("failed to list: %v", err)
			return
		}

		// stream sub-query results
		for item := range resp {
			// TODO: filter what to return back
			resCh <- item // forward results back
		}
	}(ctx, req)

	return resCh, nil
}

func (r *routing) listLocal(ctx context.Context, req *routingtypes.ListRequest) (<-chan *routingtypes.ListResponse_Item, error) {
	resCh := make(chan *routingtypes.ListResponse_Item, 100)

	if req.GetRecord() != nil {
		// get agent from store
		meta, err := r.storeAPI.Lookup(ctx, req.GetRecord())
		if err != nil {
			return nil, fmt.Errorf("failed to lookup agent: %v", err)
		}

		// get agent from store
		reader, err := r.storeAPI.Pull(ctx, req.GetRecord())
		if err != nil {
			return nil, fmt.Errorf("failed to pull agent: %v", err)
		}

		// read all
		data, err := io.ReadAll(reader)
		if err != nil {
			return nil, fmt.Errorf("failed to read agent: %v", err)
		}

		// get agent object
		var agent *coretypes.Agent
		if err := json.Unmarshal(data, &agent); err != nil {
			return nil, fmt.Errorf("failed to unmarshal agent: %w", err)
		}

		// get agent skills
		var skills []string
		for _, skill := range agent.Skills {
			skills = append(skills, skill.Key())
		}

		// send back to caller
		resCh <- &routingtypes.ListResponse_Item{
			Record: meta,
			Labels: skills,
			Peer: &routingtypes.Peer{
				Id: peer.ID(r.server.Host().ID()).String(),
			},
		}
		close(resCh) // send close after sending the result
		return resCh, nil
	}

	// list all agents with specific labels
	if req.Labels != nil {
		// make filters
		var filters map[string]struct{}
		if len(req.Labels) > 0 {
			filters = make(map[string]struct{})
			for _, label := range req.Labels {
				filters[label] = struct{}{}
			}
		}

		// get labels with given skill
		// check if it has first
		stats, err := r.getPeerSkills(ctx, r.server.Host().ID().String(), filters)
		if err != nil {
			return nil, fmt.Errorf("failed to get peer stats: %w", err)
		}

		// find minimum value in stats
		// this will be our reference point for data filtration
		// we match ALL-OF!
		var minStats uint64
		var minSkill string
		for key, val := range stats {
			if val < minStats && val > 0 {
				minStats = val
				minSkill = key
			}
		}

		// check if we matched anything
		if minSkill == "" || minStats == 0 {
			return nil, errors.New("no matching skills found")
		}

		// start matching things
		delete(filters, minSkill)
		res, err := r.dstore.Query(ctx, query.Query{
			Prefix:   fmt.Sprintf("/skills/%s/", minSkill),
			KeysOnly: true,
			Filters: []query.Filter{&recordFilter{
				skills: filters,
				store:  r.dstore,
				ctx:    ctx,
			}},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to query datastore: %w", err)
		}

		// stream results back
		go func() {
			defer close(resCh)
			for entry := range res.Next() {
				if entry.Error != nil {
					log.Printf("failed to list: %v", entry.Error)
					continue
				}

				// get digest from key
				digest, err := getAgentDigestFromKey(entry.Key)
				if err != nil {
					log.Printf("failed to get agent digest: %v", err)
					return
				}

				// get agent from store
				meta, err := r.storeAPI.Lookup(ctx, &coretypes.ObjectRef{Digest: digest})
				if err != nil {
					log.Printf("failed to lookup agent: %v", err)
					return
				}

				// send back to caller
				resCh <- &routingtypes.ListResponse_Item{
					Record: meta,
					Labels: req.Labels,
					Peer: &routingtypes.Peer{
						Id: peer.ID(r.server.Host().ID()).String(),
					},
				}
			}
		}()
		return resCh, nil
	}

	return nil, errors.New("invalid request")
}

func (r *routing) handleNotify(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// check if anything on notify
procLoop:
	for {
		select {
		case <-ctx.Done():
			return
		case notif := <-r.notifyCh:
			// lookup
			meta, err := r.service.Lookup(ctx, notif.Peer.ID, notif.Ref)
			if err != nil {
				log.Printf("failed to lookup: %v", err)
				continue procLoop
			}

			// fetch model directly from peer, update table, and drop it
			// TODO: resolve this once we have a List API working
			object, err := r.service.Pull(ctx, notif.Peer.ID, notif.Ref)
			if err != nil {
				log.Printf("failed to pull: %v", err)
				continue procLoop
			}
			agent := object.Agent

			// extract skills
			var skills []string
			for _, skill := range agent.Skills {
				skills = append(skills, skill.Key())
			}

			// update peer routing table
			if err := r.updatePeerSkills(ctx, notif.Peer.ID.String(), skills); err != nil {
				log.Printf("failed to update peer skills: %v", err)
				continue procLoop
			}

			log.Printf("successfully processed agent %v", meta)
		}
	}
}

func (r *routing) updatePeerSkills(ctx context.Context, peerID string, skills []string) error {
	statsKey := datastore.NewKey("/stats/peers/" + peerID)

	// check if it has first
	prevStats, err := r.getPeerSkills(ctx, peerID, nil)
	if err != nil {
		return fmt.Errorf("failed to get peer stats: %w", err)
	}

	// update with new data
	for _, skill := range skills {
		prevStats[skill] += 1
	}

	// write updated data
	statsData, err := json.Marshal(prevStats)
	if err != nil {
		return fmt.Errorf("failed to marshal stats: %w", err)
	}

	// write to datastore
	if err := r.dstore.Put(ctx, statsKey, statsData); err != nil {
		return fmt.Errorf("failed to put stats: %w", err)
	}

	return nil
}

func (r *routing) getPeerSkills(ctx context.Context, peerID string, filters map[string]struct{}) (skillStats, error) {
	statsKey := datastore.NewKey("/stats/peers/" + peerID)

	// check if it has first
	has, err := r.dstore.Has(ctx, statsKey)
	if err != nil {
		return nil, fmt.Errorf("failed to check if key exists: %w", err)
	}

	// read data for this peer if exists
	stats := skillStats{}
	if has {
		statsData, err := r.dstore.Get(ctx, statsKey)
		if err != nil {
			return nil, fmt.Errorf("failed to get stats: %w", err)
		}

		// process
		err = json.Unmarshal(statsData, &stats)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal stats: %w", err)
		}

		return stats, nil
	}

	// find intersection if filters requested
	if len(filters) > 0 {
		for filterKey := range filters {
			_, ok := stats[filterKey]
			if !ok {
				delete(stats, filterKey)
			}
		}
	}

	return stats, nil
}

type skillStats map[string]uint64

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

type recordFilter struct {
	skills map[string]struct{}
	store  types.Datastore
	ctx    context.Context
}

func (f *recordFilter) Filter(e query.Entry) bool {
	// check if key contains all the skills
	for skill := range f.skills {
		ok, err := f.store.Has(f.ctx, datastore.NewKey(fmt.Sprintf("/skills/%s/%s", skill, path.Base(e.Key))))
		if err != nil {
			return false
		}
		return ok
	}
	return true
}
