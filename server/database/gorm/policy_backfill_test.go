// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"testing"

	policyconfig "github.com/agntcy/dir/server/policy/config"
	"github.com/agntcy/dir/server/policy/enforcement"
	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPolicyBackfillBookkeeping(t *testing.T) {
	t.Parallel()

	db := setupGateDB(t, nil)
	seedGateRecord(t, db, gateOK, "skill/ok")
	seedGateRecord(t, db, gateBad, "skill/bad")

	setVerdict(t, db, gateOK, policyA, true)

	remaining, err := db.CountRecordsNeedingPolicyEvaluation(policyA.ID, policyA.Version)
	require.NoError(t, err)
	assert.Equal(t, int64(1), remaining)

	setVerdict(t, db, gateBad, policyA, false)

	remaining, err = db.CountRecordsNeedingPolicyEvaluation(policyA.ID, policyA.Version)
	require.NoError(t, err)
	assert.Zero(t, remaining, "a non-compliant verdict is still a verdict")

	remaining, err = db.CountRecordsNeedingPolicyEvaluation(policyA.ID, "another-version")
	require.NoError(t, err)
	assert.Equal(t, int64(2), remaining, "a verdict under another version does not count")

	backfilled, err := db.PolicyBackfilled(policyA.ID)
	require.NoError(t, err)
	assert.False(t, backfilled)

	require.NoError(t, db.MarkPolicyBackfilled(policyA.ID))
	require.NoError(t, db.MarkPolicyBackfilled(policyA.ID), "marking again")

	backfilled, err = db.PolicyBackfilled(policyA.ID)
	require.NoError(t, err)
	assert.True(t, backfilled)

	backfilled, err = db.PolicyBackfilled(policyB.ID)
	require.NoError(t, err)
	assert.False(t, backfilled, "another policy")
}

// A policy through its life, against the real filter, resolver and registry:
// not enforced while the records are first evaluated, enforced once they are,
// and, once edited, enforced at once, hiding every record until it has been
// evaluated under the new rule.
func TestPolicyLifecycle(t *testing.T) {
	t.Parallel()

	v1 := types.EnforcedPolicy{ID: policyA.ID, Version: "v1"}
	v2 := types.EnforcedPolicy{ID: policyA.ID, Version: "v2"}

	base := setupGateDB(t, nil)
	seedGateRecord(t, base, gateOK, "skill/ok")
	seedGateRecord(t, base, gateBad, "skill/bad")

	// The reconciler registers the policy and evaluates the first record.
	require.NoError(t, base.RegisterPolicyVersion(v1.ID, v1.Version))
	setVerdict(t, base, gateOK, v1, true)

	resolver, err := enforcement.NewResolver(policyconfig.EnforcementConfig{
		Search:   policyconfig.ModeEnforce,
		Fetch:    policyconfig.ModeEnforce,
		Policies: []string{v1.ID},
	}, base)
	require.NoError(t, err)

	served := base.Served(resolver.Current, nil)
	all := []string{gateOK, gateBad}

	assert.ElementsMatch(t, all, gateCIDs(t, served), "not enforced while a record has no verdict")

	// The second record is evaluated: every record has a verdict.
	setVerdict(t, base, gateBad, v1, false)
	require.NoError(t, resolver.Refresh())
	assert.Equal(t, []string{gateOK}, gateCIDs(t, served), "enforced once the records are covered")

	// The policy is edited. It is enforced at once, so nothing the new rule
	// rejects is served while the records are evaluated under it.
	require.NoError(t, base.RegisterPolicyVersion(v2.ID, v2.Version))
	require.NoError(t, resolver.Refresh())
	assert.Empty(t, gateCIDs(t, served), "hidden until evaluated under the new rule")
	assert.Empty(t, servable(t, served, all...))

	remaining, err := base.CountRecordsNeedingPolicyEvaluation(v2.ID, v2.Version)
	require.NoError(t, err)
	assert.Equal(t, int64(2), remaining)

	// The new rule is looser for one record and stricter for the other.
	setVerdict(t, base, gateOK, v2, false)
	setVerdict(t, base, gateBad, v2, true)

	assert.Equal(t, []string{gateBad}, gateCIDs(t, served))
	assert.Equal(t, []string{gateBad}, servable(t, served, all...))

	// A record published meanwhile is hidden until it has a verdict too.
	const published = "baeareipublished0000000000000000000000000000000000000000000000"

	seedGateRecord(t, base, published, "skill/published")
	assert.Equal(t, []string{gateBad}, gateCIDs(t, served))

	setVerdict(t, base, published, v2, true)
	assert.ElementsMatch(t, []string{gateBad, published}, gateCIDs(t, served))
}

// Each backfill query reports a database failure rather than an answer.
func TestPolicyBackfillBookkeeping_DatabaseFailure(t *testing.T) {
	t.Parallel()

	db := setupGateDB(t, nil)

	sqlDB, err := db.gormDB.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	_, err = db.CountRecordsNeedingPolicyEvaluation(policyA.ID, policyA.Version)
	require.ErrorContains(t, err, "count records needing policy evaluation")

	_, err = db.PolicyBackfilled(policyA.ID)
	require.ErrorContains(t, err, "check policy backfill")

	require.ErrorContains(t, db.MarkPolicyBackfilled(policyA.ID), "mark policy backfilled")
}
