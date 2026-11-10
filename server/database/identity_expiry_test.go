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

//nolint:gosmopolitan // Regression test deliberately simulates non-UTC host timezones.
func TestAGNTCYExpiredResultsLeaveVerifiedSearch(t *testing.T) {
	original := time.Local

	t.Cleanup(func() { time.Local = original })

	for _, zone := range []string{"UTC", "Europe/Paris", "America/New_York", "Asia/Tokyo"} {
		t.Run(zone, func(t *testing.T) {
			t.Setenv("TZ", zone)
			location, err := time.LoadLocation(zone)
			require.NoError(t, err)

			time.Local = location
			for _, storedZone := range []*time.Location{time.UTC, location} {
				t.Run(storedZone.String(), func(t *testing.T) { checkExpiredResults(t, storedZone) })
			}
		})
	}
}

func checkExpiredResults(t *testing.T, storedZone *time.Location) {
	t.Helper()
	db := setupTestDB(t)
	seedDB(t, db)

	past, future := time.Now().In(storedZone).Add(-2*time.Hour), time.Now().In(storedZone).Add(30*time.Minute)
	for _, row := range []gormdb.IdentityClaim{
		{RecordCID: marketingAgent.GetCid(), Role: types.ClaimRoleIdentity, Subject: "agntcy://Agent", Status: types.ClaimStatusVerified, ValidUntil: &past},
		{RecordCID: healthcareAgent.GetCid(), Role: types.ClaimRoleIdentity, Subject: "agntcy://Agent", Status: types.ClaimStatusVerified, ValidUntil: &future},
	} {
		require.NoError(t, db.UpsertIdentityClaim(&persistedObservation{IdentityClaim: row}))
		row.Role = types.ClaimRoleOwner
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

	// Ownership filters must agree with identity filters in every timezone.
	owners, err := db.GetRecordCIDs(types.WithOwners("agntcy://Agent"), types.WithOwnerVerified())
	require.NoError(t, err)
	assert.Equal(t, []string{healthcareAgent.GetCid()}, owners)

	unverified, err := db.GetRecordCIDs(types.WithOwners("agntcy://Agent"), types.WithVerified(false))
	require.NoError(t, err)
	assert.Equal(t, []string{marketingAgent.GetCid()}, unverified)
	stored, err := db.GetIdentityClaimByCID(healthcareAgent.GetCid(), types.ClaimRoleIdentity)
	require.NoError(t, err)

	validity, ok := stored.(types.IdentityClaimValidity)
	require.True(t, ok)

	_, offset := validity.GetValidUntil().Zone()
	assert.Zero(t, offset, "deadline stored in UTC")

	// Existing unbounded schemes still work, and renewal clears a previous deadline.
	seedClaim(t, db, marketingAgent.GetCid(), types.ClaimRoleIdentity, "did:web:example.org", types.ClaimStatusVerified)
	result, err = db.GetIdentityClaimByCID(marketingAgent.GetCid(), types.ClaimRoleIdentity)
	require.NoError(t, err)

	observation, ok = result.(types.IdentityClaimValidity)
	require.True(t, ok)
	assert.True(t, observation.GetValidUntil().IsZero())
}
