// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// nolint
package routing

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/server/config"
	routingconfig "github.com/agntcy/dir/server/routing/config"
	"github.com/agntcy/dir/server/store"
	ociconfig "github.com/agntcy/dir/server/store/oci/config"
	"github.com/agntcy/dir/server/types"
	"github.com/ipfs/go-datastore"
	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
)

func getObjectRef(a *coretypes.Agent) *coretypes.ObjectRef {
	raw, _ := json.Marshal(a) //nolint:errchkjson

	return &coretypes.ObjectRef{
		Type:        coretypes.ObjectType_OBJECT_TYPE_AGENT.String(),
		Digest:      digest.FromBytes(raw).String(),
		Size:        uint64(len(raw)),
		Annotations: a.GetAnnotations(),
	}
}

func toPtr[T any](v T) *T {
	return &v
}

func newTestServer(t *testing.T, ctx context.Context, bootPeers []string) *route {
	t.Helper()

	// override interval for routing table refresh
	realInterval := refreshInterval
	refreshInterval = 1 * time.Second

	defer func() {
		refreshInterval = realInterval
	}()

	// define opts
	opts := types.NewOptions(
		&config.Config{
			Provider: string(store.OCI),
			OCI: ociconfig.Config{
				LocalDir: t.TempDir(),
			},
			Routing: routingconfig.Config{
				ListenAddress:  "/ip4/0.0.0.0/tcp/0",
				BootstrapPeers: bootPeers,
			},
		},
		datastore.NewMapDatastore(),
	)

	// create new store
	s, err := store.New(opts)
	assert.NoError(t, err)

	// create example server
	r, err := New(ctx, s, opts)
	assert.NoError(t, err)

	return r.(*route)
}
