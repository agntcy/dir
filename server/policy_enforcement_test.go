// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"testing"

	gormdb "github.com/agntcy/dir/server/database/gorm"
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

	served, err := servedDatabase(db, policyconfig.EnforcementConfig{Policies: enforcedA}, true)
	require.NoError(t, err)
	assert.Same(t, db, served)
}

// A checked mode serves a separate view, so the database the reconciler
// reads stays unfiltered.
func TestServedDatabase_CheckedModeServesAGatedView(t *testing.T) {
	t.Parallel()

	db := newTestDatabase(t)

	served, err := servedDatabase(db, policyconfig.EnforcementConfig{Fetch: policyconfig.ModeEnforce, Policies: enforcedA}, true)
	require.NoError(t, err)
	assert.NotSame(t, db, served)

	ok, err := served.IsRecordServable("baeareigatenone000000000000000000000000000000000000000000000000")
	require.NoError(t, err)
	assert.False(t, ok, "the served view enforces the policies")

	ok, err = db.IsRecordServable("baeareigatenone000000000000000000000000000000000000000000000000")
	require.NoError(t, err)
	assert.True(t, ok, "the database itself does not")
}

// The server refuses to start rather than serve records it was told to check
// without checking them.
func TestServedDatabase_RefusesWhatItCannotEnforce(t *testing.T) {
	t.Parallel()

	var notGorm types.DatabaseAPI

	_, err := servedDatabase(notGorm, policyconfig.EnforcementConfig{Search: policyconfig.ModeShadow, Policies: enforcedA}, true)
	require.ErrorContains(t, err, "cannot enforce content policies")

	_, err = servedDatabase(newTestDatabase(t), policyconfig.EnforcementConfig{Search: policyconfig.ModeEnforce}, true)
	require.ErrorContains(t, err, "invalid policy enforcement config")
}
