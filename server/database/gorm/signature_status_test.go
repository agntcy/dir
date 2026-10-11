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

// One record per signature status:
//   - sigVerified: verified only
//   - sigMixed: one verified and one failed signer (counts as verified)
//   - sigFailed: failed only
//   - sigUnknown: no verification rows
const (
	sigVerified = "bafybeigdyrztsigstatus000000000000000000000000000000000001"
	sigMixed    = "bafybeigdyrztsigstatus000000000000000000000000000000000002"
	sigFailed   = "bafybeigdyrztsigstatus000000000000000000000000000000000003"
	sigUnknown  = "bafybeigdyrztsigstatus000000000000000000000000000000000004"
)

func setupSignatureStatusDB(t *testing.T) *DB {
	t.Helper()

	gdb, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	require.NoError(t, err)

	db := &DB{gormDB: gdb}
	require.NoError(t, db.migrate())

	for _, cid := range []string{sigVerified, sigMixed, sigFailed, sigUnknown} {
		require.NoError(t, db.gormDB.Create(&Record{
			RecordCID: cid,
			Name:      "directory.agntcy.org/test/" + cid[len(cid)-1:],
			Version:   "1.0.0",
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
		{"unknown is the absence of any row", []string{types.SignatureStatusUnknown}, []string{sigUnknown}},
		{"statuses combine with OR", []string{types.SignatureStatusFailed, types.SignatureStatusUnknown}, []string{sigFailed, sigUnknown}},
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

// [failed, unknown] must select exactly what trusted=false selects, so
// operators can switch to signature_statuses without changing what is pruned.
func TestGetRecordCIDs_SignatureStatuses_MatchesUntrusted(t *testing.T) {
	db := setupSignatureStatusDB(t)

	untrusted, err := db.GetRecordCIDs(types.WithTrusted(false))
	require.NoError(t, err)

	failedOrUnknown, err := db.GetRecordCIDs(types.WithSignatureStatuses(types.SignatureStatusFailed, types.SignatureStatusUnknown))
	require.NoError(t, err)

	assert.ElementsMatch(t, untrusted, failedOrUnknown)
}

// The OR of statuses must stay grouped when ANDed with other filters.
func TestGetRecordCIDs_SignatureStatuses_CombinedWithOtherFilters(t *testing.T) {
	db := setupSignatureStatusDB(t)

	cids, err := db.GetRecordCIDs(
		types.WithSignatureStatuses(types.SignatureStatusFailed, types.SignatureStatusUnknown),
		types.WithNames("*test/4"),
	)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{sigUnknown}, cids)
}
