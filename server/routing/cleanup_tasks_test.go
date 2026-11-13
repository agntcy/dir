// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/agntcy/dir/server/types"
	"github.com/ipfs/go-datastore"
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
