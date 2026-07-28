// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"testing"

	coretypes "github.com/agntcy/dir/api/core/types"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type moduleFixture struct {
	id                uint64
	name              string
	data              map[string]any
	artifactMediaType string
}

func (m *moduleFixture) GetAnnotations() map[string]string { return nil }
func (m *moduleFixture) GetID() uint64                     { return m.id }
func (m *moduleFixture) GetName() string                   { return m.name }
func (m *moduleFixture) GetData() map[string]any           { return m.data }
func (m *moduleFixture) GetArtifactMediaType() string      { return m.artifactMediaType }

type recordFixture struct {
	cid     string
	name    string
	version string
	modules []coretypes.Module
}

func (r *recordFixture) GetCid() string                    { return r.cid }
func (r *recordFixture) GetName() string                   { return r.name }
func (r *recordFixture) GetVersion() string                { return r.version }
func (r *recordFixture) GetDescription() string            { return "" }
func (r *recordFixture) GetSchemaVersion() string          { return "1.0.0" }
func (r *recordFixture) GetCreatedAt() string              { return "2024-01-01T00:00:00Z" }
func (r *recordFixture) GetAuthors() []string              { return nil }
func (r *recordFixture) GetSkills() []coretypes.Skill      { return nil }
func (r *recordFixture) GetLocators() []coretypes.Locator  { return nil }
func (r *recordFixture) GetModules() []coretypes.Module    { return r.modules }
func (r *recordFixture) GetDomains() []coretypes.Domain    { return nil }
func (r *recordFixture) GetAnnotations() map[string]string { return nil }
func (r *recordFixture) GetPreviousRecordCid() string      { return "" }

func TestConvertModules_PersistsArtifactMediaType(t *testing.T) {
	const mediaType = "application/agent-skills+gzip"

	modules := convertModules([]coretypes.Module{
		&moduleFixture{
			id:                10302,
			name:              "core/language_model/agentskills",
			artifactMediaType: mediaType,
		},
	}, "bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi")

	require.Len(t, modules, 1)
	assert.Equal(t, mediaType, modules[0].ArtifactMediaType)
}

func TestAddRecord_PersistsModuleArtifactMediaType(t *testing.T) {
	const (
		cid       = "bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi"
		mediaType = "application/agent-skills+gzip"
	)

	db, err := New(testDB(t))
	require.NoError(t, err)

	record := &recordFixture{
		cid:     cid,
		name:    "example/skill",
		version: "1.0.0",
		modules: []coretypes.Module{
			&moduleFixture{
				id:                10302,
				name:              "core/language_model/agentskills",
				artifactMediaType: mediaType,
			},
		},
	}

	require.NoError(t, db.AddRecord(record))

	var modules []Module
	require.NoError(t, db.gormDB.Where("record_cid = ?", cid).Find(&modules).Error)
	require.Len(t, modules, 1)
	assert.Equal(t, mediaType, modules[0].ArtifactMediaType)
}

func testDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec("PRAGMA foreign_keys = ON").Error)

	return db
}
