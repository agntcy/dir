// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sqlite

import (
	"errors"
	"fmt"
	"time"

	"github.com/agntcy/dir/server/database/utils"
	"github.com/agntcy/dir/server/types"
	"gorm.io/gorm"
)

type Record struct {
	CreatedAt time.Time
	UpdatedAt time.Time
	RecordCID string `gorm:"column:record_cid;primarykey;not null"`
	Name      string `gorm:"not null"`
	Version   string `gorm:"not null"`

	Skills   []Skill   `gorm:"foreignKey:RecordCID;references:RecordCID;constraint:OnDelete:CASCADE"`
	Locators []Locator `gorm:"foreignKey:RecordCID;references:RecordCID;constraint:OnDelete:CASCADE"`
	Modules  []Module  `gorm:"foreignKey:RecordCID;references:RecordCID;constraint:OnDelete:CASCADE"`
	Domains  []Domain  `gorm:"foreignKey:RecordCID;references:RecordCID;constraint:OnDelete:CASCADE"`
}

// Implement central Record interface.
func (r *Record) GetCid() string {
	return r.RecordCID
}

func (r *Record) GetRecordData() (types.RecordData, error) {
	return &RecordDataAdapter{record: r}, nil
}

// RecordDataAdapter adapts SQLite Record to central RecordData interface.
type RecordDataAdapter struct {
	record *Record
}

func (r *RecordDataAdapter) GetAnnotations() map[string]string {
	// SQLite records don't store annotations, return empty map
	return make(map[string]string)
}

func (r *RecordDataAdapter) GetDomains() []types.Domain {
	domains := make([]types.Domain, len(r.record.Domains))
	for i, domain := range r.record.Domains {
		domains[i] = &domain
	}

	return domains
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

func (r *RecordDataAdapter) GetModules() []types.Module {
	modules := make([]types.Module, len(r.record.Modules))
	for i, module := range r.record.Modules {
		modules[i] = &module
	}

	return modules
}

func (r *RecordDataAdapter) GetSignature() types.Signature {
	// SQLite records don't store signature information
	return nil
}

func (r *RecordDataAdapter) GetPreviousRecordCid() string {
	// SQLite records don't store previous record CID
	return ""
}

func (d *DB) AddRecord(record types.Record) error {
	// Extract record data
	recordData, err := record.GetRecordData()
	if err != nil {
		return fmt.Errorf("failed to get record data: %w", err)
	}

	// Get CID
	cid := record.GetCid()

	// Check if record already exists
	var existingRecord Record

	err = d.gormDB.Where("record_cid = ?", cid).First(&existingRecord).Error
	if err == nil {
		// Record exists, skip insert
		logger.Debug("Record already exists in search database, skipping insert", "record_cid", existingRecord.RecordCID, "cid", cid)

		return nil
	}

	// If error is not "record not found", return the error
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("failed to check existing record: %w", err)
	}

	// Build complete Record with all associations
	sqliteRecord := &Record{
		RecordCID: cid,
		Name:      recordData.GetName(),
		Version:   recordData.GetVersion(),
		Skills:    convertSkills(recordData.GetSkills(), cid),
		Locators:  convertLocators(recordData.GetLocators(), cid),
		Modules:   convertModules(recordData.GetModules(), cid),
		Domains:   convertDomains(recordData.GetDomains(), cid),
	}

	// Let GORM handle the entire creation with associations
	if err := d.gormDB.Create(sqliteRecord).Error; err != nil {
		return fmt.Errorf("failed to add record to SQLite database: %w", err)
	}

	logger.Debug("Added new record with associations to SQLite database", "record_cid", sqliteRecord.RecordCID, "cid", cid,
		"skills", len(sqliteRecord.Skills), "locators", len(sqliteRecord.Locators), "modules", len(sqliteRecord.Modules), "domains", len(sqliteRecord.Domains))

	return nil
}

// GetRecords retrieves records based on the provided options.
func (d *DB) GetRecords(opts ...types.FilterOption) ([]types.Record, error) {
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

	// Apply all filters.
	query = d.handleFilterOptions(query, cfg)

	// Execute the query to get records.
	var dbRecords []Record
	if err := query.Preload("Skills").Preload("Locators").Preload("Modules").Preload("Domains").Find(&dbRecords).Error; err != nil {
		return nil, fmt.Errorf("failed to query records: %w", err)
	}

	// Convert to Record interfaces.
	result := make([]types.Record, len(dbRecords))
	for i := range dbRecords {
		result[i] = &dbRecords[i]
	}

	return result, nil
}

