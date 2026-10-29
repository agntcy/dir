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
	gormlib "gorm.io/gorm"
)

func TestGetPolicyFacts(t *testing.T) {
	gdb, err := gormlib.Open(sqlite.Open("file::memory:"), &gormlib.Config{})
	require.NoError(t, err)

	db := &DB{gormDB: gdb}
	require.NoError(t, db.migrate())

	const cid = "bafybeigdyrztpolicyfacts000000000000000000000000000000001"

	require.NoError(t, db.gormDB.Create(&Record{
		RecordCID: cid,
		Name:      "directory.agntcy.org/test/facts",
		Version:   "1.0.0",
		Signed:    true,
	}).Error)

	verifiedAt := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	require.NoError(t, db.UpsertSignatureVerification(&SignatureVerification{
		RecordCID:     cid,
		SignerKey:     "key-a",
		Status:        types.SignatureStatusVerified,
		SignerSubject: "did:web:acme.com",
		SignerIssuer:  "https://issuer.example",
		ContentType:   "application/vnd.oci.image.manifest.v1+json",
		CreatedAt:     verifiedAt,
		UpdatedAt:     verifiedAt,
	}))
	require.NoError(t, db.UpsertScanReport(&ScanReport{
		RecordCID:    cid,
		ScannerType:  "MCP",
		IsSafe:       true,
		MaxSeverity:  "NONE",
		Status:       types.ScanStatusCompleted,
		Source:       types.ScanSourceLocal,
		FindingCount: 3,
	}, types.DefaultScanSchedule()))
	require.NoError(t, db.UpsertIdentityClaim(&IdentityClaim{
		RecordCID:  cid,
		Role:       types.ClaimRoleIdentity,
		Subject:    "did:web:acme.com",
		Status:     types.ClaimStatusVerified,
		VerifiedAt: verifiedAt,
	}))

	missing, err := db.GetPolicyFacts("bafybeigdyrztmissing0000000000000000000000000000000000001")
	require.NoError(t, err)
	assert.Zero(t, missing)

	facts, err := db.GetPolicyFacts(cid)
	require.NoError(t, err)
	assert.True(t, facts.Signed)
	require.Len(t, facts.Signatures, 1)
	assert.Equal(t, types.SignatureStatusVerified, facts.Signatures[0].GetStatus())
	assert.Equal(t, "did:web:acme.com", facts.Signatures[0].GetSignerSubject())
	require.Len(t, facts.Scans, 1)
	assert.Equal(t, "MCP", facts.Scans[0].GetScannerType())
	assert.True(t, facts.Scans[0].GetIsSafe())
	assert.Equal(t, types.ScanSourceLocal, facts.Scans[0].GetSource())
	assert.Equal(t, 3, facts.Scans[0].GetFindingCount())
	require.Len(t, facts.Claims, 1)
	assert.Equal(t, types.ClaimRoleIdentity, facts.Claims[0].GetRole())
	assert.Equal(t, types.ClaimStatusVerified, facts.Claims[0].GetStatus())
	assert.False(t, facts.IndexedAt.IsZero(), "the row's own created_at")
}

// When a record was indexed here is this node's observation. The record's OASF
// created_at is the publisher's, and a policy reading indexed_at must not be
// given a value its author chose.
func TestGetPolicyFacts_IndexedAtIsNotTheRecordsOwnCreatedAt(t *testing.T) {
	gdb, err := gormlib.Open(sqlite.Open("file::memory:"), &gormlib.Config{})
	require.NoError(t, err)

	db := &DB{gormDB: gdb}
	require.NoError(t, db.migrate())

	const cid = "bafybeigdyrztpolicyfacts000000000000000000000000000000002"

	require.NoError(t, db.gormDB.Create(&Record{
		RecordCID:     cid,
		Name:          "directory.agntcy.org/test/asserted",
		Version:       "1.0.0",
		OASFCreatedAt: "1999-01-01T00:00:00Z",
	}).Error)

	facts, err := db.GetPolicyFacts(cid)
	require.NoError(t, err)
	assert.False(t, facts.IndexedAt.IsZero())
	assert.Greater(t, facts.IndexedAt.Year(), 1999, "not the created_at the record asserts")
}
