// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"testing"
	"time"

	"github.com/agntcy/dir/server/types"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// fakeIdentityClaim satisfies types.ClaimObject for exercising UpsertClaim
// without depending on the concrete gorm row type.
type fakeIdentityClaim struct {
	cid, subject, status, errMsg string
	verifiedAt                   *time.Time
}

func (f fakeIdentityClaim) GetRecordCID() string      { return f.cid }
func (f fakeIdentityClaim) GetSubject() string        { return f.subject }
func (f fakeIdentityClaim) GetStatus() string         { return f.status }
func (f fakeIdentityClaim) GetError() string          { return f.errMsg }
func (f fakeIdentityClaim) GetVerifiedAt() *time.Time { return f.verifiedAt }

func newIdentityTestDB(t *testing.T) *DB {
	t.Helper()

	gdb, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	require.NoError(t, err)

	db := &DB{gormDB: gdb}
	require.NoError(t, db.migrate())

	return db
}

func TestUpsertClaim_CreateThenUpdate_IsIdempotentPerRecordAndRole(t *testing.T) {
	db := newIdentityTestDB(t)

	require.NoError(t, db.gormDB.Create(&Record{RecordCID: "cid-1", Name: "n", Version: "v"}).Error)

	require.NoError(t, db.UpsertClaim(types.ClaimRoleIdentity, fakeIdentityClaim{cid: "cid-1", subject: "did:web:acme.com", status: ClaimStatusFailed, errMsg: "boom"}))

	claim, err := db.GetClaimByCID("cid-1", types.ClaimRoleIdentity)
	require.NoError(t, err)
	assert.Equal(t, ClaimStatusFailed, claim.GetStatus())

	now := time.Now()
	require.NoError(t, db.UpsertClaim(types.ClaimRoleIdentity, fakeIdentityClaim{cid: "cid-1", subject: "did:web:acme.com", status: ClaimStatusVerified, verifiedAt: &now}))

	claim, err = db.GetClaimByCID("cid-1", types.ClaimRoleIdentity)
	require.NoError(t, err)
	assert.Equal(t, ClaimStatusVerified, claim.GetStatus())
	assert.Empty(t, claim.GetError(), "update must clear the previous error message")

	var count int64
	require.NoError(t, db.gormDB.Model(&Claim{}).Where("record_cid = ? AND role = ?", "cid-1", types.ClaimRoleIdentity).Count(&count).Error)
	assert.Equal(t, int64(1), count, "upsert must not create a second row for the same record+role")
}

func TestUpsertClaim_IdentityAndOwnerCoexistPerRecord(t *testing.T) {
	db := newIdentityTestDB(t)

	require.NoError(t, db.gormDB.Create(&Record{RecordCID: "cid-1", Name: "n", Version: "v"}).Error)

	require.NoError(t, db.UpsertClaim(types.ClaimRoleIdentity, fakeIdentityClaim{cid: "cid-1", subject: "did:web:acme.com:agent", status: ClaimStatusVerified}))
	require.NoError(t, db.UpsertClaim(types.ClaimRoleOwner, fakeIdentityClaim{cid: "cid-1", subject: "did:web:acme.com", status: ClaimStatusFailed}))

	identity, err := db.GetClaimByCID("cid-1", types.ClaimRoleIdentity)
	require.NoError(t, err)
	assert.Equal(t, "did:web:acme.com:agent", identity.GetSubject())
	assert.Equal(t, ClaimStatusVerified, identity.GetStatus())

	owner, err := db.GetClaimByCID("cid-1", types.ClaimRoleOwner)
	require.NoError(t, err)
	assert.Equal(t, "did:web:acme.com", owner.GetSubject())
	assert.Equal(t, ClaimStatusFailed, owner.GetStatus())
}

func TestGetClaimByCID_NotFound(t *testing.T) {
	db := newIdentityTestDB(t)

	_, err := db.GetClaimByCID("does-not-exist", types.ClaimRoleIdentity)
	require.ErrorIs(t, err, ErrClaimNotFound)
}

