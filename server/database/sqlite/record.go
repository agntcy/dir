// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sqlite

import (
	"errors"
	"fmt"
	"strings"
	"time"

	searchv1 "github.com/agntcy/dir/api/search/v1"
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

// GetRecords retrieves records with full details based on query expression.
func (d *DB) GetRecords(expr *types.QueryExpression, limit, offset int) ([]types.Record, error) {
	// Start with the base query for records.
	query := d.gormDB.Model(&Record{}).Distinct()

	// Apply pagination.
	if limit > 0 {
		query = query.Limit(limit)
	}

	if offset > 0 {
		query = query.Offset(offset)
	}

	// Apply expression-based filters
	joined := &joinedTables{}
	query = d.buildExpressionQuery(query, expr, joined)

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

// GetRecordCIDs retrieves record CIDs based on query expression.
func (d *DB) GetRecordCIDs(expr *types.QueryExpression, limit, offset int) ([]string, error) {
	// Start with the base query for records - only select CID for efficiency.
	query := d.gormDB.Model(&Record{}).Select("records.record_cid").Distinct()

	// Apply pagination.
	if limit > 0 {
		query = query.Limit(limit)
	}

	if offset > 0 {
		query = query.Offset(offset)
	}

	// Apply expression-based filters
	joined := &joinedTables{}
	query = d.buildExpressionQuery(query, expr, joined)

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

// joinedTables tracks which tables have been joined to avoid duplicate joins.
type joinedTables struct {
	skills   bool
	locators bool
	modules  bool
	domains  bool
}

// buildExpressionQuery recursively builds SQL WHERE clauses from an expression tree.
//
//nolint:gocognit,cyclop
func (d *DB) buildExpressionQuery(query *gorm.DB, expr *types.QueryExpression, joined *joinedTables) *gorm.DB {
	if expr == nil {
		return query
	}

	// Check which field is set (only one should be non-nil)
	if expr.Query != nil {
		// Leaf node - process single query
		return d.buildSingleQuery(query, expr.Query, joined)
	}

	if expr.And != nil {
		// AND expression - all sub-expressions must match
		return d.buildAndExpression(query, expr.And, joined)
	}

	if expr.Or != nil {
		// OR expression - at least one sub-expression must match
		return d.buildOrExpression(query, expr.Or, joined)
	}

	if expr.Not != nil {
		// NOT expression - negate the sub-expression
		return d.buildNotExpression(query, expr.Not, joined)
	}

	logger.Warn("Empty expression - no fields set")

	return query
}

// buildSingleQuery processes a single RecordQuery and applies it to the GORM query.
//
//nolint:gocognit,cyclop
func (d *DB) buildSingleQuery(query *gorm.DB, q *searchv1.RecordQuery, joined *joinedTables) *gorm.DB {
	switch q.GetType() {
	case searchv1.RecordQueryType_RECORD_QUERY_TYPE_UNSPECIFIED:
		logger.Warn("Unspecified query type", "query", q)

		return query

	case searchv1.RecordQueryType_RECORD_QUERY_TYPE_NAME:
		condition, arg := utils.BuildSingleWildcardCondition("records.name", q.GetValue())

		return query.Where(condition, arg)

	case searchv1.RecordQueryType_RECORD_QUERY_TYPE_VERSION:
		condition, arg := utils.BuildSingleWildcardCondition("records.version", q.GetValue())

		return query.Where(condition, arg)

	case searchv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL_ID:
		if !joined.skills {
			query = query.Joins("JOIN skills ON skills.record_cid = records.record_cid")
			joined.skills = true
		}

		// Parse skill ID
		var skillID uint64
		if _, err := fmt.Sscanf(q.GetValue(), "%d", &skillID); err != nil {
			logger.Warn("Failed to parse skill ID", "value", q.GetValue(), "error", err)

			return query
		}

		return query.Where("skills.skill_id = ?", skillID)

	case searchv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL_NAME:
		if !joined.skills {
			query = query.Joins("JOIN skills ON skills.record_cid = records.record_cid")
			joined.skills = true
		}

		condition, arg := utils.BuildSingleWildcardCondition("skills.name", q.GetValue())

		return query.Where(condition, arg)

	case searchv1.RecordQueryType_RECORD_QUERY_TYPE_LOCATOR:
		if !joined.locators {
			query = query.Joins("JOIN locators ON locators.record_cid = records.record_cid")
			joined.locators = true
		}
		// Parse locator (type:url format)
		return d.buildLocatorQuery(query, q.GetValue())

	case searchv1.RecordQueryType_RECORD_QUERY_TYPE_MODULE:
		if !joined.modules {
			query = query.Joins("JOIN modules ON modules.record_cid = records.record_cid")
			joined.modules = true
		}

		condition, arg := utils.BuildSingleWildcardCondition("modules.name", q.GetValue())

		return query.Where(condition, arg)

	case searchv1.RecordQueryType_RECORD_QUERY_TYPE_DOMAIN_ID:
		if !joined.domains {
			query = query.Joins("JOIN domains ON domains.record_cid = records.record_cid")
			joined.domains = true
		}

		// Parse domain ID
		var domainID uint64
		if _, err := fmt.Sscanf(q.GetValue(), "%d", &domainID); err != nil {
			logger.Warn("Failed to parse domain ID", "value", q.GetValue(), "error", err)

			return query
		}

		return query.Where("domains.domain_id = ?", domainID)

	case searchv1.RecordQueryType_RECORD_QUERY_TYPE_DOMAIN_NAME:
		if !joined.domains {
			query = query.Joins("JOIN domains ON domains.record_cid = records.record_cid")
			joined.domains = true
		}

		condition, arg := utils.BuildSingleWildcardCondition("domains.name", q.GetValue())

		return query.Where(condition, arg)

	default:
		logger.Warn("Unknown query type", "type", q.GetType())

		return query
	}
}

// buildLocatorQuery handles locator queries with type:url format.
func (d *DB) buildLocatorQuery(query *gorm.DB, value string) *gorm.DB {
	value = strings.TrimSpace(value)
	if value == "" {
		return query
	}

	// Split on first colon to separate type from URL
	l := strings.SplitN(value, ":", 2) //nolint:mnd

	// Case 1: Single part (no colon at all)
	if len(l) == 1 {
		// If starts with *, treat as URL pattern
		// Example: "*marketing-strategy"
		if strings.HasPrefix(l[0], "*") {
			condition, arg := utils.BuildSingleWildcardCondition("locators.url", l[0])

			return query.Where(condition, arg)
		}

		// Otherwise treat as type
		// Example: "docker_image"
		if strings.TrimSpace(l[0]) != "" {
			condition, arg := utils.BuildSingleWildcardCondition("locators.type", l[0])

			return query.Where(condition, arg)
		}

		return query
	}

	// Case 2: Two parts after split (contains at least one colon)
	if len(l) == 2 { //nolint:mnd
		// If the second part starts with // and first part is *, it's a wildcard URL pattern
		// Example: "*://ghcr.io/agntcy/marketing-strategy" -> pure URL pattern
		if strings.HasPrefix(l[1], "//") && strings.HasPrefix(l[0], "*") {
			condition, arg := utils.BuildSingleWildcardCondition("locators.url", value)

			return query.Where(condition, arg)
		}

		// If the second part starts with // and first part is NOT *, it's a standalone URL
		// Example: "http://localhost:8081" -> splits to ["http", "//localhost:8081"]
		// Search for the FULL URL, not type+url separately
		if strings.HasPrefix(l[1], "//") {
			condition, arg := utils.BuildSingleWildcardCondition("locators.url", value)

			return query.Where(condition, arg)
		}

		// Otherwise it's standard type:url format
		// Example: "docker_image:https://..." -> type AND url filters
		if strings.TrimSpace(l[0]) != "" {
			condition, arg := utils.BuildSingleWildcardCondition("locators.type", l[0])
			query = query.Where(condition, arg)
		}

		if strings.TrimSpace(l[1]) != "" {
			condition, arg := utils.BuildSingleWildcardCondition("locators.url", l[1])
			query = query.Where(condition, arg)
		}
	}

	return query
}

// buildAndExpression processes an AND expression.
func (d *DB) buildAndExpression(query *gorm.DB, and *types.AndExpression, joined *joinedTables) *gorm.DB {
	for _, subExpr := range and.Expressions {
		query = d.buildExpressionQuery(query, subExpr, joined)
	}

	return query
}

// buildOrExpression processes an OR expression by building a subquery.
//
//nolint:gocognit
func (d *DB) buildOrExpression(query *gorm.DB, or *types.OrExpression, joined *joinedTables) *gorm.DB {
	if len(or.Expressions) == 0 {
		return query
	}

	// Build OR conditions as a single WHERE clause with OR operators
	// We need to collect all the conditions and arguments
	var conditions []string

	var args []interface{}

	for _, subExpr := range or.Expressions {
		// Create a temporary query to extract the WHERE condition
		tempQuery := d.gormDB.Model(&Record{}).Where("1=1")
		tempJoined := &joinedTables{}
		tempQuery = d.buildExpressionQuery(tempQuery, subExpr, tempJoined)

		// Apply any necessary joins to the main query
		if tempJoined.skills && !joined.skills {
			query = query.Joins("JOIN skills ON skills.record_cid = records.record_cid")
			joined.skills = true
		}

		if tempJoined.locators && !joined.locators {
			query = query.Joins("JOIN locators ON locators.record_cid = records.record_cid")
			joined.locators = true
		}

		if tempJoined.modules && !joined.modules {
			query = query.Joins("JOIN modules ON modules.record_cid = records.record_cid")
			joined.modules = true
		}

		if tempJoined.domains && !joined.domains {
			query = query.Joins("JOIN domains ON domains.record_cid = records.record_cid")
			joined.domains = true
		}

		// Extract the SQL from the temp query
		sql := tempQuery.ToSQL(func(tx *gorm.DB) *gorm.DB {
			return tx.Find(&[]Record{})
		})

		// Parse out the WHERE clause (this is a simplified approach)
		// In production, we'd want more robust SQL parsing
		if idx := strings.Index(sql, "WHERE"); idx != -1 {
			whereClause := sql[idx+6:]
			// Remove the trailing parts (ORDER BY, LIMIT, etc.)
			if endIdx := strings.Index(whereClause, "ORDER"); endIdx != -1 {
				whereClause = whereClause[:endIdx]
			}

			conditions = append(conditions, strings.TrimSpace(whereClause))
		}
	}

	// Combine conditions with OR
	if len(conditions) > 0 {
		orCondition := "(" + strings.Join(conditions, " OR ") + ")"
		query = query.Where(orCondition, args...)
	}

	return query
}

// buildNotExpression processes a NOT expression.
func (d *DB) buildNotExpression(query *gorm.DB, not *types.NotExpression, joined *joinedTables) *gorm.DB {
	if not.Expression == nil {
		return query
	}

	// Build a subquery for the NOT condition
	tempQuery := d.gormDB.Model(&Record{}).Where("1=1")
	tempJoined := &joinedTables{}
	tempQuery = d.buildExpressionQuery(tempQuery, not.Expression, tempJoined)

	// Apply any necessary joins to the main query
	if tempJoined.skills && !joined.skills {
		query = query.Joins("JOIN skills ON skills.record_cid = records.record_cid")
		joined.skills = true
	}

	if tempJoined.locators && !joined.locators {
		query = query.Joins("JOIN locators ON locators.record_cid = records.record_cid")
		joined.locators = true
	}

	if tempJoined.modules && !joined.modules {
		query = query.Joins("JOIN modules ON modules.record_cid = records.record_cid")
		joined.modules = true
	}

	if tempJoined.domains && !joined.domains {
		query = query.Joins("JOIN domains ON domains.record_cid = records.record_cid")
		joined.domains = true
	}

	// Extract the SQL from the temp query
	sql := tempQuery.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Find(&[]Record{})
	})

	// Parse out the WHERE clause
	if idx := strings.Index(sql, "WHERE"); idx != -1 {
		whereClause := sql[idx+6:]
		// Remove the trailing parts
		if endIdx := strings.Index(whereClause, "ORDER"); endIdx != -1 {
			whereClause = whereClause[:endIdx]
		}

		notCondition := "NOT (" + strings.TrimSpace(whereClause) + ")"
		query = query.Where(notCondition)
	}

	return query
}
