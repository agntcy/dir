// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"testing"
	"time"

	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func currentPolicyVersion(t *testing.T, db *DB, policyID string) (string, bool) {
	t.Helper()

	version, found, err := db.GetCurrentPolicyVersion(policyID)
	require.NoError(t, err)

	return version, found
}

// policyGroups reads back what a version was recorded as reading, and whether
// anything was recorded for it at all. It reads the table the same way the
// invalidation predicate does, rather than through an accessor on DB: nothing
// in the server asks this question in Go, so an exported method answering it
// would exist only for these tests.
//
// A version that reads nothing is recorded as one types.FactGroupNone row.
// Dropping it here is what makes the two cases the tests care about distinct:
// nil groups with registered true means it reads nothing, nil with false
// means nothing is known.
func policyGroups(t *testing.T, db *DB, policyID, policyVersion string) ([]string, bool) {
	t.Helper()

	var stored []string

	require.NoError(t, db.gormDB.Model(&PolicyFactDependency{}).
		Where("policy_id = ? AND policy_version = ?", policyID, policyVersion).
		Order("fact_group").
		Pluck("fact_group", &stored).Error)

	if len(stored) == 0 {
		return nil, false
	}

	var groups []string

	for _, group := range stored {
		if group != types.FactGroupNone {
			groups = append(groups, group)
		}
	}

	return groups, true
}

// What a policy reads has to survive being stored, because the write that
// invalidates a verdict is in another process from the evaluator that read it.
func TestRegisterPolicyVersion_StoresWhatTheVersionReads(t *testing.T) {
	t.Parallel()

	db := setupPolicyEvalDB(t)

	require.NoError(t, db.RegisterPolicyVersion("cel:trusted", "v1",
		[]string{types.FactGroupScan, types.FactGroupSignature}))

	groups, registered := policyGroups(t, db, "cel:trusted", "v1")
	assert.True(t, registered)
	assert.Equal(t, []string{types.FactGroupScan, types.FactGroupSignature}, groups)
}

// "Reads no metadata" and "nothing is known about this policy" have to be
// distinguishable: the first means a metadata write should leave its verdicts
// alone, the second means it must not.
func TestRegisterPolicyVersion_ReadingNothingIsNotTheSameAsUnregistered(t *testing.T) {
	t.Parallel()

	db := setupPolicyEvalDB(t)

	require.NoError(t, db.RegisterPolicyVersion("opa:require-license", "v1", nil))

	groups, registered := policyGroups(t, db, "opa:require-license", "v1")
	assert.True(t, registered, "the policy is known")
	assert.Empty(t, groups, "and reads no metadata")

	groups, registered = policyGroups(t, db, "opa:unseen", "v1")
	assert.False(t, registered, "nothing is known about this one")
	assert.Empty(t, groups)
}

// An edit is a new version, so the dependencies of the old one stay as they
// were: its verdicts no longer count, but nothing rewrites history.
func TestRegisterPolicyVersion_EachVersionKeepsItsOwnDependencies(t *testing.T) {
	t.Parallel()

	db := setupPolicyEvalDB(t)

	require.NoError(t, db.RegisterPolicyVersion("cel:trusted", "v1", []string{types.FactGroupScan}))
	require.NoError(t, db.RegisterPolicyVersion("cel:trusted", "v2", []string{types.FactGroupIdentity}))

	v1Groups, _ := policyGroups(t, db, "cel:trusted", "v1")
	assert.Equal(t, []string{types.FactGroupScan}, v1Groups)

	v2Groups, _ := policyGroups(t, db, "cel:trusted", "v2")
	assert.Equal(t, []string{types.FactGroupIdentity}, v2Groups)
}

// The reconciler registers on every run, so the common case is registering
// what is already there.
func TestRegisterPolicyVersion_RegisteringAgainDoesNotDuplicate(t *testing.T) {
	t.Parallel()

	db := setupPolicyEvalDB(t)

	groups := []string{types.FactGroupScan, types.FactGroupSignature}
	require.NoError(t, db.RegisterPolicyVersion("cel:trusted", "v1", groups))
	require.NoError(t, db.RegisterPolicyVersion("cel:trusted", "v1", groups))

	stored, _ := policyGroups(t, db, "cel:trusted", "v1")
	assert.Equal(t, groups, stored)
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

	require.NoError(t, db.RegisterPolicyVersion("opa:require-annotation", "v1", nil))

	version, found := currentPolicyVersion(t, db, "opa:require-annotation")
	require.True(t, found)
	assert.Equal(t, "v1", version)

	require.NoError(t, db.RegisterPolicyVersion("opa:require-annotation", "v2", nil))

	version, _ = currentPolicyVersion(t, db, "opa:require-annotation")
	assert.Equal(t, "v2", version)
}

// Returning a policy to an earlier version makes that version current again,
// and registering the version already current only refreshes when it was seen.
func TestRegisterPolicyVersion_ReturningToAnEarlierVersion(t *testing.T) {
	t.Parallel()

	db := setupPolicyEvalDB(t)

	require.NoError(t, db.RegisterPolicyVersion("opa:require-annotation", "v1", nil))
	time.Sleep(2 * time.Millisecond)
	require.NoError(t, db.RegisterPolicyVersion("opa:require-annotation", "v2", nil))
	time.Sleep(2 * time.Millisecond)
	require.NoError(t, db.RegisterPolicyVersion("opa:require-annotation", "v1", nil))
	require.NoError(t, db.RegisterPolicyVersion("opa:require-annotation", "v1", nil))

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

	require.NoError(t, db.RegisterPolicyVersion("opa:a", "a1", nil))
	require.NoError(t, db.RegisterPolicyVersion("opa:b", "b7", nil))

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

	require.ErrorContains(t, db.RegisterPolicyVersion("opa:a", "a1", nil), "register policy version")

	_, _, err = db.GetCurrentPolicyVersion("opa:a")
	require.ErrorContains(t, err, "get current policy version")
}
