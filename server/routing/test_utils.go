// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:testifylint
package routing

import (
	"context"
	"testing"
	"time"

	"github.com/agntcy/dir/server/config"
	rankingcfg "github.com/agntcy/dir/server/ranking/config"
	routingconfig "github.com/agntcy/dir/server/routing/config"
	"github.com/agntcy/dir/server/store"
	storeconfig "github.com/agntcy/dir/server/store/config"
	ociconfig "github.com/agntcy/dir/server/store/oci/config"
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

	// create example server (nil DB; remote search will use defaults-only scoring)
	r, err := New(ctx, s, nil, rankingcfg.Config{
		Weights: rankingcfg.Weights{
			QueryRelevance: rankingcfg.DefaultQueryRelevanceWeight,
			Trust:          rankingcfg.DefaultTrustWeight,
			Popularity:     rankingcfg.DefaultPopularityWeight,
			Completeness:   rankingcfg.DefaultCompletenessWeight,
			Freshness:      rankingcfg.DefaultFreshnessWeight,
			SchemaRecency:  rankingcfg.DefaultSchemaRecencyWeight,
		},
		TrustSplit: rankingcfg.TrustSplit{
			Signed:       rankingcfg.DefaultTrustSplitSigned,
			SigVerified:  rankingcfg.DefaultTrustSplitSigVerified,
			NameVerified: rankingcfg.DefaultTrustSplitNameVerified,
		},
		Freshness:     rankingcfg.Freshness{HalfLifeDays: rankingcfg.DefaultFreshnessHalfLifeDays},
		Popularity:    rankingcfg.Popularity{SaturationAtProviders: rankingcfg.DefaultPopularitySaturation},
		Completeness:  rankingcfg.Completeness{SaturationAtEntries: rankingcfg.DefaultCompletenessSaturation},
		MaxCandidates: rankingcfg.DefaultMaxCandidates,
	}, opts)
	assert.NoError(t, err)

	// check the type assertion
	routeInstance, ok := r.(*route)
	assert.True(t, ok, "expected r to be of type *route")

	return routeInstance
}
