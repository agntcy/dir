// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"testing"

	"github.com/agntcy/dir/server/types"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const (
	policyTestCID   = "baeareitestpolicy0000000000000000000000000000000000000000000000"
	policyTestOther = "baeareitestpolicyother00000000000000000000000000000000000000000"
)

func setupPolicyEvalDB(t *testing.T) *DB {
	t.Helper()

	gdb, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	require.NoError(t, err)

	db, err := New(gdb)
	require.NoError(t, err)

	return db
}

func seedPolicyEvalRecord(t *testing.T, db *DB, cid string) {
	t.Helper()

	require.NoError(t, db.gormDB.Create(&Record{
		RecordCID:     cid,
		Name:          "policy-test-" + cid[:12],
		Version:       "1.0.0",
		SchemaVersion: "0.8.0",
	}).Error)
}

func loadPolicyEval(t *testing.T, db *DB, recordCID, policyID string) *PolicyEvaluation {
	t.Helper()

	var row PolicyEvaluation
	require.NoError(t, db.gormDB.
		Where("record_cid = ? AND policy_id = ?", recordCID, policyID).
		Take(&row).Error)

	return &row
}

func needsEvalCIDs(t *testing.T, db *DB, policyID, policyVersion string) []string {
	t.Helper()

	records, err := db.GetRecordsNeedingPolicyEvaluation(policyID, policyVersion)
	require.NoError(t, err)

	cids := make([]string, 0, len(records))
	for _, r := range records {
		cids = append(cids, r.GetCid())
	}

	return cids
}

// --- UpsertPolicyEvaluation ---

func TestUpsertPolicyEvaluation_StoresAnEvaluatedVerdict(t *testing.T) {
	t.Parallel()

	db := setupPolicyEvalDB(t)

	require.NoError(t, db.UpsertPolicyEvaluation(&PolicyEvaluation{
		RecordCID:     policyTestCID,
		PolicyID:      "opa:require-annotation",
		PolicyVersion: "v1",
		Compliant:     true,
		Status:        types.PolicyEvalStatusEvaluated,
	}))

	row := loadPolicyEval(t, db, policyTestCID, "opa:require-annotation")

	assert.True(t, row.Compliant)
	assert.Equal(t, types.PolicyEvalStatusEvaluated, row.Status)
	assert.Equal(t, "v1", row.PolicyVersion)
}

// The fail-closed rule the read-path filter (see #2208) depends on: a row
// that did not reach a verdict is never allowed to read as compliant, no
// matter what the caller passed.
func TestUpsertPolicyEvaluation_FailedStatusForcesCompliantFalse(t *testing.T) {
	t.Parallel()

	db := setupPolicyEvalDB(t)

	require.NoError(t, db.UpsertPolicyEvaluation(&PolicyEvaluation{
		RecordCID:     policyTestCID,
		PolicyID:      "opa:require-annotation",
		PolicyVersion: "v1",
		Compliant:     true, // deliberately wrong, to prove the row wins
		Status:        types.PolicyEvalStatusFailed,
		Reason:        "evaluator timed out",
	}))

	row := loadPolicyEval(t, db, policyTestCID, "opa:require-annotation")

	assert.False(t, row.Compliant)
	assert.Equal(t, types.PolicyEvalStatusFailed, row.Status)
	assert.Equal(t, "evaluator timed out", row.Reason)
}

func TestUpsertPolicyEvaluation_ReevaluationReplacesThePriorVerdict(t *testing.T) {
	t.Parallel()

	db := setupPolicyEvalDB(t)

	require.NoError(t, db.UpsertPolicyEvaluation(&PolicyEvaluation{
		RecordCID: policyTestCID, PolicyID: "opa:require-annotation",
		PolicyVersion: "v1", Compliant: false, Status: types.PolicyEvalStatusEvaluated,
		Reason: "missing annotation",
	}))
	require.NoError(t, db.UpsertPolicyEvaluation(&PolicyEvaluation{
		RecordCID: policyTestCID, PolicyID: "opa:require-annotation",
		PolicyVersion: "v2", Compliant: true, Status: types.PolicyEvalStatusEvaluated,
	}))

	row := loadPolicyEval(t, db, policyTestCID, "opa:require-annotation")

	assert.True(t, row.Compliant)
	assert.Equal(t, "v2", row.PolicyVersion)
	assert.Empty(t, row.Reason)
}

func TestUpsertPolicyEvaluation_DistinctPoliciesOnTheSameRecordCoexist(t *testing.T) {
	t.Parallel()

	db := setupPolicyEvalDB(t)

	require.NoError(t, db.UpsertPolicyEvaluation(&PolicyEvaluation{
		RecordCID: policyTestCID, PolicyID: "opa:require-annotation",
		PolicyVersion: "v1", Compliant: true, Status: types.PolicyEvalStatusEvaluated,
	}))
	require.NoError(t, db.UpsertPolicyEvaluation(&PolicyEvaluation{
		RecordCID: policyTestCID, PolicyID: "oasf:schema",
		PolicyVersion: "v1", Compliant: false, Status: types.PolicyEvalStatusEvaluated,
		Reason: "unknown skill id",
	}))

	rows, err := db.GetPolicyEvaluations(policyTestCID)
	require.NoError(t, err)
	require.Len(t, rows, 2)
}

