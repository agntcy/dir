// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sqlite

import (
	"errors"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/server/types"
	"gorm.io/gorm"
)

type Record struct {
	gorm.Model
	Name    string `gorm:"not null"`
	Version string `gorm:"not null"`
	CID     string `gorm:"unique;not null"`

	Skills     []Skill     `gorm:"foreignKey:AgentID;constraint:OnDelete:CASCADE"`
	Locators   []Locator   `gorm:"foreignKey:AgentID;constraint:OnDelete:CASCADE"`
	Extensions []Extension `gorm:"foreignKey:AgentID;constraint:OnDelete:CASCADE"`
}

// Implement central Record interface.
func (r *Record) GetCid() string {
	return r.CID
}

func (r *Record) GetRecordData() types.RecordData {
	return &RecordDataAdapter{record: r}
}

// RecordDataAdapter adapts SQLite Record to central RecordData interface.
type RecordDataAdapter struct {
	record *Record
}

func (r *RecordDataAdapter) GetAnnotations() map[string]string {
	// SQLite records don't store annotations, return empty map
	return make(map[string]string)
}

func (r *RecordDataAdapter) GetSchemaVersion() string {
	// Default schema version for search records
	return "v1"
}

func (r *RecordDataAdapter) GetName() string {
	return r.record.Name
}

func (r *RecordDataAdapter) GetVersion() string {
	return r.record.Version
}

func (r *RecordDataAdapter) GetDescription() string {
	// SQLite records don't store description
	return ""
}

func (r *RecordDataAdapter) GetAuthors() []string {
	// SQLite records don't store authors
	return []string{}
}

func (r *RecordDataAdapter) GetCreatedAt() string {
	return r.record.CreatedAt.Format("2006-01-02T15:04:05Z")
}

func (r *RecordDataAdapter) GetSkills() []types.Skill {
	skills := make([]types.Skill, len(r.record.Skills))
	for i, skill := range r.record.Skills {
		skills[i] = &skill
	}

	return skills
}

func (r *RecordDataAdapter) GetLocators() []types.Locator {
	locators := make([]types.Locator, len(r.record.Locators))
	for i, locator := range r.record.Locators {
		locators[i] = &locator
	}

	return locators
}

func (r *RecordDataAdapter) GetExtensions() []types.Extension {
	extensions := make([]types.Extension, len(r.record.Extensions))
	for i, extension := range r.record.Extensions {
		extensions[i] = &extension
	}

	return extensions
}

func (r *RecordDataAdapter) GetSignature() types.Signature {
	// SQLite records don't store signature information
	return nil
}

func (r *RecordDataAdapter) GetPreviousRecordCid() string {
	// SQLite records don't store previous record CID
	return ""
}

func (d *DB) addRecordTx(tx *gorm.DB, record types.Record) (uint, error) {
	recordData := record.GetRecordData()

	sqliteRecord := &Record{
		Name:    recordData.GetName(),
		Version: recordData.GetVersion(),
		CID:     record.GetCid(),
	}

	if err := tx.Create(sqliteRecord).Error; err != nil {
		return 0, fmt.Errorf("failed to add record to SQLite database: %w", err)
	}

	logger.Debug("Added record to SQLite database", "record_id", sqliteRecord.ID)

	return sqliteRecord.ID, nil
}

