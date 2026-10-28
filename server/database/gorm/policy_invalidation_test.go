// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"testing"

	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// invalidationPolicyID is the verdict every test here watches. Which policy
// reached it does not matter yet: a metadata write supersedes all of a
// record's verdicts, and narrowing that to the policies that read the fact
// that moved is a later increment.
const invalidationPolicyID = "cel:trusted"

// setupInvalidationDB returns a database holding one indexed record with one
// current verdict, which is the state a metadata write has to invalidate.
func setupInvalidationDB(t *testing.T) *DB {
	t.Helper()

	db := setupPolicyEvalDB(t)
	seedPolicyEvalRecord(t, db, policyTestCID)
	seedVerdict(t, db)

	return db
}

func seedVerdict(t *testing.T, db *DB) {
	t.Helper()

	require.NoError(t, db.UpsertPolicyEvaluation(&PolicyEvaluation{
		RecordCID:     policyTestCID,
		PolicyID:      invalidationPolicyID,
		PolicyVersion: "v1",
		Status:        types.PolicyEvalStatusEvaluated,
		Compliant:     true,
	}))
}

// assertSuperseded and assertStillCurrent read the same row. They are separate
// so that a failure names which of the two the test expected.
func assertSuperseded(t *testing.T, db *DB) {
	t.Helper()

	row := loadPolicyEval(t, db, policyTestCID, invalidationPolicyID)
	assert.Equal(t, types.PolicyEvalStatusSuperseded, row.Status, "the write should have stopped the verdict")
}

func assertStillCurrent(t *testing.T, db *DB) {
	t.Helper()

	row := loadPolicyEval(t, db, policyTestCID, invalidationPolicyID)
	assert.Equal(t, types.PolicyEvalStatusEvaluated, row.Status, "the write changed nothing a policy can read")
}

func scanReport(status, severity string, safe bool) *ScanReport {
	return &ScanReport{
		RecordCID:   policyTestCID,
		ScannerType: "MCP",
		Status:      status,
		MaxSeverity: severity,
		IsSafe:      safe,
		Source:      types.ScanSourceLocal,
	}
}

func upsertScan(t *testing.T, db *DB, report *ScanReport) {
	t.Helper()

	require.NoError(t, db.UpsertScanReport(report, types.DefaultScanSchedule()))
}

// --- scan reports ---

func TestUpsertScanReport_AFirstReportSupersedes(t *testing.T) {
	t.Parallel()

	db := setupInvalidationDB(t)

	upsertScan(t, db, scanReport(types.ScanStatusCompleted, "NONE", true))

	assertSuperseded(t, db)
}

func TestUpsertScanReport_AChangedResultSupersedes(t *testing.T) {
	t.Parallel()

	db := setupInvalidationDB(t)

	upsertScan(t, db, scanReport(types.ScanStatusCompleted, "NONE", true))
	seedVerdict(t, db)

	upsertScan(t, db, scanReport(types.ScanStatusCompleted, "HIGH", false))

	assertSuperseded(t, db)
}

// A rescan that confirms the previous result is the common case: the TTL
// expires, the scanner runs, and nothing it reports has changed. The row is
// still rewritten and updated_at still moves, which is why the comparison
// cannot include it.
func TestUpsertScanReport_ReportingTheSameResultAgainChangesNothing(t *testing.T) {
	t.Parallel()

	db := setupInvalidationDB(t)

	upsertScan(t, db, scanReport(types.ScanStatusCompleted, "HIGH", false))
	seedVerdict(t, db)

	upsertScan(t, db, scanReport(types.ScanStatusCompleted, "HIGH", false))

	assertStillCurrent(t, db)
}

// A repeated failure advances the failure count and the retry schedule. Those
// are the reconciler's own bookkeeping, which no policy reads.
func TestUpsertScanReport_RetryBookkeepingAloneChangesNothing(t *testing.T) {
	t.Parallel()

	db := setupInvalidationDB(t)

	failure := scanReport(types.ScanStatusFailed, "", false)
	failure.FailureReason = "network"

	upsertScan(t, db, failure)
	seedVerdict(t, db)

	upsertScan(t, db, failure)

	assertStillCurrent(t, db)

	row := loadScanReport(t, db)
	assert.Equal(t, 2, row.ConsecutiveFailures, "the retry counter should still have advanced")
}

