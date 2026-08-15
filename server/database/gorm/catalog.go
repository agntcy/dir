// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"fmt"
	"math"
	"sort"
	"strings"

	catalogv1 "github.com/agntcy/dir/api/catalog/v1"
	"github.com/agntcy/dir/server/database/utils"
	"github.com/agntcy/dir/server/types"
	"gorm.io/gorm"
)

// defaultCatalogPageSize is applied when the caller does not set a limit.
const defaultCatalogPageSize = 20

const emptyJSONObject = "{}"

// catalogSortColumns allow-lists the columns a controller may sort by,
// mapping the logical name to the qualified column so caller-supplied sort
// keys are never interpolated into the query verbatim.
var catalogSortColumns = map[string]string{
	"created_at":     "records.oasf_created_at",
	"name":           "records.name",
	"version":        "records.version",
	"schema_version": "records.schema_version",
	"record_cid":     "records.record_cid",
}

// CountCatalogEntries returns the number of distinct records matching the
// given filters. Limit and offset are ignored.
func (d *DB) CountCatalogEntries(opts ...types.CatalogQueryOption) (uint32, error) {
	cfg, err := types.CatalogFiltersFromOptions(opts...)
	if err != nil {
		return 0, err //nolint:wrapcheck
	}

	recordCfg := cfg.RecordFilters
	recordCfg.Limit = 0
	recordCfg.Offset = 0

	query := d.gormDB.Model(&Record{})
	query = d.handleFilterOptions(query, &recordCfg)
	query = applyKnownCatalogModuleFilter(query)
	query = applyMediaTypeFilter(query, cfg.MediaTypeFilters)
	query = applyTagFilter(query, cfg.TagFilters)
	query = query.Distinct("records.record_cid")

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count catalog records: %w", err)
	}

	if count < 0 || math.MaxUint32 < count {
		return 0, fmt.Errorf("can't convert %d to uint32", count)
	}

	return uint32(count), nil
}

type annotationRow struct {
	Key   string
	Value string
}

// ListCatalogTags returns distinct catalog tags derived from OASF skills,
// domains, and record annotations, sorted lexicographically by label.
func (d *DB) ListCatalogTags() ([]*catalogv1.CatalogTag, error) {
	var tags []*catalogv1.CatalogTag

	var skillNames []string
	if err := d.gormDB.
		Model(&Skill{}).
		Distinct().
		Pluck("name", &skillNames).Error; err != nil {
		return nil, fmt.Errorf("list skill tags: %w", err)
	}

	for _, skillName := range skillNames {
		tags = append(tags, &catalogv1.CatalogTag{
			Id:    catalogv1.SkillTag("*", skillName),
			Label: catalogv1.TagLabel(skillName),
		})
	}

	var domainNames []string
	if err := d.gormDB.
		Model(&Domain{}).
		Distinct().
		Pluck("name", &domainNames).Error; err != nil {
		return nil, fmt.Errorf("list domain tags: %w", err)
	}

	for _, domainName := range domainNames {
		tags = append(tags, &catalogv1.CatalogTag{
			Id:    catalogv1.DomainTag("*", domainName),
			Label: catalogv1.TagLabel(domainName),
		})
	}

	var annotationRows []annotationRow
	if err := d.gormDB.
		Table("annotations").
		Select("annotations.key, annotations.value").
		Distinct().
		Scan(&annotationRows).Error; err != nil {
		return nil, fmt.Errorf("list annotation tags: %w", err)
	}

	for _, row := range annotationRows {
		tags = append(tags, &catalogv1.CatalogTag{
			Id:    catalogv1.AnnotationTag(row.Key, row.Value),
			Label: catalogv1.AnnotationLabel(row.Key, row.Value),
		})
	}

	sort.Slice(tags, func(i, j int) bool {
		return tags[i].GetLabel() < tags[j].GetLabel()
	})

	return tags, nil
}