// --- GetPolicyEvaluations ---

func TestGetPolicyEvaluations_UnknownRecordReturnsEmpty(t *testing.T) {
	t.Parallel()

	db := setupPolicyEvalDB(t)

	rows, err := db.GetPolicyEvaluations(policyTestCID)
	require.NoError(t, err)
	assert.Empty(t, rows)
}

func TestGetPolicyEvaluations_OnlyReturnsRowsForTheRequestedRecord(t *testing.T) {
	t.Parallel()

	db := setupPolicyEvalDB(t)

	require.NoError(t, db.UpsertPolicyEvaluation(&PolicyEvaluation{
		RecordCID: policyTestCID, PolicyID: "opa:require-annotation",
		PolicyVersion: "v1", Compliant: true, Status: types.PolicyEvalStatusEvaluated,
	}))
	require.NoError(t, db.UpsertPolicyEvaluation(&PolicyEvaluation{
		RecordCID: policyTestOther, PolicyID: "opa:require-annotation",
		PolicyVersion: "v1", Compliant: true, Status: types.PolicyEvalStatusEvaluated,
	}))

	rows, err := db.GetPolicyEvaluations(policyTestCID)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, policyTestCID, rows[0].GetRecordCID())
}

// --- GetRecordsNeedingPolicyEvaluation ---

func TestGetRecordsNeedingPolicyEvaluation_NeverEvaluatedIsSelected(t *testing.T) {
	t.Parallel()

	db := setupPolicyEvalDB(t)
	seedPolicyEvalRecord(t, db, policyTestCID)

	assert.Contains(t, needsEvalCIDs(t, db, "opa:require-annotation", "v1"), policyTestCID)
}

func TestGetRecordsNeedingPolicyEvaluation_CurrentVersionSuppresses(t *testing.T) {
	t.Parallel()

	db := setupPolicyEvalDB(t)
	seedPolicyEvalRecord(t, db, policyTestCID)

	require.NoError(t, db.UpsertPolicyEvaluation(&PolicyEvaluation{
		RecordCID: policyTestCID, PolicyID: "opa:require-annotation",
		PolicyVersion: "v1", Compliant: true, Status: types.PolicyEvalStatusEvaluated,
	}))

	assert.NotContains(t, needsEvalCIDs(t, db, "opa:require-annotation", "v1"), policyTestCID)
}

// A policy version bump has to re-select every record evaluated under the
// old version — the row exists, but it is stale, not authoritative for v2.
func TestGetRecordsNeedingPolicyEvaluation_SupersededVersionIsReselected(t *testing.T) {
	t.Parallel()

	db := setupPolicyEvalDB(t)
	seedPolicyEvalRecord(t, db, policyTestCID)

	require.NoError(t, db.UpsertPolicyEvaluation(&PolicyEvaluation{
		RecordCID: policyTestCID, PolicyID: "opa:require-annotation",
		PolicyVersion: "v1", Compliant: true, Status: types.PolicyEvalStatusEvaluated,
	}))

	assert.Contains(t, needsEvalCIDs(t, db, "opa:require-annotation", "v2"), policyTestCID)
}

func TestGetRecordsNeedingPolicyEvaluation_OnlyTheStaleRecordIsSelected(t *testing.T) {
	t.Parallel()

	db := setupPolicyEvalDB(t)
	seedPolicyEvalRecord(t, db, policyTestCID)
	seedPolicyEvalRecord(t, db, policyTestOther)

	require.NoError(t, db.UpsertPolicyEvaluation(&PolicyEvaluation{
		RecordCID: policyTestCID, PolicyID: "opa:require-annotation",
		PolicyVersion: "v1", Compliant: true, Status: types.PolicyEvalStatusEvaluated,
	}))

	cids := needsEvalCIDs(t, db, "opa:require-annotation", "v1")
	assert.NotContains(t, cids, policyTestCID)
	assert.Contains(t, cids, policyTestOther)
}

// A different policy_id is a distinct axis: an unrelated policy's up-to-date
// row must not suppress evaluation for this one.
func TestGetRecordsNeedingPolicyEvaluation_UnrelatedPolicyDoesNotSuppress(t *testing.T) {
	t.Parallel()

	db := setupPolicyEvalDB(t)
	seedPolicyEvalRecord(t, db, policyTestCID)

	require.NoError(t, db.UpsertPolicyEvaluation(&PolicyEvaluation{
		RecordCID: policyTestCID, PolicyID: "oasf:schema",
		PolicyVersion: "v1", Compliant: true, Status: types.PolicyEvalStatusEvaluated,
	}))

	assert.Contains(t, needsEvalCIDs(t, db, "opa:require-annotation", "v1"), policyTestCID)
}
