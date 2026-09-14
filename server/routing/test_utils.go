// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:testifylint
package routing

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/agntcy/dir/server/config"
	"github.com/agntcy/dir/server/database"
	dbconfig "github.com/agntcy/dir/server/database/config"
	routingconfig "github.com/agntcy/dir/server/routing/config"
	"github.com/agntcy/dir/server/store"
	storeconfig "github.com/agntcy/dir/server/store/config"
	ociconfig "github.com/agntcy/dir/server/store/oci/config"
	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestDatabase returns a real SQLite-backed index. Local list is mostly
// filter translation, so it is worth testing against the query engine that
// runs in production rather than a hand-written matcher.
func newTestDatabase(t *testing.T) types.DatabaseAPI {
	t.Helper()

	db, err := database.New(dbconfig.Config{
		Type:   string(database.SQLite),
		SQLite: dbconfig.SQLiteConfig{Path: filepath.Join(t.TempDir(), "test.db")},
	})
	require.NoError(t, err)

	return db
}

// newTestServer starts a routing node. Pass a database only when the test
// exercises the peer query RPC; a node without one answers Unimplemented.
//
//nolint:revive
func newTestServer(t *testing.T, ctx context.Context, bootPeers []string, db types.DatabaseAPI) *route {
	t.Helper()

	refreshInterval := 1 * time.Second

	// define opts with faster refresh interval for testing
	// Use a unique temporary directory for each test to avoid datastore sharing
	opts := types.NewOptions(
		&config.Config{
			Store: storeconfig.Config{
				Provider: string(store.OCI),
				OCI: ociconfig.Config{
					LocalDir: t.TempDir(),
				},
			},
			Routing: routingconfig.Config{
				ListenAddress:   "/ip4/0.0.0.0/tcp/0",
				BootstrapPeers:  bootPeers,
				RefreshInterval: refreshInterval, // Fast refresh for testing
				DatastoreDir:    t.TempDir(),     // Use isolated BadgerDB for each test
			},
		},
	)

	// create new store
	s, err := store.New(opts)
	assert.NoError(t, err)

	r, err := New(ctx, s, db, opts)
	assert.NoError(t, err)

	// check the type assertion
	routeInstance, ok := r.(*route)
	assert.True(t, ok, "expected r to be of type *route")

	return routeInstance
}