// GetCatalogEntries returns the AI Catalog entries matching the given filters,
// using peek-ahead pagination (Limit+1) to report hasMore. Records with no AI
// Catalog projection are skipped rather than failing the page.
func (d *DB) GetCatalogEntries(opts ...types.CatalogQueryOption) ([]*catalogv1.CatalogEntry, bool, error) {
	cfg, err := types.CatalogFiltersFromOptions(opts...)
	if err != nil {
		return nil, false, err //nolint:wrapcheck
	}

	recordCfg := cfg.RecordFilters

	pageSize := recordCfg.Limit
	if pageSize <= 0 {
		pageSize = defaultCatalogPageSize
	}

	// Eager-load the associations the projection walks; DISTINCT drops the
	// duplicate rows introduced by the filter JOINs.
	query := d.gormDB.
		Model(&Record{}).
		Select("records.*").
		Distinct().
		Preload("Modules").
		Preload("Skills").
		Preload("Domains").
		Preload("Annotations").
		Preload("Signatures").
		Preload("NameVerification").
		Preload("ScanReports").
		Limit(pageSize + 1)

	if recordCfg.Offset > 0 {
		query = query.Offset(recordCfg.Offset)
	}

	query = d.handleFilterOptions(query, &recordCfg)
	query = applyKnownCatalogModuleFilter(query)
	query = applyMediaTypeFilter(query, cfg.MediaTypeFilters)
	query = applyTagFilter(query, cfg.TagFilters)

	query, err = applyCatalogOrder(query, &recordCfg)
	if err != nil {
		return nil, false, err //nolint:wrapcheck
	}

	var records []Record
	if err := query.Find(&records).Error; err != nil {
		return nil, false, fmt.Errorf("query catalog records: %w", err)
	}

	hasMore := len(records) > pageSize
	if hasMore {
		records = records[:pageSize]
	}

	entries := make([]*catalogv1.CatalogEntry, 0, len(records))

	for i := range records {
		opts := []catalogv1.ConvertOption{
			catalogv1.WithSignatures(convertSignatures(records[i].Signatures)),
			catalogv1.WithScanReports(convertScanReports(records[i].ScanReports)),
			catalogv1.WithTrustStatus(deriveTrustStatus(&records[i])),
		}

		if metrics, err := d.GetUsageMetrics(records[i].RecordCID); err == nil {
			opts = append(opts, catalogv1.WithUsageMetrics(metrics))
		} else {
			logger.Debug("skipping usage metrics for record", "record_cid", records[i].RecordCID, "error", err)
		}

		entry, err := catalogv1.RecordToCatalog(&records[i], opts...)
		if err != nil {
			// Expected for records without a known catalog module.
			logger.Debug("skipping record without catalog projection", "record_cid", records[i].RecordCID, "error", err)

			continue
		}

		entries = append(entries, entry)
	}

	return entries, hasMore, nil
}

func applyKnownCatalogModuleFilter(query *gorm.DB) *gorm.DB {
	moduleNames := catalogv1.KnownCatalogModuleNames()

	return query.Where(`
EXISTS (
	SELECT 1 FROM modules catalog_modules
	WHERE catalog_modules.record_cid = records.record_cid
	AND catalog_modules.name IN ?
	AND catalog_modules.data IS NOT NULL
	AND catalog_modules.data != ?
)`, moduleNames, emptyJSONObject)
}

// applyMediaTypeFilter restricts the query to records matching one or more
// catalog media types. Agent Skills markdown vs gzip is determined by the
// indexed modules.artifact_media_type column.
func applyMediaTypeFilter(query *gorm.DB, filters []types.MediaTypeFilter) *gorm.DB {
	if len(filters) == 0 {
		return query
	}

	conditions := make([]string, 0, len(filters))
	args := make([]any, 0, len(filters))

	for _, filter := range filters {
		condition, filterArgs, err := mediaTypeExistsCondition(filter)
		if err != nil {
			logger.Error("unsupported media type filter", "error", err)

			return query.Where("1 = 0")
		}

		if condition == "" {
			continue
		}

		conditions = append(conditions, condition)
		args = append(args, filterArgs...)
	}

	if len(conditions) == 0 {
		return query
	}

	condition := strings.Join(conditions, " OR ")
	if len(conditions) > 1 {
		condition = "(" + condition + ")"
	}

	return query.Where(condition, args...)
}

