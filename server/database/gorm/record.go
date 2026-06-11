// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"errors"
	"fmt"
	"strings"
	"time"

	coretypes "github.com/agntcy/dir/api/core/types"
	"github.com/agntcy/dir/server/database/utils"
	"github.com/agntcy/dir/server/types"
	"gorm.io/gorm"
)

var _ coretypes.Record = (*Record)(nil)

type Record struct {
	CreatedAt     time.Time
	UpdatedAt     time.Time
	RecordCID     string   `gorm:"column:record_cid;primarykey;not null"`
	Name          string   `gorm:"not null"`
	Version       string   `gorm:"not null"`
	Description   string   `gorm:"column:description"`
	SchemaVersion string   `gorm:"column:schema_version"`
	OASFCreatedAt string   `gorm:"column:oasf_created_at"`
	Authors       []string `gorm:"column:authors;serializer:json"` // Stored as JSON array
	Signed        bool     `gorm:"column:signed;default:false"`    // Whether at least one signature is attached

	Skills      []Skill                 `gorm:"foreignKey:RecordCID;references:RecordCID;constraint:OnDelete:CASCADE"`
	Locators    []Locator               `gorm:"foreignKey:RecordCID;references:RecordCID;constraint:OnDelete:CASCADE"`
	Modules     []Module                `gorm:"foreignKey:RecordCID;references:RecordCID;constraint:OnDelete:CASCADE"`
	Domains     []Domain                `gorm:"foreignKey:RecordCID;references:RecordCID;constraint:OnDelete:CASCADE"`
	Annotations []Annotation            `gorm:"foreignKey:RecordCID;references:RecordCID;constraint:OnDelete:CASCADE"`
	Signatures  []SignatureVerification `gorm:"foreignKey:RecordCID;references:RecordCID;constraint:OnDelete:CASCADE"`
}

func (r *Record) GetCid() string {
	return r.RecordCID
}

func (r *Record) GetAnnotations() map[string]string {
	annotations := make(map[string]string, len(r.Annotations))
	for _, a := range r.Annotations {
		annotations[a.Key] = a.Value
	}

	return annotations
}

func (r *Record) GetDomains() []coretypes.Domain {
	domains := make([]coretypes.Domain, len(r.Domains))
	for i, domain := range r.Domains {
		domains[i] = &domain
	}

	return domains
}

func (r *Record) GetSchemaVersion() string {
	if r.SchemaVersion != "" {
		return r.SchemaVersion
	}

	// Default schema version for search records
	return "v1"
}

func (r *Record) GetName() string {
	return r.Name
}

func (r *Record) GetVersion() string {
	return r.Version
}

func (r *Record) GetDescription() string {
	return r.Description
}

func (r *Record) GetAuthors() []string {
	return r.Authors
}

func (r *Record) GetCreatedAt() string {
	if r.OASFCreatedAt != "" {
		return r.OASFCreatedAt
	}

	return r.CreatedAt.Format("2006-01-02T15:04:05Z")
}

func (r *Record) GetSkills() []coretypes.Skill {
	skills := make([]coretypes.Skill, len(r.Skills))
	for i, skill := range r.Skills {
		skills[i] = &skill
	}

	return skills
}

func (r *Record) GetLocators() []coretypes.Locator {
	locators := make([]coretypes.Locator, len(r.Locators))
	for i, locator := range r.Locators {
		locators[i] = &locator
	}

	return locators
}

func (r *Record) GetModules() []coretypes.Module {
	modules := make([]coretypes.Module, len(r.Modules))
	for i, module := range r.Modules {
		modules[i] = &module
	}

	return modules
}

func (r *Record) GetPreviousRecordCid() string {
	// Database records don't store previous record CID
	return ""
}

func (d *DB) AddRecord(record coretypes.Record) error {
	// Get CID
	cid := record.GetCid()

	// Check if record already exists
	var existingRecord Record

	err := d.gormDB.Where("record_cid = ?", cid).First(&existingRecord).Error
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
	dbRecord := &Record{
		RecordCID:     cid,
		Name:          record.GetName(),
		Version:       record.GetVersion(),
		Description:   record.GetDescription(),
		SchemaVersion: record.GetSchemaVersion(),
		OASFCreatedAt: record.GetCreatedAt(),
		Authors:       record.GetAuthors(),
		Skills:        convertSkills(record.GetSkills(), cid),
		Locators:      convertLocators(record.GetLocators(), cid),
		Modules:       convertModules(record.GetModules(), cid),
		Domains:       convertDomains(record.GetDomains(), cid),
		Annotations:   convertAnnotations(record.GetAnnotations(), cid),
	}

	// Let GORM handle the entire creation with associations
	if err := d.gormDB.Create(dbRecord).Error; err != nil {
		return fmt.Errorf("failed to add record to database: %w", err)
	}

	logger.Debug("Added new record with associations to database", "record_cid", dbRecord.RecordCID, "cid", cid,
		"skills", len(dbRecord.Skills), "locators", len(dbRecord.Locators), "modules", len(dbRecord.Modules),
		"domains", len(dbRecord.Domains), "annotations", len(dbRecord.Annotations))

	return nil
}

