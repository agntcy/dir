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

// One record per case:
//   - sigVerified: verified only
//   - sigMixed: one verified and one failed signer (counts as verified)
//   - sigFailed: failed only (verified before, no longer does)
//   - sigUnsigned: no signature attached
//   - sigPending: signed, but no verification row yet (matches no status)
const (
	sigVerified = "bafybeigdyrztsigstatus000000000000000000000000000000000001"
	sigMixed    = "bafybeigdyrztsigstatus000000000000000000000000000000000002"
	sigFailed   = "bafybeigdyrztsigstatus000000000000000000000000000000000003"
	sigUnsigned = "bafybeigdyrztsigstatus000000000000000000000000000000000004"
	sigPending  = "bafybeigdyrztsigstatus000000000000000000000000000000000005"
)

func setupSignatureStatusDB(t *testing.T) *DB {
	t.Helper()

	gdb, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	require.NoError(t, err)

	db := &DB{gormDB: gdb}
	require.NoError(t, db.migrate())

	for _, r := range []struct {
		cid    string
		signed bool
	}{
		{sigVerified, true},
		{sigMixed, true},
		{sigFailed, true},
		{sigUnsigned, false},
		{sigPending, true},
	} {
		require.NoError(t, db.gormDB.Create(&Record{
			RecordCID: r.cid,
			Name:      "directory.agntcy.org/test/" + r.cid[len(r.cid)-1:],
			Version:   "1.0.0",
			Signed:    r.signed,
		}).Error)
	}

	for _, row := range []struct{ cid, signer, status string }{
		{sigVerified, "key-a", "verified"},
		{sigMixed, "key-a", "verified"},
		{sigMixed, "key-b", "failed"},
		{sigFailed, "key-a", "failed"},
	} {
		now := time.Now()
		require.NoError(t, db.UpsertSignatureVerification(&SignatureVerification{
			RecordCID:   row.cid,
			SignerKey:   row.signer,
			Status:      row.status,
			ContentType: "application/vnd.oci.image.manifest.v1+json",
			CreatedAt:   now,
			UpdatedAt:   now,
		}))
	}

	return db
}

func TestGetRecordCIDs_SignatureStatuses(t *testing.T) {
	db := setupSignatureStatusDB(t)

	tests := []struct {
		name     string
		statuses []string
		want     []string
	}{
		{"verified includes records with any verified signer", []string{types.SignatureStatusVerified}, []string{sigVerified, sigMixed}},
		{"failed requires rows and no verified signer", []string{types.SignatureStatusFailed}, []string{sigFailed}},
		{"unsigned is a record with no signature attached", []string{types.SignatureStatusUnsigned}, []string{sigUnsigned}},
		{"statuses combine with OR", []string{types.SignatureStatusFailed, types.SignatureStatusUnsigned}, []string{sigFailed, sigUnsigned}},
		{"unrecognized status matches nothing", []string{"bogus"}, []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cids, err := db.GetRecordCIDs(types.WithSignatureStatuses(tt.statuses...))
			require.NoError(t, err)
			assert.ElementsMatch(t, tt.want, cids)
		})
	}
}

// A signed record without a verification result may simply not be verified
// yet, so no status may select it for pruning.
func TestGetRecordCIDs_SignatureStatuses_ExcludeSignedWithoutResult(t *testing.T) {
	db := setupSignatureStatusDB(t)

	cids, err := db.GetRecordCIDs(types.WithSignatureStatuses(types.SignatureStatuses()...))
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{sigVerified, sigMixed, sigFailed, sigUnsigned}, cids)
	assert.NotContains(t, cids, sigPending)
}

// The OR of statuses must stay grouped when ANDed with other filters.
func TestGetRecordCIDs_SignatureStatuses_CombinedWithOtherFilters(t *testing.T) {
	db := setupSignatureStatusDB(t)

	cids, err := db.GetRecordCIDs(
		types.WithSignatureStatuses(types.SignatureStatusFailed, types.SignatureStatusUnsigned),
		types.WithNames("*test/4"),
	)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{sigUnsigned}, cids)
}