// GetRecordCIDs retrieves only record CIDs based on the provided options.
// This is optimized for cases where only CIDs are needed, avoiding expensive joins and preloads.
func (d *DB) GetRecordCIDs(opts ...types.FilterOption) ([]string, error) {
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
	query := d.gormDB.Model(&Record{}).Select("records.record_cid").Distinct()

	// Apply pagination.
	if cfg.Limit > 0 {
		query = query.Limit(cfg.Limit)
	}

	if cfg.Offset > 0 {
		query = query.Offset(cfg.Offset)
	}

	// Apply all filters.
	query = d.handleFilterOptions(query, cfg)

	// Execute the query to get only CIDs (no preloading needed).
	var cids []string
	if err := query.Pluck("record_cid", &cids).Error; err != nil {
		return nil, fmt.Errorf("failed to query record CIDs: %w", err)
	}

	// Return CIDs directly - no need for wrapper objects.
	return cids, nil
}

// RemoveRecord removes a record from the search database by CID.
// Uses CASCADE DELETE to automatically remove related Skills, Locators, and Modules.
func (d *DB) RemoveRecord(cid string) error {
	result := d.gormDB.Where("record_cid = ?", cid).Delete(&Record{})

	if result.Error != nil {
		return fmt.Errorf("failed to remove record from search database: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		// Record not found in search database (might not have been indexed)
		logger.Debug("No record found in search database", "cid", cid)

		return nil // Not an error - might be a storage-only record
	}

	logger.Debug("Removed record from search database", "cid", cid, "rows_affected", result.RowsAffected)

	return nil
}

// handleFilterOptions applies the provided filters to the query.
//
//nolint:gocognit,cyclop,nestif
func (d *DB) handleFilterOptions(query *gorm.DB, cfg *types.RecordFilters) *gorm.DB {
	// Apply record-level filters with wildcard support.
	if cfg.Name != "" {
		condition, arg := utils.BuildSingleWildcardCondition("records.name", cfg.Name)
		query = query.Where(condition, arg)
	}

	if cfg.Version != "" {
		condition, arg := utils.BuildSingleWildcardCondition("records.version", cfg.Version)
		query = query.Where(condition, arg)
	}

	// Handle skill filters with wildcard support.
	if len(cfg.SkillIDs) > 0 || len(cfg.SkillNames) > 0 {
		query = query.Joins("JOIN skills ON skills.record_cid = records.record_cid")

		if len(cfg.SkillIDs) > 0 {
			query = query.Where("skills.skill_id IN ?", cfg.SkillIDs)
		}

		if len(cfg.SkillNames) > 0 {
			condition, args := utils.BuildWildcardCondition("skills.name", cfg.SkillNames)
			if condition != "" {
				query = query.Where(condition, args...)
			}
		}
	}

	// Handle locator filters with wildcard support.
	if len(cfg.LocatorTypes) > 0 || len(cfg.LocatorURLs) > 0 {
		query = query.Joins("JOIN locators ON locators.record_cid = records.record_cid")

		if len(cfg.LocatorTypes) > 0 {
			condition, args := utils.BuildWildcardCondition("locators.type", cfg.LocatorTypes)
			if condition != "" {
				query = query.Where(condition, args...)
			}
		}

		if len(cfg.LocatorURLs) > 0 {
			condition, args := utils.BuildWildcardCondition("locators.url", cfg.LocatorURLs)
			if condition != "" {
				query = query.Where(condition, args...)
			}
		}
	}

	// Handle module filters with wildcard support.
	if len(cfg.ModuleNames) > 0 {
		query = query.Joins("JOIN modules ON modules.record_cid = records.record_cid")

		if len(cfg.ModuleNames) > 0 {
			condition, args := utils.BuildWildcardCondition("modules.name", cfg.ModuleNames)
			if condition != "" {
				query = query.Where(condition, args...)
			}
		}
	}

	// Handle domain filters with wildcard support.
	if len(cfg.DomainIDs) > 0 || len(cfg.DomainNames) > 0 {
		query = query.Joins("JOIN domains ON domains.record_cid = records.record_cid")

		if len(cfg.DomainIDs) > 0 {
			query = query.Where("domains.domain_id IN ?", cfg.DomainIDs)
		}

		if len(cfg.DomainNames) > 0 {
			condition, args := utils.BuildWildcardCondition("domains.name", cfg.DomainNames)
			if condition != "" {
				query = query.Where(condition, args...)
			}
		}
	}

	return query
}