// GetRecords retrieves full records based on the provided filters.
func (d *DB) GetRecords(opts ...types.FilterOption) ([]coretypes.Record, error) {
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

	// Apply all filters.
	query = d.handleFilterOptions(query, cfg)
	query = query.Order("records.created_at DESC")

	// Execute the query.
	var records []Record
	if err := query.Find(&records).Error; err != nil {
		return nil, fmt.Errorf("failed to query records: %w", err)
	}

	// Convert to interface type.
	result := make([]coretypes.Record, len(records))
	for i := range records {
		result[i] = &records[i]
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
	// Remove signature verifications first
	if err := d.gormDB.Where("record_cid = ?", cid).Delete(&SignatureVerification{}).Error; err != nil {
		return fmt.Errorf("failed to remove signature verifications: %w", err)
	}

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
//nolint:gocognit,cyclop,nestif,gocyclo,maintidx
func (d *DB) handleFilterOptions(query *gorm.DB, cfg *types.RecordFilters) *gorm.DB {
	// Filter by CID (exact match on primary key).
	if len(cfg.CIDs) > 0 {
		query = query.Where("records.record_cid IN ?", cfg.CIDs)
	}

	// Apply record-level filters with wildcard support.
	if len(cfg.Names) > 0 {
		condition, args := utils.BuildWildcardCondition("records.name", cfg.Names)
		if condition != "" {
			query = query.Where(condition, args...)
		}
	}

	if len(cfg.Versions) > 0 {
		condition, args := utils.BuildComparisonConditions("records.version", cfg.Versions)
		if condition != "" {
			query = query.Where(condition, args...)
		}
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

	// Handle annotation filters with wildcard support.
	if len(cfg.AnnotationKeys) > 0 || len(cfg.AnnotationValues) > 0 {
		query = query.Joins("JOIN annotations ON annotations.record_cid = records.record_cid")

		if len(cfg.AnnotationKeys) > 0 {
			condition, args := utils.BuildWildcardCondition("annotations.key", cfg.AnnotationKeys)
			if condition != "" {
				query = query.Where(condition, args...)
			}
		}

		if len(cfg.AnnotationValues) > 0 {
			condition, args := utils.BuildWildcardCondition("annotations.value", cfg.AnnotationValues)
			if condition != "" {
				query = query.Where(condition, args...)
			}
		}
	}

	// Handle created_at filter with comparison operator support.
	if len(cfg.CreatedAts) > 0 {
		condition, args := utils.BuildComparisonConditions("records.oasf_created_at", cfg.CreatedAts)
		if condition != "" {
			query = query.Where(condition, args...)
		}
	}

	// Handle author filters with wildcard support (searching in JSON array).
	if len(cfg.Authors) > 0 {
		// Build OR conditions for each author pattern against the JSON string
		var authorConditions []string

		var authorArgs []any

		for _, author := range cfg.Authors {
			condition, arg := utils.BuildSingleWildcardCondition("records.authors", "*"+author+"*")
			authorConditions = append(authorConditions, condition)
			authorArgs = append(authorArgs, arg)
		}

		if len(authorConditions) > 0 {
			query = query.Where(strings.Join(authorConditions, " OR "), authorArgs...)
		}
	}

	// Handle schema version filter with comparison operator support.
	if len(cfg.SchemaVersions) > 0 {
		condition, args := utils.BuildComparisonConditions("records.schema_version", cfg.SchemaVersions)
		if condition != "" {
			query = query.Where(condition, args...)
		}
	}

	// Handle module ID filters.
	if len(cfg.ModuleIDs) > 0 {
		// Check if modules join already exists
		if len(cfg.ModuleNames) == 0 {
			query = query.Joins("JOIN modules ON modules.record_cid = records.record_cid")
		}

		query = query.Where("modules.module_id IN ?", cfg.ModuleIDs)
	}

	// Handle verified filter.
	if cfg.Verified != nil {
		if *cfg.Verified {
			// Filter for verified records only
			query = query.Joins("JOIN name_verifications ON name_verifications.record_cid = records.record_cid").
				Where("name_verifications.status = ?", VerificationStatusVerified)
		} else {
			// Filter for non-verified records (either no verification or failed)
			query = query.Joins("LEFT JOIN name_verifications ON name_verifications.record_cid = records.record_cid").
				Where("name_verifications.status IS NULL OR name_verifications.status != ?", VerificationStatusVerified)
		}
	}

	// Handle trusted filter (signature verification passed; derived from signature_verifications).
	if cfg.Trusted != nil {
		const verifiedStatus = "verified"
		if *cfg.Trusted {
			query = query.Where("EXISTS (SELECT 1 FROM signature_verifications sv WHERE sv.record_cid = records.record_cid AND sv.status = ?)", verifiedStatus)
		} else {
			query = query.Where("NOT EXISTS (SELECT 1 FROM signature_verifications sv WHERE sv.record_cid = records.record_cid AND sv.status = ?)", verifiedStatus)
		}
	}

	return query
}

// SetRecordSigned marks a record as signed.
// This is called when a signature is attached to a record.
func (d *DB) SetRecordSigned(recordCID string) error {
	result := d.gormDB.Model(&Record{}).
		Where("record_cid = ?", recordCID).
		Update("signed", true)

	if result.Error != nil {
		return fmt.Errorf("failed to set record as signed: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("record not found: %s", recordCID)
	}

	logger.Debug("Marked record as signed", "record_cid", recordCID)

	return nil
}