func mediaTypeExistsCondition(filter types.MediaTypeFilter) (string, []any, error) {
	if filter.ModuleName == "" {
		return "", nil, fmt.Errorf("media type filter missing module name")
	}

	args := []any{filter.ModuleName, emptyJSONObject}

	clause := `
EXISTS (
	SELECT 1 FROM modules media_type_modules
	WHERE media_type_modules.record_cid = records.record_cid
	AND media_type_modules.name = ?
	AND media_type_modules.data IS NOT NULL
	AND media_type_modules.data != ?`

	if filter.ArtifactMediaType != "" {
		clause += `
	AND media_type_modules.artifact_media_type = ?`

		args = append(args, filter.ArtifactMediaType)
	}

	clause += `
)`

	return clause, args, nil
}

func applyTagFilter(query *gorm.DB, filters []types.TagFilter) *gorm.DB {
	if len(filters) == 0 {
		return query
	}

	conditions := make([]string, 0, len(filters))
	args := make([]any, 0, len(filters))

	for _, filter := range filters {
		condition, filterArgs, err := tagExistsCondition(filter)
		if err != nil {
			logger.Error("unsupported tag filter", "error", err)

			return query.Where("1 = 0")
		}

		if condition == "" {
			continue
		}

		conditions = append(conditions, condition)
		args = append(args, filterArgs...)
	}

	if len(conditions) == 0 {
		return query
	}

	condition := strings.Join(conditions, " OR ")
	if len(conditions) > 1 {
		condition = "(" + condition + ")"
	}

	return query.Where(condition, args...)
}

func tagExistsCondition(filter types.TagFilter) (string, []any, error) {
	switch {
	case filter.SkillName != "":
		condition, arg := utils.BuildSingleWildcardCondition("tag_skills.name", filter.SkillName)

		return fmt.Sprintf(`
EXISTS (
	SELECT 1 FROM skills tag_skills
	WHERE tag_skills.record_cid = records.record_cid
	AND %s
)`, condition), []any{arg}, nil
	case filter.DomainName != "":
		condition, arg := utils.BuildSingleWildcardCondition("tag_domains.name", filter.DomainName)

		return fmt.Sprintf(`
EXISTS (
	SELECT 1 FROM domains tag_domains
	WHERE tag_domains.record_cid = records.record_cid
	AND %s
)`, condition), []any{arg}, nil
	case filter.Annotation != nil:
		return `
EXISTS (
	SELECT 1 FROM annotations tag_annotations
	WHERE tag_annotations.record_cid = records.record_cid
	AND tag_annotations.key = ?
	AND tag_annotations.value = ?
)`, []any{filter.Annotation.Key, filter.Annotation.Value}, nil
	case filter.AnnotationKey != "":
		condition, arg := utils.BuildSingleWildcardCondition("tag_annotations.key", filter.AnnotationKey)

		return fmt.Sprintf(`
EXISTS (
	SELECT 1 FROM annotations tag_annotations
	WHERE tag_annotations.record_cid = records.record_cid
	AND %s
)`, condition), []any{arg}, nil
	default:
		return "", nil, fmt.Errorf("tag filter has no match criteria")
	}
}

func convertScanReports(reports []ScanReport) []catalogv1.ScanReportSummary {
	result := make([]catalogv1.ScanReportSummary, len(reports))
	for i := range reports {
		result[i] = &reports[i]
	}

	return result
}

func deriveTrustStatus(record *Record) catalogv1.TrustStatus {
	signatureStatuses := make([]string, len(record.Signatures))
	for i := range record.Signatures {
		signatureStatuses[i] = record.Signatures[i].Status
	}

	nameVerificationStatus := ""
	if record.NameVerification != nil {
		nameVerificationStatus = record.NameVerification.Status
	}

	return catalogv1.DeriveTrustStatus(signatureStatuses, nameVerificationStatus)
}

// applyCatalogOrder appends the allow-listed ORDER BY clauses plus a
// primary-key tiebreaker for stable paging, defaulting to newest-first.
func applyCatalogOrder(query *gorm.DB, cfg *types.RecordFilters) (*gorm.DB, error) {
	if len(cfg.OrderBy) == 0 {
		query = query.Order("records.oasf_created_at DESC")
	}

	for _, o := range cfg.OrderBy {
		column, ok := catalogSortColumns[o.Column]
		if !ok {
			return nil, fmt.Errorf("unsupported sort column %q", o.Column)
		}

		direction := sortASC
		if o.Desc {
			direction = sortDESC
		}

		query = query.Order(fmt.Sprintf("%s %s", column, direction))
	}

	return query.Order("records.record_cid ASC"), nil
}
