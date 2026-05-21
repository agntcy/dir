// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:testifylint
package routing

import (
	"context"
	"testing"
	"time"

	"github.com/agntcy/dir/config"
	"github.com/agntcy/dir/server/store"
	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
)

const testLocalPeerID = "local-peer"

//nolint:revive
func newTestServer(t *testing.T, ctx context.Context, bootPeers []string) *route {
	t.Helper()

	refreshInterval := 1 * time.Second

	// define opts with faster refresh interval for testing
	// Use a unique temporary directory for each test to avoid datastore sharing
	opts := types.NewOptions(
		&config.Config{
			Registry: config.Registry{
				LocalDir: t.TempDir(),
			},
			APIServer: config.APIServer{
				Routing: config.Routing{
					ListenAddress:   "/ip4/0.0.0.0/tcp/0",
					BootstrapPeers:  bootPeers,
					RefreshInterval: refreshInterval,
					DatastoreDir:    t.TempDir(),
				},
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
