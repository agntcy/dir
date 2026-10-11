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

// signedAgent has only a verified signer. Together with the shared fixtures it
// covers every signature status:
//   - signedAgent: verified only
//   - marketingAgent: one verified and one failed signer (counts as verified)
//   - healthcareAgent: failed only
//   - codeAssistant: no verification rows (unknown)
var signedAgent = &testRecord{
	cid:           "bafybeihkoviema7g3gxyt6la7vd5ho32ictqbilu3wnlo3rs7ewhnp7lly",
	name:          "directory.agntcy.org/cisco/signed-agent",
	version:       "1.0.0",
	schemaVersion: "0.8.0",
	createdAt:     "2024-04-01T12:00:00Z",
}

func seedSignatureVerification(t *testing.T, db *gormdb.DB, cid, signerKey, status string) {
	t.Helper()

	now := time.Now()

	require.NoError(t, db.UpsertSignatureVerification(&gormdb.SignatureVerification{
		RecordCID:   cid,
		SignerKey:   signerKey,
		Status:      status,
		ContentType: "application/vnd.oci.image.manifest.v1+json",
		CreatedAt:   now,
		UpdatedAt:   now,
	}))
}

func seedSignatureStatuses(t *testing.T) *gormdb.DB {
	t.Helper()

	db := setupTestDB(t)
	seedDB(t, db)
	require.NoError(t, db.AddRecord(signedAgent))

	seedSignatureVerification(t, db, signedAgent.GetCid(), "key-a", "verified")
	seedSignatureVerification(t, db, marketingAgent.GetCid(), "key-a", "verified")
	seedSignatureVerification(t, db, marketingAgent.GetCid(), "key-b", "failed")
	seedSignatureVerification(t, db, healthcareAgent.GetCid(), "key-a", "failed")

	return db
}

func TestGetRecordCIDs_SignatureStatuses(t *testing.T) {
	db := seedSignatureStatuses(t)

	tests := []struct {
		name     string
		statuses []string
		want     []string
	}{
		{
			name:     "verified includes records with any verified signer",
			statuses: []string{types.SignatureStatusVerified},
			want:     []string{signedAgent.GetCid(), marketingAgent.GetCid()},
		},
		{
			name:     "failed requires rows and no verified signer",
			statuses: []string{types.SignatureStatusFailed},
			want:     []string{healthcareAgent.GetCid()},
		},
		{
			name:     "unknown is the absence of any row",
			statuses: []string{types.SignatureStatusUnknown},
			want:     []string{codeAssistant.GetCid()},
		},
		{
			name:     "statuses combine with OR",
			statuses: []string{types.SignatureStatusFailed, types.SignatureStatusUnknown},
			want:     []string{healthcareAgent.GetCid(), codeAssistant.GetCid()},
		},
		{
			name:     "unrecognized status matches nothing",
			statuses: []string{"bogus"},
			want:     []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cids, err := db.GetRecordCIDs(types.WithSignatureStatuses(tt.statuses...))
			require.NoError(t, err)
			assert.ElementsMatch(t, tt.want, cids)
		})
	}
}

// [failed, unknown] must select exactly what trusted=false selects today, so
// operators can switch to signature_statuses without changing what is pruned.
func TestGetRecordCIDs_SignatureStatuses_MatchesUntrusted(t *testing.T) {
	db := seedSignatureStatuses(t)

	untrusted, err := db.GetRecordCIDs(types.WithTrusted(false))
	require.NoError(t, err)

	failedOrUnknown, err := db.GetRecordCIDs(types.WithSignatureStatuses(types.SignatureStatusFailed, types.SignatureStatusUnknown))
	require.NoError(t, err)

	assert.ElementsMatch(t, untrusted, failedOrUnknown)
}

// The OR of statuses must stay grouped when ANDed with other filters.
func TestGetRecordCIDs_SignatureStatuses_CombinedWithOtherFilters(t *testing.T) {
	db := seedSignatureStatuses(t)

	cids, err := db.GetRecordCIDs(
		types.WithSignatureStatuses(types.SignatureStatusFailed, types.SignatureStatusUnknown),
		types.WithNames("*code-assistant"),
	)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{codeAssistant.GetCid()}, cids)
}
