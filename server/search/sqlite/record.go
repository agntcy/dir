// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sqlite

import (
	"errors"
	"fmt"

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

func (r *Record) GetName() string {
	return r.Name
}

func (r *Record) GetVersion() string {
	return r.Version
}

func (r *Record) GetCID() string {
	return r.CID
}

func (r *Record) GetSkillObjects() []types.SkillObject {
	skills := make([]types.SkillObject, len(r.Skills))
	for i, skill := range r.Skills {
		skills[i] = &skill
	}

	return skills
}

func (r *Record) GetLocatorObjects() []types.LocatorObject {
	locators := make([]types.LocatorObject, len(r.Locators))
	for i, locator := range r.Locators {
		locators[i] = &locator
	}

	return locators
}

func (r *Record) GetExtensionObjects() []types.ExtensionObject {
	extensions := make([]types.ExtensionObject, len(r.Extensions))
	for i, extension := range r.Extensions {
		extensions[i] = &extension
	}

	return extensions
}

func (d *DB) addRecordTx(tx *gorm.DB, recordObject types.RecordObject) (uint, error) {
	record := &Record{
		Name:    recordObject.GetName(),
		Version: recordObject.GetVersion(),
		CID:     recordObject.GetCID(),
	}

	if err := tx.Create(record).Error; err != nil {
		return 0, fmt.Errorf("failed to add record to SQLite search database: %w", err)
	}

	logger.Debug("Added record to SQLite search database", "record_id", record.ID)

	return record.ID, nil
}

func (d *DB) AddRecord(record types.RecordObject) error {
	err := d.gormDB.Transaction(func(tx *gorm.DB) error {
		id, err := d.addRecordTx(tx, record)
		if err != nil {
			return fmt.Errorf("failed to add record to search index: %w", err)
		}

		for _, extension := range record.GetExtensionObjects() {
			if _, err = d.addExtensionTx(tx, extension, id); err != nil {
				return fmt.Errorf("failed to add extension to search index: %w", err)
			}
		}

		for _, locator := range record.GetLocatorObjects() {
			if _, err = d.addLocatorTx(tx, locator, id); err != nil {
				return fmt.Errorf("failed to add locator to search index: %w", err)
			}
		}

		for _, skill := range record.GetSkillObjects() {
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
func (d *DB) GetRecords(opts ...types.FilterOption) ([]types.RecordObject, error) { //nolint:cyclop
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
	query := d.gormDB.Model(&Record{})

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

	// Convert to RecordObject interfaces.
	result := make([]types.RecordObject, len(dbRecords))
	for i := range dbRecords {
		result[i] = &dbRecords[i]
	}

	return result, nil
}
