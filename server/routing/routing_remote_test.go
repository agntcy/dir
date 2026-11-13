// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"testing"
	"time"

	typesv1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1"
	coretypes "github.com/agntcy/dir/api/core/types"
	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/server/config"
	"github.com/agntcy/dir/server/policy"
	routingconfig "github.com/agntcy/dir/server/routing/config"
	"github.com/agntcy/dir/server/store"
	storeconfig "github.com/agntcy/dir/server/store/config"
	ociconfig "github.com/agntcy/dir/server/store/oci/config"
	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNewRejectsInvalidDHTParams(t *testing.T) {
	opts := types.NewOptions(
		&config.Config{
			Store: storeconfig.Config{
				Provider: string(store.OCI),
				OCI:      ociconfig.Config{LocalDir: t.TempDir()},
			},
			Routing: routingconfig.Config{
				ListenAddress:        "/ip4/0.0.0.0/tcp/0",
				DatastoreDir:         t.TempDir(),
				DHTReprovideInterval: RecordTTL, // must be lower than record_ttl
			},
		},
	)

	s, err := store.New(opts)
	require.NoError(t, err)

	_, err = New(t.Context(), s, nil, nil, policy.NewRecordGate(nil, nil), opts)
	require.ErrorContains(t, err, "invalid routing configuration")
}

// noCIDRecord is a record that reports no CID, which GossipSub refuses to publish.
type noCIDRecord struct{ coretypes.Record }

func (noCIDRecord) GetCid() string { return "" }

func newGossipTestServer(t *testing.T, bootPeers []string) *route {
	t.Helper()

	r := newTestServerWithConfig(t, t.Context(), bootPeers, func(c *routingconfig.Config) {
		c.GossipSub.Enabled = true
	})

	t.Cleanup(func() { _ = r.Stop() })

	return r
}

func TestProvideAndAnnounce(t *testing.T) {
	ctx := t.Context()
	r := newGossipTestServer(t, nil)
	require.NotNil(t, r.remote.pubsubManager)

	record := corev1.New(&typesv1.Record{
		Name:          "announce-agent",
		SchemaVersion: "1.0.0",
		Skills:        []*typesv1.Skill{{Name: "test_skill", Id: 1}},
	})
	adapter, err := record.Decode()
	require.NoError(t, err)

	t.Run("provide rejects an invalid CID", func(t *testing.T) {
		require.Equal(t, codes.InvalidArgument, status.Code(r.remote.provide(ctx, "not-a-cid")))
	})

	t.Run("provide fails without DHT peers and Publish returns it", func(t *testing.T) {
		require.Equal(t, codes.Internal, status.Code(r.remote.provide(ctx, record.GetCid())))
		require.Equal(t, codes.Internal, status.Code(r.remote.Publish(ctx, adapter)))
	})

	t.Run("announce publishes over GossipSub", func(t *testing.T) {
		require.NoError(t, r.remote.announce(ctx, adapter))
	})

	t.Run("announce returns GossipSub errors", func(t *testing.T) {
		require.Error(t, r.remote.announce(ctx, noCIDRecord{adapter}))
	})
}

func TestPublishSucceedsWhenGossipSubFails(t *testing.T) {
	ctx := t.Context()
	boot := newGossipTestServer(t, nil)
	r := newGossipTestServer(t, boot.remote.server.P2pAddrs())

	// wait for the DHT connection so the provide succeeds
	<-boot.remote.server.DHT().RefreshRoutingTable()
	time.Sleep(time.Second)

	record := corev1.New(&typesv1.Record{
		Name:          "gossip-fails-agent",
		SchemaVersion: "1.0.0",
		Skills:        []*typesv1.Skill{{Name: "test_skill", Id: 1}},
	})
	adapter, err := record.Decode()
	require.NoError(t, err)

	// A closed topic makes the GossipSub publish fail; the DHT announcement
	// already succeeded, so Publish only logs it.
	require.NoError(t, r.remote.pubsubManager.Close())
	require.NoError(t, r.remote.Publish(ctx, adapter))
}
