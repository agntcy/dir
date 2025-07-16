// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:testifylint
package routing

import (
	"context"
	"testing"
	"time"

	"github.com/agntcy/dir/server/config"
	routingconfig "github.com/agntcy/dir/server/routing/config"
	"github.com/agntcy/dir/server/store"
	ociconfig "github.com/agntcy/dir/server/store/oci/config"
	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
)

func toPtr[T any](v T) *T {
	return &v
}

//nolint:revive
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
	)

	// create new store
	s, err := store.New(opts)
	assert.NoError(t, err)

	// create example server
	r, err := New(ctx, s, opts)
	assert.NoError(t, err)

	// check the type assertion
	routeInstance, ok := r.(*route)
	assert.True(t, ok, "expected r to be of type *route")

	return routeInstance
}
