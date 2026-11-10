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
