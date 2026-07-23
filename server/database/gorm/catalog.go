// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"fmt"
	"math"

	catalogv1 "github.com/agntcy/dir/api/catalog/v1"
	"github.com/agntcy/dir/server/types"
	"gorm.io/gorm"
)

// defaultCatalogPageSize is applied when the caller does not set a limit.
const defaultCatalogPageSize = 20

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

func parseOpts(opts ...types.FilterOption) (*types.RecordFilters, error) {
	cfg := &types.RecordFilters{}

	for _, opt := range opts {
		if opt == nil {
			return nil, fmt.Errorf("nil filter option provided")
		}

		opt(cfg)
	}

	return cfg, nil
}

// CountCatalogEntries returns the number of distinct records matching the
// given filters. Limit and offset options are ignored.
func (d *DB) CountCatalogEntries(opts ...types.FilterOption) (uint32, error) {
	cfg, err := parseOpts(opts...)
	if err != nil {
		return 0, err
	}

	cfg.Limit = 0
	cfg.Offset = 0

	query := d.gormDB.Model(&Record{})
	query = d.handleFilterOptions(query, cfg)
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

// GetCatalogEntries returns the AI Catalog entries matching the given record
// filters, using peek-ahead pagination (Limit+1) to report hasMore. Records
// with no AI Catalog projection are skipped rather than failing the page.
func (d *DB) GetCatalogEntries(opts ...types.FilterOption) ([]*catalogv1.CatalogEntry, bool, error) {
	cfg, err := parseOpts(opts...)
	if err != nil {
		return nil, false, err
	}

	pageSize := cfg.Limit
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
		Preload("ScanReports").
		Limit(pageSize + 1)

	if cfg.Offset > 0 {
		query = query.Offset(cfg.Offset)
	}

	query = d.handleFilterOptions(query, cfg)

	query, err = applyCatalogOrder(query, cfg)
	if err != nil {
		return nil, false, err
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

func convertScanReports(reports []ScanReport) []catalogv1.ScanReportSummary {
	result := make([]catalogv1.ScanReportSummary, len(reports))
	for i := range reports {
		result[i] = &reports[i]
	}

	return result
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
