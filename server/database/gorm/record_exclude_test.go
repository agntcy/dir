// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"testing"

	"github.com/agntcy/dir/server/types"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TestGetRecordCIDs_NegatedDescription_NullSurvives guards against
// BuildNegatedCondition's NULL-safety for the description column: a NULL
// value must count as "does not match" and be retained under negation, not
// silently dropped. This is a white-box test inside package gorm (following
// TestRemoveRecord_CascadesForeignKeys's pattern in record_remove_test.go) so
// it can force a genuine SQL NULL directly via db.gormDB, without adding a
// raw-SQL-execution method to production code just for this one test.
func TestGetRecordCIDs_NegatedDescription_NullSurvives(t *testing.T) {
	gdb, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	require.NoError(t, err)

	db := &DB{gormDB: gdb}
	require.NoError(t, db.migrate())

	withDescription := &Record{
		RecordCID:   "bafybeigdyrztnegdesc0000000000000000000000000000000000001",
		Name:        "directory.agntcy.org/test/has-description",
		Version:     "1.0.0",
		Description: "this system enables real-time fraud detection",
	}
	withoutDescription := &Record{
		RecordCID: "bafybeigdyrztnegdesc0000000000000000000000000000000000002",
		Name:      "directory.agntcy.org/test/no-description",
		Version:   "1.0.0",
	}

	require.NoError(t, db.gormDB.Create(withDescription).Error)
	require.NoError(t, db.gormDB.Create(withoutDescription).Error)

	// AddRecord (via coretypes.Record.GetDescription()) would write '' here, not
	// SQL NULL. Force a genuine NULL directly so this test exercises the
	// "IS NULL OR ..." branch BuildNegatedCondition exists for.
	require.NoError(t, db.gormDB.Exec(
		"UPDATE records SET description = NULL WHERE record_cid = ?", withoutDescription.RecordCID,
	).Error)

	cids, err := db.GetRecordCIDs(types.WithoutDescriptions("*fraud*"))
	require.NoError(t, err)
	assert.NotContains(t, cids, withDescription.RecordCID)
	assert.Contains(t, cids, withoutDescription.RecordCID)
}
