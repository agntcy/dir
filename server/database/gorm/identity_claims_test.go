// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestMigrate_IndexesIdentityClaimSubjectCaseInsensitively(t *testing.T) {
	gdb, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	require.NoError(t, err)

	db := &DB{gormDB: gdb}
	require.NoError(t, db.migrate())

	var sql string
	require.NoError(t, gdb.Raw("SELECT sql FROM sqlite_master WHERE type = 'index' AND name = 'idx_identity_claims_role_subject_lower'").Scan(&sql).Error)
	assert.Contains(t, sql, "LOWER(subject)")

	// Migrating again is a no-op.
	require.NoError(t, db.migrate())
}
