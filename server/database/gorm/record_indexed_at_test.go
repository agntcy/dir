// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// The indexer decides before it writes the row, so it asks for the time of a
// row that already exists. A CID with no row has no time, which is an answer
// rather than an error: absence is what the policy is told.
func TestGetRecordIndexedAt(t *testing.T) {
	gdb, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	require.NoError(t, err)

	db := &DB{gormDB: gdb}
	require.NoError(t, db.migrate())

	cid := "baeareitestindexedat00000000000000000000000000000000000000000"
	seedRecord(t, db, cid, "indexed-at-test", "signer-1", time.Now().UTC())

	var want time.Time
	require.NoError(t, gdb.Model(&Record{}).Where("record_cid = ?", cid).Pluck("created_at", &want).Error)

	got, err := db.GetRecordIndexedAt(cid)
	require.NoError(t, err)
	require.True(t, want.Equal(got), "want the row's created_at %s, got %s", want, got)

	missing, err := db.GetRecordIndexedAt("baeareinorowsuchrecord000000000000000000000000000000000000000")
	require.NoError(t, err)
	require.True(t, missing.IsZero(), "a record that is not indexed has no index time, got %s", missing)
}
