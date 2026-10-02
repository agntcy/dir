// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func currentPolicyVersion(t *testing.T, db *DB, policyID string) (string, bool) {
	t.Helper()

	version, found, err := db.GetCurrentPolicyVersion(policyID)
	require.NoError(t, err)

	return version, found
}

func TestGetCurrentPolicyVersion_UnregisteredPolicy(t *testing.T) {
	t.Parallel()

	db := setupPolicyEvalDB(t)

	version, found := currentPolicyVersion(t, db, "opa:require-annotation")
	assert.False(t, found)
	assert.Empty(t, version)
}

// Editing a policy registers a new version, which becomes the current one.
func TestRegisterPolicyVersion_ANewVersionBecomesCurrent(t *testing.T) {
	t.Parallel()

	db := setupPolicyEvalDB(t)

	require.NoError(t, db.RegisterPolicyVersion("opa:require-annotation", "v1"))

	version, found := currentPolicyVersion(t, db, "opa:require-annotation")
	require.True(t, found)
	assert.Equal(t, "v1", version)

	require.NoError(t, db.RegisterPolicyVersion("opa:require-annotation", "v2"))

	version, _ = currentPolicyVersion(t, db, "opa:require-annotation")
	assert.Equal(t, "v2", version)
}

// Returning a policy to an earlier version makes that version current again,
// and registering the version already current only refreshes when it was seen.
func TestRegisterPolicyVersion_ReturningToAnEarlierVersion(t *testing.T) {
	t.Parallel()

	db := setupPolicyEvalDB(t)

	require.NoError(t, db.RegisterPolicyVersion("opa:require-annotation", "v1"))
	time.Sleep(2 * time.Millisecond)
	require.NoError(t, db.RegisterPolicyVersion("opa:require-annotation", "v2"))
	time.Sleep(2 * time.Millisecond)
	require.NoError(t, db.RegisterPolicyVersion("opa:require-annotation", "v1"))
	require.NoError(t, db.RegisterPolicyVersion("opa:require-annotation", "v1"))

	version, _ := currentPolicyVersion(t, db, "opa:require-annotation")
	assert.Equal(t, "v1", version)

	var rows []PolicyVersion
	require.NoError(t, db.gormDB.Order("id").Find(&rows).Error)
	require.Len(t, rows, 2, "a version is registered once")
	assert.True(t, rows[0].LastSeenAt.After(rows[0].FirstSeenAt), "v1 was seen again")
}

func TestRegisterPolicyVersion_PoliciesAreIndependent(t *testing.T) {
	t.Parallel()

	db := setupPolicyEvalDB(t)

	require.NoError(t, db.RegisterPolicyVersion("opa:a", "a1"))
	require.NoError(t, db.RegisterPolicyVersion("opa:b", "b7"))

	version, _ := currentPolicyVersion(t, db, "opa:a")
	assert.Equal(t, "a1", version)

	version, _ = currentPolicyVersion(t, db, "opa:b")
	assert.Equal(t, "b7", version)
}

func TestPolicyVersionRegistry_DatabaseFailure(t *testing.T) {
	t.Parallel()

	db := setupPolicyEvalDB(t)

	sqlDB, err := db.gormDB.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	require.ErrorContains(t, db.RegisterPolicyVersion("opa:a", "a1"), "register policy version")

	_, _, err = db.GetCurrentPolicyVersion("opa:a")
	require.ErrorContains(t, err, "get current policy version")
}
