// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package database

import (
	"testing"
	"time"

	gormdb "github.com/agntcy/dir/server/database/gorm"
	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// persistedObservation models a result written while it was valid: its stored
// status remains verified even though its deadline has now elapsed.
type persistedObservation struct{ gormdb.IdentityClaim }

func (c *persistedObservation) GetStatus() string { return types.ClaimStatusVerified }

func TestAGNTCYExpiredResultsLeaveVerifiedSearch(t *testing.T) {
	db := setupTestDB(t)
	seedDB(t, db)

	past, future := time.Now().Add(-time.Minute), time.Now().Add(time.Hour)
	for _, row := range []gormdb.IdentityClaim{
		{RecordCID: marketingAgent.GetCid(), Role: types.ClaimRoleIdentity, Subject: "agntcy://Agent", Status: types.ClaimStatusVerified, ValidUntil: &past},
		{RecordCID: healthcareAgent.GetCid(), Role: types.ClaimRoleIdentity, Subject: "agntcy://Agent", Status: types.ClaimStatusVerified, ValidUntil: &future},
	} {
		require.NoError(t, db.UpsertIdentityClaim(&persistedObservation{IdentityClaim: row}))
	}

	cids, err := db.GetRecordCIDs(types.WithIdentities("agntcy://Agent"), types.WithIdentityVerified())
	require.NoError(t, err)
	assert.Equal(t, []string{healthcareAgent.GetCid()}, cids)

	count, err := db.CountRecords(types.WithIdentities("agntcy://*"), types.WithIdentityVerified())
	require.NoError(t, err)
	assert.EqualValues(t, 1, count)

	result, err := db.GetIdentityClaimByCID(marketingAgent.GetCid(), types.ClaimRoleIdentity)
	require.NoError(t, err)

	row, ok := result.(*gormdb.IdentityClaim)
	require.True(t, ok)
	assert.Equal(t, types.ClaimStatusVerified, row.Status, "stored status is still verified")
	assert.Equal(t, types.ClaimStatusFailed, result.GetStatus(), "read status must honor expiry")
	assert.Contains(t, result.GetError(), "expired")
	observation, ok := result.(types.IdentityClaimValidity)
	require.True(t, ok)
	assert.True(t, observation.GetValidUntil().Equal(past))

	// Existing unbounded schemes still work, and renewal clears a previous deadline.
	seedClaim(t, db, marketingAgent.GetCid(), types.ClaimRoleIdentity, "did:web:example.org", types.ClaimStatusVerified)
	result, err = db.GetIdentityClaimByCID(marketingAgent.GetCid(), types.ClaimRoleIdentity)
	require.NoError(t, err)

	observation, ok = result.(types.IdentityClaimValidity)
	require.True(t, ok)
	assert.True(t, observation.GetValidUntil().IsZero())
}
