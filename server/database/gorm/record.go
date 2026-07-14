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

// recordSortColumns allow-lists the columns GetRecords may sort by, mapping
// the logical name to the SQL expression so caller-supplied keys are never
// interpolated into the query verbatim.
var recordSortColumns = map[string]string{
	"created_at":       "records.created_at",
	"name":             "records.name",
	"version":          "records.version",
	"schema_version":   "records.schema_version",
	"record_cid":       "records.record_cid",
	"popularity_score": "COALESCE(rum.pull_count, 0) + COALESCE(rum.lookup_count, 0)",
	"provider_count":   "COALESCE(rum.provider_count, 0)",
}

const (
	sortASC  = "ASC"
	sortDESC = "DESC"
)

// usageMetricsColumns is the subset of recordSortColumns that require a LEFT
// JOIN on record_usage_metrics.
var usageMetricsColumns = map[string]bool{
	"popularity_score": true,
	"provider_count":   true,
}

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
	ScanReports []ScanReport            `gorm:"foreignKey:RecordCID;references:RecordCID"`
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

	var err error

	query, err = applyRecordOrder(query, cfg)
	if err != nil {
		return nil, err
	}

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

// cidRecord is a minimal scan target for GetRecordCIDs.
type cidRecord struct {
	RecordCID string `gorm:"column:record_cid"`
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

	// Check whether usage-metrics columns are needed for ORDER BY before building SELECT.
	needsUsageMetrics := false

	for _, o := range cfg.OrderBy {
		if usageMetricsColumns[o.Column] {
			needsUsageMetrics = true

			break
		}
	}

	// PostgreSQL requires every ORDER BY expression to appear in the SELECT list when
	// using SELECT DISTINCT. Include all columns that applyRecordOrder may reference.
	// Since record_cid is the primary key, all records columns are functionally
	// dependent on it, so DISTINCT still de-duplicates at the record level.
	selectCols := "records.record_cid, records.created_at, records.name, records.version, records.schema_version"
	if needsUsageMetrics {
		selectCols += ", COALESCE(rum.pull_count, 0) + COALESCE(rum.lookup_count, 0)"
		selectCols += ", COALESCE(rum.provider_count, 0)"
	}

	query := d.gormDB.Model(&Record{}).Select(selectCols).Distinct()

	// Apply pagination.
	if cfg.Limit > 0 {
		query = query.Limit(cfg.Limit)
	}

	if cfg.Offset > 0 {
		query = query.Offset(cfg.Offset)
	}

	// Apply all filters.
	query = d.handleFilterOptions(query, cfg)

	var err error

	query, err = applyRecordOrder(query, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to apply order: %w", err)
	}

	// Execute the query to get only CIDs (no preloading needed).
	var results []cidRecord
	if err := query.Find(&results).Error; err != nil {
		return nil, fmt.Errorf("failed to query record CIDs: %w", err)
	}

	cids := make([]string, len(results))
	for i, r := range results {
		cids[i] = r.RecordCID
	}

	return cids, nil
}

// RemoveRecord removes a record from the search database by CID.
// Uses CASCADE DELETE to automatically remove related Skills, Locators, and Modules.
func (d *DB) RemoveRecord(cid string) error {
	// Remove signature verifications first
	if err := d.gormDB.Where("record_cid = ?", cid).Delete(&SignatureVerification{}).Error; err != nil {
		return fmt.Errorf("failed to remove signature verifications: %w", err)
	}

	// Remove usage metrics
	if err := d.gormDB.Where("record_cid = ?", cid).Delete(&RecordUsageMetrics{}).Error; err != nil {
		return fmt.Errorf("failed to remove usage metrics: %w", err)
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

// applyRecordOrder appends ORDER BY clauses to query. Automatically adds a
// LEFT JOIN on record_usage_metrics when a popularity column is requested.
// Defaults to newest-first when no OrderBy directives are provided.
// A stable secondary sort (created_at DESC, record_cid ASC) is always appended
// so pagination is deterministic regardless of the primary sort mode.
func applyRecordOrder(query *gorm.DB, cfg *types.RecordFilters) (*gorm.DB, error) {
	if len(cfg.OrderBy) == 0 {
		return query.Order("records.created_at DESC").Order("records.record_cid ASC"), nil
	}

	needsUsageJoin := false

	for _, o := range cfg.OrderBy {
		if usageMetricsColumns[o.Column] {
			needsUsageJoin = true

			break
		}
	}

	if needsUsageJoin {
		query = query.Joins("LEFT JOIN record_usage_metrics rum ON rum.record_cid = records.record_cid")
	}

	for _, o := range cfg.OrderBy {
		col, ok := recordSortColumns[o.Column]
		if !ok {
			return nil, fmt.Errorf("unsupported sort column %q", o.Column)
		}

		dir := sortASC
		if o.Desc {
			dir = sortDESC
		}

		query = query.Order(fmt.Sprintf("%s %s", col, dir))
	}

	return query.Order("records.created_at DESC").Order("records.record_cid ASC"), nil
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

	// Handle scan safe filter (is_safe column across all scanner types).
	if cfg.ScanSafe != nil {
		if *cfg.ScanSafe {
			// Safe: scanned AND no scanner flagged as unsafe.
			query = query.Where(
				"EXISTS (SELECT 1 FROM scan_reports sr WHERE sr.record_cid = records.record_cid) AND " +
					"NOT EXISTS (SELECT 1 FROM scan_reports sr WHERE sr.record_cid = records.record_cid AND sr.is_safe = false)",
			)
		} else {
			// Unsafe: at least one scanner reported is_safe = false.
			query = query.Where("EXISTS (SELECT 1 FROM scan_reports sr WHERE sr.record_cid = records.record_cid AND sr.is_safe = false)")
		}
	}

	// Handle description filters with wildcard support.
	if len(cfg.Descriptions) > 0 {
		condition, args := utils.BuildWildcardCondition("records.description", cfg.Descriptions)
		if condition != "" {
			query = query.Where(condition, args...)
		}
	}

	// Handle scan severity filter (max severity >= threshold across all scanner types).
	if len(cfg.ScanSeverities) > 0 {
		var severities []string
		for _, threshold := range cfg.ScanSeverities {
			severities = append(severities, scanSeveritiesGTE(threshold)...)
		}

		if len(severities) > 0 {
			query = query.Where("EXISTS (SELECT 1 FROM scan_reports sr WHERE sr.record_cid = records.record_cid AND sr.max_severity IN ?)", severities)
		}
	}

	return query
}

// scanSeveritiesGTE returns all severity strings that are >= the given threshold.
// Values are the short names stored in the max_severity column (e.g. "HIGH").
func scanSeveritiesGTE(threshold string) []string {
	order := []string{"NONE", "INFO", "LOW", "MEDIUM", "HIGH", "CRITICAL"}

	idx := -1

	for i, s := range order {
		if strings.EqualFold(s, threshold) {
			idx = i

			break
		}
	}

	if idx < 0 {
		return nil
	}

	return order[idx:]
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
