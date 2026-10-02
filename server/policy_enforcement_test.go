// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"path/filepath"
	"testing"

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

var enforcedA = []string{"opa:a"}

// With every mode off the APIs read the database as it is.
func TestServedDatabase_OffServesTheDatabaseItself(t *testing.T) {
	t.Parallel()

	db := newTestDatabase(t)

	served, _, err := servedDatabase(db, policyconfig.EnforcementConfig{Policies: enforcedA}, true, nil)
	require.NoError(t, err)
	assert.Same(t, db, served)
}

// A checked mode serves a separate view, so the database the reconciler
// reads stays unfiltered, and the gate reports to the metrics server.
func TestServedDatabase_CheckedModeServesAGatedView(t *testing.T) {
	t.Parallel()

	db := newTestDatabase(t)
	require.NoError(t, db.RegisterPolicyVersion("opa:a", "v1"))

	metricsServer := metrics.New("127.0.0.1:0")

	served, _, err := servedDatabase(db, policyconfig.EnforcementConfig{Fetch: policyconfig.ModeEnforce, Policies: enforcedA}, true, metricsServer)
	require.NoError(t, err)
	assert.NotSame(t, db, served)

	ok, err := served.IsRecordServable("baeareigatenone000000000000000000000000000000000000000000000000")
	require.NoError(t, err)
	assert.False(t, ok, "the served view enforces the policies")

	ok, err = db.IsRecordServable("baeareigatenone000000000000000000000000000000000000000000000000")
	require.NoError(t, err)
	assert.True(t, ok, "the database itself does not")

	families, err := metricsServer.Registry().Gather()
	require.NoError(t, err)

	names := make([]string, 0, len(families))
	for _, family := range families {
		names = append(names, family.GetName())
	}

	assert.Contains(t, names, "dir_policy_gate_fetches_excluded_total")
}

// The server refuses to start rather than serve records it was told to check
// without checking them.
func TestServedDatabase_RefusesWhatItCannotEnforce(t *testing.T) {
	t.Parallel()

	var notGorm types.DatabaseAPI

	_, _, err := servedDatabase(notGorm, policyconfig.EnforcementConfig{Search: policyconfig.ModeShadow, Policies: enforcedA}, true, nil)
	require.ErrorContains(t, err, "cannot enforce content policies")

	_, _, err = servedDatabase(newTestDatabase(t), policyconfig.EnforcementConfig{Search: policyconfig.ModeEnforce}, true, nil)
	require.ErrorContains(t, err, "invalid policy enforcement config")
}

// Without a database handed in, the server opens the configured one, and its
// APIs read through the view the enforcement config calls for.
func TestOpenDatabase_OpensTheConfiguredDatabase(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}
	cfg.Database = dbconfig.Config{Type: "sqlite", SQLite: dbconfig.SQLiteConfig{Path: filepath.Join(t.TempDir(), "dir.db")}}
	cfg.Policy.Enforcement = policyconfig.EnforcementConfig{Fetch: policyconfig.ModeEnforce, Policies: enforcedA}

	db, served, _, err := openDatabase(cfg, nil, nil)
	require.NoError(t, err)
	require.IsType(t, &gormdb.DB{}, db)
	assert.NotSame(t, db, served)

	given := newTestDatabase(t)

	db, _, _, err = openDatabase(cfg, given, nil)
	require.NoError(t, err)
	assert.Same(t, given, db, "a database handed in is used as is")

	cfg.Database.Type = "unknown"

	db, served, current, err := openDatabase(cfg, nil, nil)
	require.ErrorContains(t, err, "failed to create database API")
	assert.Nil(t, db)
	assert.Nil(t, served)
	assert.Nil(t, current)
}

// A policy no evaluator has registered has no version whose verdicts could be
// asked for, so it is not enforced yet.
func TestServedDatabase_UnregisteredPolicyIsNotEnforced(t *testing.T) {
	t.Parallel()

	db := newTestDatabase(t)

	served, _, err := servedDatabase(db, policyconfig.EnforcementConfig{Fetch: policyconfig.ModeEnforce, Policies: enforcedA}, true, nil)
	require.NoError(t, err)

	ok, err := served.IsRecordServable("baeareigatenone000000000000000000000000000000000000000000000000")
	require.NoError(t, err)
	assert.True(t, ok)
}

// The server does not start when it cannot tell which policy versions are in
// force.
func TestServedDatabase_FailsWhenVersionsCannotBeResolved(t *testing.T) {
	t.Parallel()

	gdb, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	require.NoError(t, err)

	db, err := gormdb.New(gdb)
	require.NoError(t, err)

	sqlDB, err := gdb.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	_, _, err = servedDatabase(db, policyconfig.EnforcementConfig{Fetch: policyconfig.ModeEnforce, Policies: enforcedA}, true, nil)
	require.ErrorContains(t, err, "resolve enforced policy versions")
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

// What the served view enforces is handed on only when a read checks it.
func TestServedDatabase_ReturnsWhatItEnforces(t *testing.T) {
	t.Parallel()

	_, current, err := servedDatabase(newTestDatabase(t), policyconfig.EnforcementConfig{Policies: enforcedA}, true, nil)
	require.NoError(t, err)
	assert.Nil(t, current, "every mode off")

	_, current, err = servedDatabase(newTestDatabase(t), policyconfig.EnforcementConfig{Search: policyconfig.ModeShadow, Policies: enforcedA}, true, nil)
	require.NoError(t, err)
	require.NotNil(t, current)
	assert.Equal(t, policyconfig.ModeShadow, current().Search)
}

// Database setup fails on an invalid enforcement config, and when the gate's
// metrics cannot be registered.
func TestOpenDatabase_Failures(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}
	cfg.Policy.Enforcement = policyconfig.EnforcementConfig{Search: policyconfig.ModeEnforce}

	_, _, current, err := openDatabase(cfg, newTestDatabase(t), nil)
	require.ErrorContains(t, err, "invalid policy enforcement config")
	assert.Nil(t, current)

	metricsServer := metrics.New("127.0.0.1:0")
	enforcing := policyconfig.EnforcementConfig{Search: policyconfig.ModeShadow, Policies: enforcedA}

	_, _, err = servedDatabase(newTestDatabase(t), enforcing, true, metricsServer)
	require.NoError(t, err)

	_, _, err = servedDatabase(newTestDatabase(t), enforcing, true, metricsServer)
	require.ErrorContains(t, err, "register policy gate metrics")
}
