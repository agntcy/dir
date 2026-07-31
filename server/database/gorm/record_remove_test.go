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

func TestRemoveRecord_CascadesForeignKeys(t *testing.T) {
	gdb, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, gdb.Exec("PRAGMA foreign_keys = ON").Error)

	db := &DB{gormDB: gdb}
	require.NoError(t, db.migrate())

	now := time.Now().UTC()
	cid := "baeareitestcascade0000000000000000000000000000000000000000000000"
	otherCID := "baeareitestcascadeother00000000000000000000000000000000000000"

	seedRecord(t, db, cid, "cascade-test", "signer-1", "key-1", now)
	seedRecord(t, db, otherCID, "other-record", "signer-2", "key-2", now)

	require.NoError(t, db.RemoveRecord(cid))

	models := []any{
		&Record{},
		&Skill{},
		&Locator{},
		&Module{},
		&Domain{},
		&Annotation{},
		&SignatureVerification{},
		&NameVerification{},
		&ScanReport{},
		&RecordUsageMetrics{},
	}

	for _, model := range models {
		requireZeroRowsForCID(t, db.gormDB, model, cid)
		requireOneRowForCID(t, db.gormDB, model, otherCID)
	}
}

func seedRecord(t *testing.T, db *DB, cid, name, signerKey, keyID string, now time.Time) {
	t.Helper()

	require.NoError(t, db.gormDB.Create(&Record{
		RecordCID: cid,
		Name:      name,
		Version:   "1.0.0",
		Signed:    true,
		Skills: []Skill{
			{SkillID: 10201, Name: "nlp/text_completion"},
		},
		Locators: []Locator{
			{Type: "docker_image", URL: "https://example.com/agent"},
		},
		Modules: []Module{
			{Name: "core/llm/model", ModuleID: 1},
		},
		Domains: []Domain{
			{DomainID: 102, Name: "technology/software_engineering"},
		},
		Annotations: []Annotation{
			{Key: "env", Value: "test"},
		},
		Signatures: []SignatureVerification{
			{
				RecordCID:   cid,
				SignerKey:   signerKey,
				Status:      "verified",
				ContentType: "application/vnd.oci.image.manifest.v1+json",
				CreatedAt:   now,
				UpdatedAt:   now,
			},
		},
		NameVerification: &NameVerification{
			Method: "wellknown",
			Status: VerificationStatusVerified,
			KeyID:  keyID,
		},
	}).Error)

	require.NoError(t, db.UpsertScanReport(&ScanReport{
		RecordCID:   cid,
		ScannerType: "MCP",
		IsSafe:      true,
		MaxSeverity: "NONE",
	}))
	require.NoError(t, db.IncrementPullCount(cid))
}

func requireZeroRowsForCID(t *testing.T, gdb *gorm.DB, model any, cid string) {
	t.Helper()

	var count int64
	require.NoError(t, gdb.Model(model).Where("record_cid = ?", cid).Count(&count).Error)
	require.Zero(t, count, "expected no rows for record_cid=%s in %T", cid, model)
}

func requireOneRowForCID(t *testing.T, gdb *gorm.DB, model any, cid string) {
	t.Helper()

	var count int64
	require.NoError(t, gdb.Model(model).Where("record_cid = ?", cid).Count(&count).Error)
	require.Equal(t, int64(1), count, "expected one row for record_cid=%s in %T", cid, model)
}
