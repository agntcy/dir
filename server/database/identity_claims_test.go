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

// seedClaim writes one claim result for a record.
func seedClaim(t *testing.T, db *gormdb.DB, cid, role, subject, status string) {
	t.Helper()

	require.NoError(t, db.UpsertIdentityClaim(&gormdb.IdentityClaim{
		RecordCID: cid,
		Role:      role,
		Subject:   subject,
		Status:    status,
	}))
}

// seedClaims gives each seeded record a different claim picture:
//   - marketing: verified identity and verified owner
//   - healthcare: failed identity, verified owner
//   - code: no claims at all.
func seedClaims(t *testing.T, db *gormdb.DB) {
	t.Helper()

	seedClaim(t, db, marketingAgent.GetCid(), types.ClaimRoleIdentity, "did:web:acme.com:agents:marketing", types.ClaimStatusVerified)
	seedClaim(t, db, marketingAgent.GetCid(), types.ClaimRoleOwner, "did:web:acme.com", types.ClaimStatusVerified)
	seedClaim(t, db, healthcareAgent.GetCid(), types.ClaimRoleIdentity, "spiffe://hospital.org/agents/health", types.ClaimStatusFailed)
	seedClaim(t, db, healthcareAgent.GetCid(), types.ClaimRoleOwner, "dns:hospital.org", types.ClaimStatusVerified)
}

func TestIdentityClaim_UpsertAndGet(t *testing.T) {
	db := setupTestDB(t)
	seedDB(t, db)

	cid := marketingAgent.GetCid()
	verifiedAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	require.NoError(t, db.UpsertIdentityClaim(&gormdb.IdentityClaim{
		RecordCID: cid, Role: types.ClaimRoleIdentity, Subject: "did:web:acme.com:a",
		Status: types.ClaimStatusFailed, Error: "no key found", VerifiedAt: verifiedAt,
	}))

	got, err := db.GetIdentityClaimByCID(cid, types.ClaimRoleIdentity)
	require.NoError(t, err)
	assert.Equal(t, cid, got.GetRecordCID())
	assert.Equal(t, types.ClaimRoleIdentity, got.GetRole())
	assert.Equal(t, "did:web:acme.com:a", got.GetSubject())
	assert.Equal(t, types.ClaimStatusFailed, got.GetStatus())
	assert.Equal(t, "no key found", got.GetError())
	assert.True(t, verifiedAt.Equal(got.GetVerifiedAt()), "verified_at %v", got.GetVerifiedAt())

	// A second result for the same (record, role) replaces the first.
	require.NoError(t, db.UpsertIdentityClaim(&gormdb.IdentityClaim{
		RecordCID: cid, Role: types.ClaimRoleIdentity, Subject: "did:web:acme.com:b",
		Status: types.ClaimStatusVerified, VerifiedAt: verifiedAt.Add(time.Hour),
	}))

	got, err = db.GetIdentityClaimByCID(cid, types.ClaimRoleIdentity)
	require.NoError(t, err)
	assert.Equal(t, "did:web:acme.com:b", got.GetSubject())
	assert.Equal(t, types.ClaimStatusVerified, got.GetStatus())
	assert.Empty(t, got.GetError())

	// The other role is untouched by it.
	_, err = db.GetIdentityClaimByCID(cid, types.ClaimRoleOwner)
	require.ErrorIs(t, err, gormdb.ErrIdentityClaimNotFound)
}

func TestIdentityClaim_RolesAreIndependent(t *testing.T) {
	db := setupTestDB(t)
	seedDB(t, db)

	cid := marketingAgent.GetCid()
	seedClaim(t, db, cid, types.ClaimRoleIdentity, "did:web:acme.com:a", types.ClaimStatusVerified)
	seedClaim(t, db, cid, types.ClaimRoleOwner, "did:web:acme.com", types.ClaimStatusFailed)

	identity, err := db.GetIdentityClaimByCID(cid, types.ClaimRoleIdentity)
	require.NoError(t, err)
	assert.Equal(t, "did:web:acme.com:a", identity.GetSubject())

	owner, err := db.GetIdentityClaimByCID(cid, types.ClaimRoleOwner)
	require.NoError(t, err)
	assert.Equal(t, "did:web:acme.com", owner.GetSubject())
	assert.Equal(t, types.ClaimStatusFailed, owner.GetStatus())
}

func TestIdentityClaim_NotFound(t *testing.T) {
	db := setupTestDB(t)
	seedDB(t, db)

	_, err := db.GetIdentityClaimByCID(marketingAgent.GetCid(), types.ClaimRoleIdentity)
	require.ErrorIs(t, err, gormdb.ErrIdentityClaimNotFound)
}

func TestIdentityClaim_UpsertRejectsInvalid(t *testing.T) {
	db := setupTestDB(t)
	seedDB(t, db)

	cid := marketingAgent.GetCid()

	require.Error(t, db.UpsertIdentityClaim(&gormdb.IdentityClaim{RecordCID: cid, Role: "admin", Subject: "x", Status: types.ClaimStatusVerified}))
	require.Error(t, db.UpsertIdentityClaim(&gormdb.IdentityClaim{RecordCID: cid, Role: types.ClaimRoleOwner, Subject: "x", Status: "pending"}))
}

func TestIdentityClaim_DefaultsVerifiedAt(t *testing.T) {
	db := setupTestDB(t)
	seedDB(t, db)

	before := time.Now().Add(-time.Second)

	seedClaim(t, db, marketingAgent.GetCid(), types.ClaimRoleIdentity, "did:web:acme.com:a", types.ClaimStatusVerified)

	got, err := db.GetIdentityClaimByCID(marketingAgent.GetCid(), types.ClaimRoleIdentity)
	require.NoError(t, err)
	assert.True(t, got.GetVerifiedAt().After(before), "verified_at %v", got.GetVerifiedAt())
}