func loadScanReport(t *testing.T, db *DB) *ScanReport {
	t.Helper()

	var row ScanReport
	require.NoError(t, db.gormDB.
		Where("record_cid = ? AND scanner_type = ?", policyTestCID, "MCP").
		Take(&row).Error)

	return &row
}

// --- signature verifications ---

func signatureVerification(status, subject string) *SignatureVerification {
	return &SignatureVerification{
		RecordCID:     policyTestCID,
		SignerKey:     "signer-1",
		Status:        status,
		SignerSubject: subject,
		SignerIssuer:  "https://issuer.example",
	}
}

func TestUpsertSignatureVerification_AFlippedStatusSupersedes(t *testing.T) {
	t.Parallel()

	db := setupInvalidationDB(t)

	require.NoError(t, db.UpsertSignatureVerification(signatureVerification(types.SignatureStatusVerified, "dev@example.com")))
	seedVerdict(t, db)

	require.NoError(t, db.UpsertSignatureVerification(signatureVerification(types.SignatureStatusFailed, "dev@example.com")))

	assertSuperseded(t, db)
}

// The signer is projected as meta.signature.signer_identity, so a key that
// now verifies against a different subject changes what a policy matching on
// the signer reads, even though the signature still verifies.
func TestUpsertSignatureVerification_ADifferentSignerSupersedes(t *testing.T) {
	t.Parallel()

	db := setupInvalidationDB(t)

	require.NoError(t, db.UpsertSignatureVerification(signatureVerification(types.SignatureStatusVerified, "dev@example.com")))
	seedVerdict(t, db)

	require.NoError(t, db.UpsertSignatureVerification(signatureVerification(types.SignatureStatusVerified, "someone-else@example.com")))

	assertSuperseded(t, db)
}

func TestUpsertSignatureVerification_ReverifyingTheSameSignatureChangesNothing(t *testing.T) {
	t.Parallel()

	db := setupInvalidationDB(t)

	require.NoError(t, db.UpsertSignatureVerification(signatureVerification(types.SignatureStatusVerified, "dev@example.com")))
	seedVerdict(t, db)

	require.NoError(t, db.UpsertSignatureVerification(signatureVerification(types.SignatureStatusVerified, "dev@example.com")))

	assertStillCurrent(t, db)
}

// The signature task marks a signer failed once it stops appearing in a
// verification run, which is how a revoked or rotated key reaches a policy.
func TestUpdateSignatureVerification_MarkingASignerFailedSupersedes(t *testing.T) {
	t.Parallel()

	db := setupInvalidationDB(t)

	require.NoError(t, db.UpsertSignatureVerification(signatureVerification(types.SignatureStatusVerified, "dev@example.com")))
	seedVerdict(t, db)

	require.NoError(t, db.UpdateSignatureVerification(&SignatureVerification{
		RecordCID: policyTestCID,
		SignerKey: "signer-1",
		Status:    types.SignatureStatusFailed,
	}))

	assertSuperseded(t, db)
}

// Re-writing the status a row already holds refreshes updated_at and nothing
// else, so it is the same non-event as a confirming rescan.
func TestUpdateSignatureVerification_RewritingTheSameStatusChangesNothing(t *testing.T) {
	t.Parallel()

	db := setupInvalidationDB(t)

	require.NoError(t, db.UpsertSignatureVerification(signatureVerification(types.SignatureStatusFailed, "dev@example.com")))
	seedVerdict(t, db)

	require.NoError(t, db.UpdateSignatureVerification(&SignatureVerification{
		RecordCID: policyTestCID,
		SignerKey: "signer-1",
		Status:    types.SignatureStatusFailed,
	}))

	assertStillCurrent(t, db)
}

// --- identity claims ---

