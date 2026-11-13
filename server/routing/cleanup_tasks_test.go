// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	typesv1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1"
	coretypes "github.com/agntcy/dir/api/core/types"
	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	"github.com/stretchr/testify/require"
)

func TestCleanupStaleRemoteLabels(t *testing.T) {
	ctx := t.Context()
	r := newTestServer(t, ctx, nil)

	t.Cleanup(func() { _ = r.Stop() })

	cm := r.remote.cleanupManager
	cm.maxLabelAge = time.Hour

	localPeer := r.remote.server.Host().ID().String()
	label := types.Label("/skills/AI/ML")
	now := time.Now()

	put := func(cid, peerID string, lastSeen time.Time) datastore.Key {
		key := datastore.NewKey(BuildEnhancedLabelKey(label, cid, peerID))

		value, err := json.Marshal(types.LabelMetadata{Timestamp: lastSeen, LastSeen: lastSeen})
		require.NoError(t, err)
		require.NoError(t, r.remote.dstore.Put(ctx, key, value))

		return key
	}

	staleRemote := put("cid-stale", "remote-peer", now.Add(-2*time.Hour))
	freshRemote := put("cid-fresh", "remote-peer", now)
	staleLocal := put("cid-local", localPeer, now.Add(-2*time.Hour))

	require.NoError(t, cm.cleanupStaleRemoteLabels(ctx))

	for key, want := range map[datastore.Key]bool{
		staleRemote: false,
		freshRemote: true,
		staleLocal:  true,
	} {
		has, err := r.remote.dstore.Has(ctx, key)
		require.NoError(t, err)
		require.Equal(t, want, has, "key %s", key)
	}
}

// pullCountingStore counts Pull calls so tests can check which tasks read records.
type pullCountingStore struct {
	*mockStore

	pulls int
}

func (s *pullCountingStore) Pull(ctx context.Context, ref *corev1.RecordRef) (*corev1.Record, error) {
	s.pulls++

	return s.mockStore.Pull(ctx, ref)
}

func TestReprovideTasks(t *testing.T) {
	ctx := t.Context()
	r := newTestServer(t, ctx, nil)

	t.Cleanup(func() { _ = r.Stop() })

	store := &pullCountingStore{mockStore: newMockStore()}
	record := corev1.New(&typesv1.Record{
		Name:          "reprovide-agent",
		SchemaVersion: "1.0.0",
		Skills:        []*typesv1.Skill{{Name: "test_skill", Id: 1}},
	})
	_, err := store.Push(ctx, record)
	require.NoError(t, err)

	present := record.GetCid()
	orphan := "orphan-cid" // indexed locally but missing from storage

	for _, cid := range []string{present, orphan} {
		require.NoError(t, r.remote.dstore.Put(ctx, datastore.NewKey("/records/"+cid), []byte{}))
	}

	var provided, announced []string

	cm := r.remote.cleanupManager
	cm.storeAPI = store
	cm.provideFunc = func(_ context.Context, cid string) error {
		provided = append(provided, cid)

		return nil
	}
	cm.announceFunc = func(_ context.Context, rec coretypes.Record) error {
		announced = append(announced, rec.GetCid())

		return nil
	}

	t.Run("gossip reprovide announces stored records only", func(t *testing.T) {
		cm.reprovideGossip(ctx)

		require.Equal(t, []string{present}, announced)
		require.Empty(t, provided)
	})

	t.Run("dht reprovide provides CIDs without pulling and drops orphans", func(t *testing.T) {
		pulls := store.pulls

		cm.reprovideDHT(ctx)

		require.Equal(t, []string{present}, provided)
		require.Equal(t, pulls, store.pulls, "DHT reprovide must not pull records")

		has, err := r.remote.dstore.Has(ctx, datastore.NewKey("/records/"+orphan))
		require.NoError(t, err)
		require.False(t, has, "orphaned record key must be removed")
	})
}