func (d *DB) AddRecord(record types.Record) error {
	err := d.gormDB.Transaction(func(tx *gorm.DB) error {
		id, err := d.addRecordTx(tx, record)
		if err != nil {
			return fmt.Errorf("failed to add record to search index: %w", err)
		}

		recordData := record.GetRecordData()

		for _, extension := range recordData.GetExtensions() {
			if _, err = d.addExtensionTx(tx, extension, id); err != nil {
				return fmt.Errorf("failed to add extension to search index: %w", err)
			}
		}

		for _, locator := range recordData.GetLocators() {
			if _, err = d.addLocatorTx(tx, locator, id); err != nil {
				return fmt.Errorf("failed to add locator to search index: %w", err)
			}
		}

		for _, skill := range recordData.GetSkills() {
			if _, err = d.addSkillTx(tx, skill, id); err != nil {
				return fmt.Errorf("failed to add skill to search index: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to add record: %w", err)
	}

	return nil
}

// GetRecords retrieves agent records based on the provided options.
func (d *DB) GetRecords(opts ...types.FilterOption) ([]types.Record, error) { //nolint:cyclop
	// Create default configuration.
	cfg := &types.RecordFilters{}

	// Apply all options.
	for _, opt := range opts {
		if opt == nil {
			return nil, errors.New("nil option provided")
		}

		opt(cfg)
	}

	// Start with the base query for records.
	query := d.gormDB.Model(&Record{}).Distinct()

	// Apply pagination.
	if cfg.Limit > 0 {
		query = query.Limit(cfg.Limit)
	}

	if cfg.Offset > 0 {
		query = query.Offset(cfg.Offset)
	}

	// Apply record-level filters.
	if cfg.Name != "" {
		query = query.Where("records.name LIKE ?", "%"+cfg.Name+"%")
	}

	if cfg.Version != "" {
		query = query.Where("records.version = ?", cfg.Version)
	}

	// Handle skill filters.
	if len(cfg.SkillIDs) > 0 || len(cfg.SkillNames) > 0 {
		query = query.Joins("JOIN skills ON skills.agent_id = records.id")

		if len(cfg.SkillIDs) > 0 {
			query = query.Where("skills.skill_id IN ?", cfg.SkillIDs)
		}

		if len(cfg.SkillNames) > 0 {
			query = query.Where("skills.name IN ?", cfg.SkillNames)
		}
	}

	// Handle locator filters.
	if len(cfg.LocatorTypes) > 0 || len(cfg.LocatorURLs) > 0 {
		query = query.Joins("JOIN locators ON locators.agent_id = records.id")

		if len(cfg.LocatorTypes) > 0 {
			query = query.Where("locators.type IN ?", cfg.LocatorTypes)
		}

		if len(cfg.LocatorURLs) > 0 {
			query = query.Where("locators.url IN ?", cfg.LocatorURLs)
		}
	}

	// Handle extension filters.
	if len(cfg.ExtensionNames) > 0 || len(cfg.ExtensionVersions) > 0 {
		query = query.Joins("JOIN extensions ON extensions.agent_id = records.id")

		if len(cfg.ExtensionNames) > 0 {
			query = query.Where("extensions.name IN ?", cfg.ExtensionNames)
		}

		if len(cfg.ExtensionVersions) > 0 {
			query = query.Where("extensions.version IN ?", cfg.ExtensionVersions)
		}
	}

	// Execute the query to get records.
	var dbRecords []Record
	if err := query.Preload("Skills").Preload("Locators").Preload("Extensions").Find(&dbRecords).Error; err != nil {
		return nil, fmt.Errorf("failed to query records: %w", err)
	}

	// Convert to Record interfaces.
	result := make([]types.Record, len(dbRecords))
	for i := range dbRecords {
		result[i] = &dbRecords[i]
	}

	return result, nil
}

// GetRecordRefs retrieves only record references (CIDs) based on the provided options.
// This is optimized for cases where only CIDs are needed, avoiding expensive joins and preloads.
func (d *DB) GetRecordRefs(opts ...types.FilterOption) ([]*corev1.RecordRef, error) { //nolint:cyclop
	// Create default configuration.
	cfg := &types.RecordFilters{}

	// Apply all options.
	for _, opt := range opts {
		if opt == nil {
			return nil, errors.New("nil option provided")
		}

		opt(cfg)
	}

	// Start with the base query for records - only select CID for efficiency.
	query := d.gormDB.Model(&Record{}).Select("CID").Distinct()

	// Apply pagination.
	if cfg.Limit > 0 {
		query = query.Limit(cfg.Limit)
	}

	if cfg.Offset > 0 {
		query = query.Offset(cfg.Offset)
	}

	// Apply record-level filters.
	if cfg.Name != "" {
		query = query.Where("records.name LIKE ?", "%"+cfg.Name+"%")
	}

	if cfg.Version != "" {
		query = query.Where("records.version = ?", cfg.Version)
	}

	// Handle skill filters.
	if len(cfg.SkillIDs) > 0 || len(cfg.SkillNames) > 0 {
		query = query.Joins("JOIN skills ON skills.agent_id = records.id")

		if len(cfg.SkillIDs) > 0 {
			query = query.Where("skills.skill_id IN ?", cfg.SkillIDs)
		}

		if len(cfg.SkillNames) > 0 {
			query = query.Where("skills.name IN ?", cfg.SkillNames)
		}
	}

	// Handle locator filters.
	if len(cfg.LocatorTypes) > 0 || len(cfg.LocatorURLs) > 0 {
		query = query.Joins("JOIN locators ON locators.agent_id = records.id")

		if len(cfg.LocatorTypes) > 0 {
			query = query.Where("locators.type IN ?", cfg.LocatorTypes)
		}

		if len(cfg.LocatorURLs) > 0 {
			query = query.Where("locators.url IN ?", cfg.LocatorURLs)
		}
	}

	// Handle extension filters.
	if len(cfg.ExtensionNames) > 0 || len(cfg.ExtensionVersions) > 0 {
		query = query.Joins("JOIN extensions ON extensions.agent_id = records.id")

		if len(cfg.ExtensionNames) > 0 {
			query = query.Where("extensions.name IN ?", cfg.ExtensionNames)
		}

		if len(cfg.ExtensionVersions) > 0 {
			query = query.Where("extensions.version IN ?", cfg.ExtensionVersions)
		}
	}

	// Execute the query to get only CIDs (no preloading needed).
	var cids []string
	if err := query.Pluck("CID", &cids).Error; err != nil {
		return nil, fmt.Errorf("failed to query record CIDs: %w", err)
	}

	// Convert CIDs to RecordRef objects.
	result := make([]*corev1.RecordRef, len(cids))
	for i, cid := range cids {
		result[i] = &corev1.RecordRef{Cid: cid}
	}

	return result, nil
}