func TestRemoveClaims(t *testing.T) {
	db := newIdentityTestDB(t)

	require.NoError(t, db.gormDB.Create(&Record{RecordCID: "cid-1", Name: "n", Version: "v"}).Error)
	require.NoError(t, db.UpsertClaim(types.ClaimRoleIdentity, fakeIdentityClaim{cid: "cid-1", subject: "did:web:acme.com", status: ClaimStatusVerified}))
	require.NoError(t, db.UpsertClaim(types.ClaimRoleOwner, fakeIdentityClaim{cid: "cid-1", subject: "did:web:acme.com", status: ClaimStatusVerified}))

	require.NoError(t, db.RemoveClaims("cid-1"))

	_, err := db.GetClaimByCID("cid-1", types.ClaimRoleIdentity)
	require.ErrorIs(t, err, ErrClaimNotFound)

	_, err = db.GetClaimByCID("cid-1", types.ClaimRoleOwner)
	require.ErrorIs(t, err, ErrClaimNotFound)
}

func TestGetRecordCIDs_FilterByOwnerWildcard(t *testing.T) {
	db := newIdentityTestDB(t)

	require.NoError(t, db.gormDB.Create(&Record{RecordCID: "cid-acme", Name: "acme-agent", Version: "v"}).Error)
	require.NoError(t, db.gormDB.Create(&Record{RecordCID: "cid-other", Name: "other-agent", Version: "v"}).Error)

	require.NoError(t, db.UpsertClaim(types.ClaimRoleOwner, fakeIdentityClaim{cid: "cid-acme", subject: "did:web:acme.com:agents:finance", status: ClaimStatusVerified}))
	require.NoError(t, db.UpsertClaim(types.ClaimRoleOwner, fakeIdentityClaim{cid: "cid-other", subject: "spiffe://other.org/agent", status: ClaimStatusVerified}))

	cids, err := db.GetRecordCIDs(types.WithOwners("did:web:acme.com:*"))
	require.NoError(t, err)
	assert.Contains(t, cids, "cid-acme")
	assert.NotContains(t, cids, "cid-other")
}

func TestGetRecordCIDs_FilterByOwnerVerified(t *testing.T) {
	db := newIdentityTestDB(t)

	require.NoError(t, db.gormDB.Create(&Record{RecordCID: "cid-verified", Name: "n1", Version: "v"}).Error)
	require.NoError(t, db.gormDB.Create(&Record{RecordCID: "cid-failed", Name: "n2", Version: "v"}).Error)

	require.NoError(t, db.UpsertClaim(types.ClaimRoleOwner, fakeIdentityClaim{cid: "cid-verified", subject: "did:web:acme.com", status: ClaimStatusVerified}))
	require.NoError(t, db.UpsertClaim(types.ClaimRoleOwner, fakeIdentityClaim{cid: "cid-failed", subject: "did:web:acme.com", status: ClaimStatusFailed}))

	verifiedCIDs, err := db.GetRecordCIDs(types.WithOwnerVerified(true))
	require.NoError(t, err)
	assert.Contains(t, verifiedCIDs, "cid-verified")
	assert.NotContains(t, verifiedCIDs, "cid-failed")

	unverifiedCIDs, err := db.GetRecordCIDs(types.WithOwnerVerified(false))
	require.NoError(t, err)
	assert.Contains(t, unverifiedCIDs, "cid-failed")
	assert.NotContains(t, unverifiedCIDs, "cid-verified")
}

func TestGetRecordCIDs_ExcludeOwner(t *testing.T) {
	db := newIdentityTestDB(t)

	require.NoError(t, db.gormDB.Create(&Record{RecordCID: "cid-acme", Name: "n1", Version: "v"}).Error)
	require.NoError(t, db.gormDB.Create(&Record{RecordCID: "cid-other", Name: "n2", Version: "v"}).Error)

	require.NoError(t, db.UpsertClaim(types.ClaimRoleOwner, fakeIdentityClaim{cid: "cid-acme", subject: "did:web:acme.com:agents:finance", status: ClaimStatusVerified}))
	require.NoError(t, db.UpsertClaim(types.ClaimRoleOwner, fakeIdentityClaim{cid: "cid-other", subject: "spiffe://other.org/agent", status: ClaimStatusVerified}))

	cids, err := db.GetRecordCIDs(types.WithoutOwners("did:web:acme.com:*"))
	require.NoError(t, err)
	assert.NotContains(t, cids, "cid-acme")
	assert.Contains(t, cids, "cid-other")
}
