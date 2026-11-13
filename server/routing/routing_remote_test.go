// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"testing"

	"github.com/agntcy/dir/server/config"
	"github.com/agntcy/dir/server/policy"
	routingconfig "github.com/agntcy/dir/server/routing/config"
	"github.com/agntcy/dir/server/store"
	storeconfig "github.com/agntcy/dir/server/store/config"
	ociconfig "github.com/agntcy/dir/server/store/oci/config"
	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/require"
)

func TestNewRejectsInvalidDHTParams(t *testing.T) {
	opts := types.NewOptions(
		&config.Config{
			Store: storeconfig.Config{
				Provider: string(store.OCI),
				OCI:      ociconfig.Config{LocalDir: t.TempDir()},
			},
			Routing: routingconfig.Config{
				ListenAddress:     "/ip4/0.0.0.0/tcp/0",
				DatastoreDir:      t.TempDir(),
				RepublishInterval: RecordTTL, // must be lower than record_ttl
			},
		},
	)

	s, err := store.New(opts)
	require.NoError(t, err)

	_, err = New(t.Context(), s, nil, nil, policy.NewRecordGate(nil, nil), opts)
	require.ErrorContains(t, err, "invalid routing configuration")
}
