// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"testing"
	"time"

	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/require"
)

func TestUpsertIdentityClaim_RenewingExpiredEvidenceSupersedes(t *testing.T) {
	t.Parallel()

	db := setupInvalidationDB(t)
	claim := identityClaim(types.ClaimStatusVerified, "agntcy://Agent")
	future := time.Now().UTC().Add(time.Hour)
	claim.ValidUntil = &future
	require.NoError(t, db.UpsertIdentityClaim(claim))

	// Model time passing without rewriting the stored verified status.
	require.NoError(t, db.gormDB.Model(&IdentityClaim{}).
		Where("record_cid = ? AND role = ?", claim.RecordCID, claim.Role).
		Update("valid_until", time.Now().UTC().Add(-time.Hour)).Error)
	seedVerdict(t, db)

	require.NoError(t, db.UpsertIdentityClaim(claim))
	assertSuperseded(t, db)
}

func TestPolicyGate_ExpiredClaimWrittenAsFailedSupersedes(t *testing.T) {
	t.Parallel()

	db := setupGateDB(t, enforce(policyA))
	seedGateRecord(t, db, gateOK, "skill/identity-expiry")
	require.NoError(t, db.RegisterPolicyVersion(policyA.ID, policyA.Version, []string{types.FactGroupIdentity}))

	claim := identityClaim(types.ClaimStatusVerified, "agntcy://Agent")
	claim.RecordCID = gateOK
	until := time.Now().UTC().Add(time.Hour)
	claim.ValidUntil = &until
	require.NoError(t, db.UpsertIdentityClaim(claim))
	setVerdict(t, db, gateOK, policyA, true)
	require.Equal(t, []string{gateOK}, gateCIDs(t, db))

	// Advance the evidence deadline without changing its stored status or
	// the verdict reached while it was valid. No wall-clock sleep is needed.
	require.NoError(t, db.gormDB.Model(&IdentityClaim{}).
		Where("record_cid = ? AND role = ?", claim.RecordCID, claim.Role).
		Update("valid_until", time.Now().UTC().Add(-time.Hour)).Error)

	failed := identityClaim(types.ClaimStatusFailed, claim.Subject)
	failed.RecordCID = gateOK
	require.NoError(t, db.UpsertIdentityClaim(failed))
	require.Equal(t, types.PolicyEvalStatusSuperseded, loadPolicyEval(t, db, gateOK, policyA.ID).Status)
	require.Empty(t, gateCIDs(t, db), "the old positive verdict must no longer admit the record")
}
