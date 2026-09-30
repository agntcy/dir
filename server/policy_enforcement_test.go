// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"path/filepath"
	"testing"

	"github.com/agntcy/dir/server/config"
	dbconfig "github.com/agntcy/dir/server/database/config"
	gormdb "github.com/agntcy/dir/server/database/gorm"
	"github.com/agntcy/dir/server/metrics"
	policyconfig "github.com/agntcy/dir/server/policy/config"
	"github.com/agntcy/dir/server/types"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

var enforcedA = []policyconfig.EnforcedPolicy{{ID: "opa:a", Version: "v1"}}

// With every mode off the APIs read the database as it is.
func TestServedDatabase_OffServesTheDatabaseItself(t *testing.T) {
	t.Parallel()

	db := newTestDatabase(t)

	served, err := servedDatabase(db, policyconfig.EnforcementConfig{Policies: enforcedA}, true, nil)
	require.NoError(t, err)
	assert.Same(t, db, served)
}

// A checked mode serves a separate view, so the database the reconciler
// reads stays unfiltered, and the gate reports to the metrics server.
func TestServedDatabase_CheckedModeServesAGatedView(t *testing.T) {
	t.Parallel()

	db := newTestDatabase(t)
	metricsServer := metrics.New("127.0.0.1:0")

	served, err := servedDatabase(db, policyconfig.EnforcementConfig{Fetch: policyconfig.ModeEnforce, Policies: enforcedA}, true, metricsServer)
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

	_, err := servedDatabase(notGorm, policyconfig.EnforcementConfig{Search: policyconfig.ModeShadow, Policies: enforcedA}, true, nil)
	require.ErrorContains(t, err, "cannot enforce content policies")

	_, err = servedDatabase(newTestDatabase(t), policyconfig.EnforcementConfig{Search: policyconfig.ModeEnforce}, true, nil)
	require.ErrorContains(t, err, "invalid policy enforcement config")
}

// Without a database handed in, the server opens the configured one, and its
// APIs read through the view the enforcement config calls for.
func TestOpenDatabase_OpensTheConfiguredDatabase(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}
	cfg.Database = dbconfig.Config{Type: "sqlite", SQLite: dbconfig.SQLiteConfig{Path: filepath.Join(t.TempDir(), "dir.db")}}
	cfg.Policy.Enforcement = policyconfig.EnforcementConfig{Fetch: policyconfig.ModeEnforce, Policies: enforcedA}

	db, served, err := openDatabase(cfg, nil, nil)
	require.NoError(t, err)
	require.IsType(t, &gormdb.DB{}, db)
	assert.NotSame(t, db, served)

	given := newTestDatabase(t)

	db, _, err = openDatabase(cfg, given, nil)
	require.NoError(t, err)
	assert.Same(t, given, db, "a database handed in is used as is")

	cfg.Database.Type = "unknown"

	_, _, err = openDatabase(cfg, nil, nil)
	require.ErrorContains(t, err, "failed to create database API")
}
