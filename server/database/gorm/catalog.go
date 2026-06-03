// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"fmt"

	catalogv1 "github.com/agntcy/dir/api/catalog/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/oasf-sdk/pkg/translator"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
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

// GetCatalogEntries returns the AI Catalog entries matching the given record
// filters, using peek-ahead pagination (Limit+1) to report hasMore. Records
// with no AI Catalog projection are skipped rather than failing the page.
func (d *DB) GetCatalogEntries(opts ...types.FilterOption) ([]*catalogv1.CatalogEntry, bool, error) {
	cfg := &types.RecordFilters{}

	for _, opt := range opts {
		if opt == nil {
			return nil, false, fmt.Errorf("nil filter option provided")
		}

		opt(cfg)
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
		Limit(pageSize + 1)

	if cfg.Offset > 0 {
		query = query.Offset(cfg.Offset)
	}

	query = d.handleFilterOptions(query, cfg)

	query, err := applyCatalogOrder(query, cfg)
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
		entry, err := recordToCatalogEntry(&records[i])
		if err != nil {
			// Expected for records without a known catalog module.
			logger.Debug("skipping record without catalog projection", "record_cid", records[i].RecordCID, "error", err)

			continue
		}

		entries = append(entries, entry)
	}

	return entries, hasMore, nil
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

		direction := "ASC"
		if o.Desc {
			direction = "DESC"
		}

		query = query.Order(fmt.Sprintf("%s %s", column, direction))
	}

	return query.Order("records.record_cid ASC"), nil
}

// recordToCatalogEntry reconstructs the OASF record from its associations and
// delegates the AI Catalog projection to the oasf-sdk translator.
func recordToCatalogEntry(r *Record) (*catalogv1.CatalogEntry, error) {
	record, err := recordToStruct(r)
	if err != nil {
		return nil, fmt.Errorf("build record struct: %w", err)
	}

	catalog, err := translator.RecordToCatalog(record, translator.WithCatalogCID(r.RecordCID))
	if err != nil {
		return nil, fmt.Errorf("project record to catalog: %w", err)
	}

	return catalogEntryFromStruct(catalog)
}

// recordToStruct rebuilds the subset of the OASF record the projection reads:
// identity fields plus the Modules/Skills/Domains/Annotations associations.
func recordToStruct(r *Record) (*structpb.Struct, error) {
	modules := make([]any, 0, len(r.Modules))

	for _, m := range r.Modules {
		module := map[string]any{"name": m.Name}
		if len(m.Data) > 0 {
			module["data"] = m.Data
		}

		modules = append(modules, module)
	}

	skills := make([]any, 0, len(r.Skills))
	for _, s := range r.Skills {
		skills = append(skills, map[string]any{"name": s.Name})
	}

	domains := make([]any, 0, len(r.Domains))
	for _, d := range r.Domains {
		domains = append(domains, map[string]any{"name": d.Name})
	}

	annotations := make(map[string]any, len(r.Annotations))
	for _, a := range r.Annotations {
		annotations[a.Key] = a.Value
	}

	fields := map[string]any{
		"name":           r.Name,
		"version":        r.Version,
		"description":    r.Description,
		"schema_version": r.SchemaVersion,
		"created_at":     r.OASFCreatedAt,
		"modules":        modules,
		"skills":         skills,
		"domains":        domains,
		"annotations":    annotations,
	}

	return structpb.NewStruct(fields) //nolint:wrapcheck
}

// catalogEntryFromStruct converts the translator's structpb output into the
// typed CatalogEntry message.
func catalogEntryFromStruct(s *structpb.Struct) (*catalogv1.CatalogEntry, error) {
	raw, err := protojson.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("marshal catalog entry: %w", err)
	}

	var entry catalogv1.CatalogEntry
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(raw, &entry); err != nil {
		return nil, fmt.Errorf("decode catalog entry: %w", err)
	}

	return &entry, nil
}
