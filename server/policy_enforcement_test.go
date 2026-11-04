// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"path/filepath"
	"testing"

	typesv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	policyv1 "github.com/agntcy/dir/api/policy/v1"
	"github.com/agntcy/dir/server/config"
	dbconfig "github.com/agntcy/dir/server/database/config"
	gormdb "github.com/agntcy/dir/server/database/gorm"
	"github.com/agntcy/dir/server/metrics"
	policyconfig "github.com/agntcy/dir/server/policy/config"
	"github.com/agntcy/dir/server/types"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"gorm.io/gorm"
)

func newTestDatabase(t *testing.T) *gormdb.DB {
	t.Helper()

	gdb, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	require.NoError(t, err)

	db, err := gormdb.New(gdb)
	require.NoError(t, err)

	return db
}

// searchConfig is a config whose only policy is one at op search, reading
// the verdict of an evaluation policy.
func searchConfig() *config.Config {
	cfg := &config.Config{}
	cfg.Authz.Enabled = true
	cfg.Policy.Evaluations = policyconfig.Evaluations{{
		Provider:    policyconfig.ProviderCEL,
		Name:        "trusted",
		Expressions: []string{`record.name != ""`},
	}}
	cfg.Policy.Operations = policyconfig.Operations{{
		Provider:    policyconfig.ProviderCEL,
		Ops:         []string{policyconfig.OpSearch},
		Name:        "serve-checked-records",
		Expressions: []string{`verdicts["cel:trusted"].compliant_and_current`},
	}}

	return cfg
}

func wire(t *testing.T, cfg *config.Config, db types.DatabaseAPI, metricsServer *metrics.Server,
) (types.DatabaseAPI, func() types.PolicyEnforcement) {
	t.Helper()

	wired, err := wirePolicies(context.Background(), cfg, db, metricsServer)
	require.NoError(t, err)

	return wired.Served, wired.Enforcement
}

// wireErr is what wirePolicies reports, for the cases about refusing to start.
func wireErr(cfg *config.Config, db types.DatabaseAPI, metricsServer *metrics.Server) error {
	_, err := wirePolicies(context.Background(), cfg, db, metricsServer)

	return err
}

// With nothing to enforce the APIs read a view that applies nothing, and
// there is no enforcement to hand on, which is what says there is nothing to
// audit. An evaluation policy alone is not something to enforce: it produces
// verdicts, and acts on none of them.
func TestWirePolicies_NothingEnforced(t *testing.T) {
	t.Parallel()

	cfg := searchConfig()
	cfg.Policy.Operations = nil

	_, current := wire(t, cfg, newTestDatabase(t), nil)
	assert.Nil(t, current, "no search policy and nothing at pull")
}

// A policy being checked serves a separate view, so the database the
// reconciler reads stays unfiltered, and the gate reports to the metrics
// server.
func TestWirePolicies_ACheckedPolicyServesAGatedView(t *testing.T) {
	t.Parallel()

	db := newTestDatabase(t)
	require.NoError(t, db.RegisterPolicyVersion("cel:trusted", "v1", nil))

	record := corev1.New(&typesv1alpha1.Record{Name: "refused", Version: "1.0.0", SchemaVersion: "0.7.0"})

	adapter, err := record.Decode()
	require.NoError(t, err)
	require.NoError(t, db.AddRecord(adapter))

	// A verdict, so the catalogue is covered and the latch releases; it says
	// non-compliant, so the policy reading it hides the record.
	require.NoError(t, db.UpsertPolicyEvaluation(&gormdb.PolicyEvaluation{
		RecordCID: record.GetCid(), PolicyID: "cel:trusted", PolicyVersion: "v1",
		Status: types.PolicyEvalStatusEvaluated, Compliant: false,
	}))

	metricsServer := metrics.New("127.0.0.1:0")

	served, _ := wire(t, searchConfig(), db, metricsServer)

	gated, err := served.GetRecords()
	require.NoError(t, err)
	assert.Empty(t, gated, "the served view applies the search policy")

	all, err := db.GetRecords()
	require.NoError(t, err)
	assert.Len(t, all, 1, "the database itself does not")

	families, err := metricsServer.Registry().Gather()
	require.NoError(t, err)

	names := make([]string, 0, len(families))
	for _, family := range families {
		names = append(names, family.GetName())
	}

	assert.Contains(t, names, "dir_policy_gate_pulls_excluded_total")
}