func identityClaim(status, subject string) *IdentityClaim {
	return &IdentityClaim{
		RecordCID: policyTestCID,
		Role:      types.ClaimRoleOwner,
		Subject:   subject,
		Status:    status,
	}
}

func TestUpsertIdentityClaim_AClaimResolvingDifferentlySupersedes(t *testing.T) {
	t.Parallel()

	db := setupInvalidationDB(t)

	require.NoError(t, db.UpsertIdentityClaim(identityClaim(types.ClaimStatusFailed, "example.com")))
	seedVerdict(t, db)

	require.NoError(t, db.UpsertIdentityClaim(identityClaim(types.ClaimStatusVerified, "example.com")))

	assertSuperseded(t, db)
}

// The projection prefers the stored subject over the one the record declares,
// so a claim that resolves to a different owner changes what a policy reads
// without changing whether the claim verified.
func TestUpsertIdentityClaim_AClaimResolvingToAnotherSubjectSupersedes(t *testing.T) {
	t.Parallel()

	db := setupInvalidationDB(t)

	require.NoError(t, db.UpsertIdentityClaim(identityClaim(types.ClaimStatusVerified, "example.com")))
	seedVerdict(t, db)

	require.NoError(t, db.UpsertIdentityClaim(identityClaim(types.ClaimStatusVerified, "other.example.com")))

	assertSuperseded(t, db)
}

// The identity task re-asserts every claim on every run, so this is the write
// that would otherwise re-evaluate the whole catalogue on every pass.
func TestUpsertIdentityClaim_ReassertingTheSameClaimChangesNothing(t *testing.T) {
	t.Parallel()

	db := setupInvalidationDB(t)

	require.NoError(t, db.UpsertIdentityClaim(identityClaim(types.ClaimStatusVerified, "example.com")))
	seedVerdict(t, db)

	require.NoError(t, db.UpsertIdentityClaim(identityClaim(types.ClaimStatusVerified, "example.com")))

	assertStillCurrent(t, db)
}

// Removing a claim is the one change a policy sees that no row's contents
// record, so it has to be caught at the delete rather than by a comparison.
func TestDeleteIdentityClaim_RemovingAResolvedClaimSupersedes(t *testing.T) {
	t.Parallel()

	db := setupInvalidationDB(t)

	require.NoError(t, db.UpsertIdentityClaim(identityClaim(types.ClaimStatusVerified, "example.com")))
	seedVerdict(t, db)

	require.NoError(t, db.DeleteIdentityClaim(policyTestCID, types.ClaimRoleOwner))

	assertSuperseded(t, db)
}

func TestDeleteIdentityClaim_RemovingNothingChangesNothing(t *testing.T) {
	t.Parallel()

	db := setupInvalidationDB(t)

	require.NoError(t, db.DeleteIdentityClaim(policyTestCID, types.ClaimRoleOwner))

	assertStillCurrent(t, db)
}

// --- the signed flag ---

func TestSetRecordSigned_RaisingTheFlagSupersedes(t *testing.T) {
	t.Parallel()

	db := setupInvalidationDB(t)

	require.NoError(t, db.SetRecordSigned(policyTestCID))

	assertSuperseded(t, db)
}

// A signature referrer can arrive more than once, and the second one tells a
// policy nothing the first did not.
func TestSetRecordSigned_RaisingAnAlreadyRaisedFlagChangesNothing(t *testing.T) {
	t.Parallel()

	db := setupInvalidationDB(t)

	require.NoError(t, db.SetRecordSigned(policyTestCID))
	seedVerdict(t, db)

	require.NoError(t, db.SetRecordSigned(policyTestCID))

	assertStillCurrent(t, db)
}

// Restricting the update to unsigned rows means a missing record and an
// already-signed one both affect nothing, so the error has to survive the
// change that tells them apart.
func TestSetRecordSigned_MissingRecordIsStillAnError(t *testing.T) {
	t.Parallel()

	db := setupPolicyEvalDB(t)

	err := db.SetRecordSigned(policyTestCID)

	require.ErrorContains(t, err, "record not found")
}