// faultyQueryStore fails queries, or returns a single errored entry when errEntry is set.
type faultyQueryStore struct {
	types.Datastore

	errEntry bool
}

func (s faultyQueryStore) Query(_ context.Context, q query.Query) (query.Results, error) {
	if !s.errEntry {
		return nil, errors.New("query failed")
	}

	sent := false

	return query.ResultsFromIterator(q, query.Iterator{
		Next: func() (query.Result, bool) {
			if sent {
				return query.Result{}, false
			}

			sent = true

			return query.Result{Error: errors.New("bad entry")}, true
		},
	}), nil
}

func TestReprovideTaskFailures(t *testing.T) {
	ctx := t.Context()
	r := newTestServer(t, ctx, nil)

	t.Cleanup(func() { _ = r.Stop() })

	store := &pullCountingStore{mockStore: newMockStore()}
	record := corev1.New(&typesv1.Record{
		Name:          "failing-agent",
		SchemaVersion: "1.0.0",
		Skills:        []*typesv1.Skill{{Name: "test_skill", Id: 1}},
	})
	_, err := store.Push(ctx, record)
	require.NoError(t, err)

	undecodable := "undecodable-cid"
	store.data[undecodable] = &corev1.Record{} // pulls fine, fails to decode

	for _, cid := range []string{record.GetCid(), undecodable} {
		require.NoError(t, r.remote.dstore.Put(ctx, datastore.NewKey("/records/"+cid), []byte{}))
	}

	var provideCalls, announceCalls int

	cm := r.remote.cleanupManager
	cm.storeAPI = store
	cm.provideFunc = func(context.Context, string) error {
		provideCalls++

		return errors.New("provide failed")
	}
	cm.announceFunc = func(context.Context, coretypes.Record) error {
		announceCalls++

		return errors.New("announce failed")
	}

	// Failures are logged and the cycle moves on to the next record.
	cm.reprovideDHT(ctx)
	cm.reprovideGossip(ctx)
	require.Equal(t, 2, provideCalls)
	require.Equal(t, 1, announceCalls, "the undecodable record is skipped before announcing")

	// A failing datastore query, or an errored entry, never reaches the callbacks.
	for _, faulty := range []faultyQueryStore{
		{Datastore: r.remote.dstore},
		{Datastore: r.remote.dstore, errEntry: true},
	} {
		cm.dstore = faulty
		cm.reprovideDHT(ctx)
		cm.reprovideGossip(ctx)
	}

	require.Equal(t, 2, provideCalls)
	require.Equal(t, 1, announceCalls)
}

func TestReprovideTasksRunOnInterval(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	r := newTestServer(t, ctx, nil)

	t.Cleanup(func() { _ = r.Stop() })

	require.NoError(t, r.remote.dstore.Put(ctx, datastore.NewKey("/records/some-cid"), []byte{}))

	store := newMockStore()
	record := corev1.New(&typesv1.Record{Name: "tick-agent", SchemaVersion: "1.0.0"})
	store.data["some-cid"] = record

	provided := make(chan struct{}, 1)
	announced := make(chan struct{}, 1)

	cm := &CleanupManager{
		dstore:   r.remote.dstore,
		storeAPI: store,
		server:   r.remote.server,
		provideFunc: func(context.Context, string) error {
			select {
			case provided <- struct{}{}:
			default:
			}

			return nil
		},
		announceFunc: func(context.Context, coretypes.Record) error {
			select {
			case announced <- struct{}{}:
			default:
			}

			return nil
		},
		dhtReprovideInterval:    10 * time.Millisecond,
		gossipReprovideInterval: 10 * time.Millisecond,
	}

	var wg sync.WaitGroup

	wg.Add(2)

	go cm.StartDHTReprovideTask(ctx, &wg)
	go cm.StartGossipReprovideTask(ctx, &wg)

	<-provided
	<-announced

	cancel()
	wg.Wait()
}