// A search policy is checked as soon as it is configured, and nothing else
// has to say so. Until the evaluation policy it reads has a version in
// force, it is not applied: the latch.
func TestWirePolicies_ASearchPolicyIsCheckedAsConfigured(t *testing.T) {
	t.Parallel()

	db := newTestDatabase(t)

	_, current := wire(t, searchConfig(), db, nil)
	require.NotNil(t, current, "a search policy is something to enforce")
	assert.Empty(t, current().SearchPolicies, "no evaluator has registered the policy it reads")

	// Once the evaluation policy is registered and its verdicts cover the
	// catalogue — vacuously, with no records — the search policy applies.
	require.NoError(t, db.RegisterPolicyVersion("cel:trusted", "v1", nil))

	_, current = wire(t, searchConfig(), db, nil)

	applied := current().SearchPolicies
	require.Len(t, applied, 1)
	assert.Equal(t, "cel:serve-checked-records", applied[0].ID)
	assert.Equal(t, []types.VersionedPolicy{{ID: "cel:trusted", Version: "v1"}}, applied[0].Reads)
}

// A search policy in shadow is checked but not applied: it reaches the
// enforcement struct, so it can be counted and audited, and is left out of
// the set queries carry.
func TestWirePolicies_AShadowSearchPolicyIsNotApplied(t *testing.T) {
	t.Parallel()

	db := newTestDatabase(t)
	require.NoError(t, db.RegisterPolicyVersion("cel:trusted", "v1", nil))

	cfg := searchConfig()
	cfg.Policy.Operations[0].Mode = policyconfig.ModeShadow

	_, current := wire(t, cfg, db, nil)
	require.NotNil(t, current)

	resolved := current()
	require.Len(t, resolved.SearchPolicies, 1, "a shadow policy is still checked")
	assert.Equal(t, policyconfig.ModeShadow, resolved.SearchPolicies[0].Mode)
	assert.Empty(t, resolved.EnforcingSearchPolicies(), "and not applied")
}

// The server refuses to start rather than serve records it was told to check
// without checking them.
//
// A database no query can be built on carries no gate, and every policy that
// decides what is served reads a verdict, so the registry refuses it for
// want of a verdict source. That is one refusal rather than two, and it
// names the policy.
func TestWirePolicies_RefusesWhatItCannotEnforce(t *testing.T) {
	t.Parallel()

	var notGorm types.DatabaseAPI

	require.ErrorContains(t, wireErr(searchConfig(), notGorm, nil),
		`at op "search" reads verdicts, which this process has no source of`)
}

// The server does not start when it cannot tell which policy versions are in
// force, or when the gate's metrics cannot be registered.
func TestWirePolicies_Failures(t *testing.T) {
	t.Parallel()

	gdb, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	require.NoError(t, err)

	db, err := gormdb.New(gdb)
	require.NoError(t, err)

	sqlDB, err := gdb.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	require.ErrorContains(t, wireErr(searchConfig(), db, nil), "resolve enforced policy versions")

	metricsServer := metrics.New("127.0.0.1:0")

	require.NoError(t, wireErr(searchConfig(), newTestDatabase(t), metricsServer))
	require.ErrorContains(t, wireErr(searchConfig(), newTestDatabase(t), metricsServer),
		"register policy gate metrics")
}

// Without a database handed in, the server opens the configured one.
func TestOpenDatabase_OpensTheConfiguredDatabase(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}
	cfg.Database = dbconfig.Config{Type: "sqlite", SQLite: dbconfig.SQLiteConfig{Path: filepath.Join(t.TempDir(), "dir.db")}}

	db, err := openDatabase(cfg, nil)
	require.NoError(t, err)
	require.IsType(t, &gormdb.DB{}, db)

	given := newTestDatabase(t)

	db, err = openDatabase(cfg, given)
	require.NoError(t, err)
	assert.Same(t, given, db, "a database handed in is used as is")

	cfg.Database.Type = "unknown"

	db, err = openDatabase(cfg, nil)
	require.ErrorContains(t, err, "failed to create database API")
	assert.Nil(t, db)
}

// registrations records the services registered on it.
type registrations struct {
	services []string
}

func (r *registrations) RegisterService(desc *grpc.ServiceDesc, _ any) {
	r.services = append(r.services, desc.ServiceName)
}

// The audit service returns excluded records and is protected only by its
// own permission, so it is offered only while policies are checked and
// authorization is on.
func TestRegisterPolicyAudit(t *testing.T) {
	t.Parallel()

	current := func() types.PolicyEnforcement { return types.PolicyEnforcement{} }

	var notGorm types.DatabaseAPI

	tests := []struct {
		name         string
		authzEnabled bool
		db           types.DatabaseAPI
		current      func() types.PolicyEnforcement
		want         []string
	}{
		{"policies checked, authorization on", true, newTestDatabase(t), current, []string{policyv1.PolicyAuditService_ServiceDesc.ServiceName}},
		{"authorization off", false, newTestDatabase(t), current, nil},
		{"no policy checked", true, newTestDatabase(t), nil, nil},
		{"database cannot list excluded records", true, notGorm, current, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			registrar := &registrations{}
			registerPolicyAudit(registrar, tt.authzEnabled, tt.db, nil, tt.current)

			assert.Equal(t, tt.want, registrar.services)
		})
	}
}
