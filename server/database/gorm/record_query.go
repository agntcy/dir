// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"fmt"

	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/conv"
	"gorm.io/gorm"
)

// orderFunc appends the caller's ORDER BY clauses (and any JOINs they need)
// to a record query. Record and catalog listings order differently, so the
// shared paging helper takes the ordering as a parameter.
type orderFunc func(*gorm.DB, *types.RecordFilters) (*gorm.DB, error)

// recordQuery returns a records query with the filters in cfg applied,
// including the policy gate. Limit, offset and order are left to the caller.
func (d *DB) recordQuery(cfg *types.RecordFilters) *gorm.DB {
	return d.handleFilterOptions(d.gormDB.Model(&Record{}), cfg)
}

// pageRecords returns one page of the distinct records matched by query,
// ordered by order, using peek-ahead pagination (Limit+1) to report hasMore.
// The named associations are eager-loaded; associations that need conditions
// can be preloaded on query before it is passed in. cfg.Limit must be > 0.
func (d *DB) pageRecords(
	query *gorm.DB,
	cfg *types.RecordFilters,
	order orderFunc,
	preloads ...string,
) ([]Record, bool, error) {
	// PostgreSQL requires every ORDER BY expression to appear in the SELECT
	// list when using SELECT DISTINCT. records.* covers the record columns;
	// the usage-metric expressions are added when the order references them.
	// Since record_cid is the primary key, DISTINCT still de-duplicates the
	// rows introduced by filter JOINs at the record level.
	selectCols := "records.*"
	if orderNeedsUsageMetrics(cfg) {
		selectCols += ", COALESCE(rum.pull_count, 0) + COALESCE(rum.lookup_count, 0)"
		selectCols += ", COALESCE(rum.provider_count, 0)"
	}

	query = query.Select(selectCols).Distinct()

	for _, preload := range preloads {
		query = query.Preload(preload)
	}

	query, err := order(query, cfg)
	if err != nil {
		return nil, false, err
	}

	if cfg.Offset > 0 {
		query = query.Offset(cfg.Offset)
	}

	query = query.Limit(cfg.Limit + 1)

	var records []Record
	if err := query.Find(&records).Error; err != nil {
		return nil, false, fmt.Errorf("query records: %w", err)
	}

	hasMore := len(records) > cfg.Limit
	if hasMore {
		records = records[:cfg.Limit]
	}

	return records, hasMore, nil
}

// countRecords returns the number of distinct records matched by query.
func countRecords(query *gorm.DB) (uint32, error) {
	var count int64
	if err := query.Distinct("records.record_cid").Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count records: %w", err)
	}

	total, err := conv.Uint32FromInt64(count)
	if err != nil {
		return 0, fmt.Errorf("count records: %w", err)
	}

	return total, nil
}

// orderNeedsUsageMetrics reports whether cfg orders by a usage-metrics column,
// which makes the order JOIN record_usage_metrics.
func orderNeedsUsageMetrics(cfg *types.RecordFilters) bool {
	for _, o := range cfg.OrderBy {
		if usageMetricsColumns[o.Column] {
			return true
		}
	}

	return false
}
