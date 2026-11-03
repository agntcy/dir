// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"errors"
	"sort"

	searchv1 "github.com/agntcy/dir/api/search/v1"
	"github.com/agntcy/dir/server/types"
)

// ListRecordSummaries returns one page of record summaries matching the
// provided options, and whether more records follow the page. It reads the
// database only. A limit of 0 or less defaults to
// types.DefaultRecordSummaryPageSize and is clamped to
// types.MaxRecordSummaryPageSize. List fields are sorted and never nil. The
// trust and scan fields agree with the TRUSTED, VERIFIED and SCAN_SAFE
// queries and with the AI catalog's badges.
func (d *DB) ListRecordSummaries(opts ...types.FilterOption) ([]*searchv1.RecordSummary, bool, error) {
	cfg := &types.RecordFilters{}

	for _, opt := range opts {
		if opt == nil {
			return nil, false, errors.New("nil option provided")
		}

		opt(cfg)
	}

	if cfg.Limit <= 0 {
		cfg.Limit = types.DefaultRecordSummaryPageSize
	}

	if cfg.Limit > types.MaxRecordSummaryPageSize {
		cfg.Limit = types.MaxRecordSummaryPageSize
	}

	// Preloaded like GetCatalogEntries: a failed scan row records why the scan
	// could not run, not a verdict, so it is left out.
	query := d.recordQuery(cfg).
		Preload("IdentityClaims", "role = ?", types.ClaimRoleOwner).
		Preload("ScanReports", "status IN ?", types.ScannedStatuses())

	records, hasMore, err := d.pageRecords(query, cfg, applyRecordOrder, "Skills", "Domains", "Modules", "Signatures")
	if err != nil {
		return nil, false, err
	}

	summaries := make([]*searchv1.RecordSummary, 0, len(records))

	for i := range records {
		r := &records[i]
		trust := deriveTrustStatus(r)

		summaries = append(summaries, &searchv1.RecordSummary{
			Cid:           r.RecordCID,
			Name:          r.Name,
			Version:       r.Version,
			SchemaVersion: r.SchemaVersion,
			Description:   r.Description,
			CreatedAt:     r.OASFCreatedAt,
			Authors:       sortedCopy(r.Authors),
			Skills:        sortedNames(r.Skills, func(s Skill) string { return s.Name }),
			Domains:       sortedNames(r.Domains, func(s Domain) string { return s.Name }),
			Modules:       sortedNames(r.Modules, func(s Module) string { return s.Name }),
			Trusted:       trust.Trusted,
			Verified:      trust.Verified,
			Safe:          scanVerdict(r.ScanReports),
		})
	}

	return summaries, hasMore, nil
}

// scanVerdict returns nil when reports is empty, false when any scanner
// flagged the record unsafe, and true otherwise. reports must hold only rows
// with a scanned status.
func scanVerdict(reports []ScanReport) *bool {
	if len(reports) == 0 {
		return nil
	}

	safe := true

	for i := range reports {
		if !reports[i].IsSafe {
			safe = false

			break
		}
	}

	return &safe
}

// sortedCopy returns a sorted, non-nil copy of in.
func sortedCopy(in []string) []string {
	out := make([]string, len(in))
	copy(out, in)
	sort.Strings(out)

	return out
}

// sortedNames returns the sorted, non-nil names of items.
func sortedNames[T any](items []T, name func(T) string) []string {
	out := make([]string, len(items))
	for i := range items {
		out[i] = name(items[i])
	}

	sort.Strings(out)

	return out
}