func TestIdentityClaim_RemovedWithRecord(t *testing.T) {
	db := setupTestDB(t)
	seedDB(t, db)
	seedClaims(t, db)

	require.NoError(t, db.RemoveRecord(marketingAgent.GetCid()))

	_, err := db.GetIdentityClaimByCID(marketingAgent.GetCid(), types.ClaimRoleIdentity)
	require.ErrorIs(t, err, gormdb.ErrIdentityClaimNotFound)

	// Other records keep theirs.
	_, err = db.GetIdentityClaimByCID(healthcareAgent.GetCid(), types.ClaimRoleIdentity)
	require.NoError(t, err)
}

func TestGetRecordCIDs_Identity(t *testing.T) {
	db := setupTestDB(t)
	seedDB(t, db)
	seedClaims(t, db)

	marketing, healthcare, code := marketingAgent.GetCid(), healthcareAgent.GetCid(), codeAssistant.GetCid()

	tests := map[string]struct {
		opts []types.FilterOption
		want []string
	}{
		"exact subject":                    {[]types.FilterOption{types.WithIdentities("did:web:acme.com:agents:marketing")}, []string{marketing}},
		"wildcard subject":                 {[]types.FilterOption{types.WithIdentities("did:web:*")}, []string{marketing}},
		"any of several":                   {[]types.FilterOption{types.WithIdentities("did:web:*", "spiffe://*")}, []string{marketing, healthcare}},
		"failed claim matches":             {[]types.FilterOption{types.WithIdentities("spiffe://hospital.org/*")}, []string{healthcare}},
		"no match":                         {[]types.FilterOption{types.WithIdentities("did:web:other.com")}, nil},
		"owner subject is not an identity": {[]types.FilterOption{types.WithIdentities("did:web:acme.com")}, nil},
		"excluded":                         {[]types.FilterOption{types.WithoutIdentities("did:web:*")}, []string{healthcare, code}},
		"excluded keeps unclaimed":         {[]types.FilterOption{types.WithoutIdentities("*")}, []string{code}},
		"include and exclude":              {[]types.FilterOption{types.WithIdentities("*"), types.WithoutIdentities("spiffe://*")}, []string{marketing}},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			cids, err := db.GetRecordCIDs(tt.opts...)
			require.NoError(t, err)
			assert.ElementsMatch(t, tt.want, cids)
		})
	}
}

func TestGetRecordCIDs_Owner(t *testing.T) {
	db := setupTestDB(t)
	seedDB(t, db)
	seedClaims(t, db)

	marketing, healthcare, code := marketingAgent.GetCid(), healthcareAgent.GetCid(), codeAssistant.GetCid()

	tests := map[string]struct {
		opts []types.FilterOption
		want []string
	}{
		"exact subject":                    {[]types.FilterOption{types.WithOwners("did:web:acme.com")}, []string{marketing}},
		"wildcard subject":                 {[]types.FilterOption{types.WithOwners("*.org")}, []string{healthcare}},
		"dns subject":                      {[]types.FilterOption{types.WithOwners("dns:*")}, []string{healthcare}},
		"identity subject is not an owner": {[]types.FilterOption{types.WithOwners("did:web:acme.com:agents:marketing")}, nil},
		"excluded":                         {[]types.FilterOption{types.WithoutOwners("did:web:acme.com")}, []string{healthcare, code}},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			cids, err := db.GetRecordCIDs(tt.opts...)
			require.NoError(t, err)
			assert.ElementsMatch(t, tt.want, cids)
		})
	}
}

func TestGetRecordCIDs_IdentityVerified(t *testing.T) {
	db := setupTestDB(t)
	seedDB(t, db)
	seedClaims(t, db)

	marketing, healthcare, code := marketingAgent.GetCid(), healthcareAgent.GetCid(), codeAssistant.GetCid()

	tests := map[string]struct {
		opts []types.FilterOption
		want []string
	}{
		"identity verified":            {[]types.FilterOption{types.WithIdentityVerified(true)}, []string{marketing}},
		"identity not verified":        {[]types.FilterOption{types.WithIdentityVerified(false)}, []string{healthcare, code}},
		"owner verified":               {[]types.FilterOption{types.WithOwnerVerified(true)}, []string{marketing, healthcare}},
		"owner not verified":           {[]types.FilterOption{types.WithOwnerVerified(false)}, []string{code}},
		"both verified":                {[]types.FilterOption{types.WithIdentityVerified(true), types.WithOwnerVerified(true)}, []string{marketing}},
		"owner verified, identity not": {[]types.FilterOption{types.WithOwnerVerified(true), types.WithIdentityVerified(false)}, []string{healthcare}},
		"verified with subject":        {[]types.FilterOption{types.WithOwners("dns:*"), types.WithOwnerVerified(true)}, []string{healthcare}},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			cids, err := db.GetRecordCIDs(tt.opts...)
			require.NoError(t, err)
			assert.ElementsMatch(t, tt.want, cids)
		})
	}
}

// A record with a verified claim must be counted once however many claims match.
func TestCountRecords_IdentityFiltersDoNotDuplicate(t *testing.T) {
	db := setupTestDB(t)
	seedDB(t, db)
	seedClaims(t, db)

	count, err := db.CountRecords(types.WithIdentities("*"), types.WithOwners("*"), types.WithOwnerVerified(true))
	require.NoError(t, err)
	assert.Equal(t, uint32(2), count)
}
