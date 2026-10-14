// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"fmt"
	"testing"
	"time"

	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const recordQuerySeedCount = 5

// recordQueryCID returns the CID of the i-th seeded record (1-based).
func recordQueryCID(i int) string {
	return fmt.Sprintf("baeareirecordquery%045d", i)
}

// seedRecordQueryDB seeds records whose insertion time increases with i while
// their OASF created_at decreases, so record and catalog ordering disagree.
// Every record carries two skills under the same prefix, so a wildcard skill
// filter joins each record twice.
func seedRecordQueryDB(t *testing.T) *DB {
	t.Helper()

	db := setupPolicyEvalDB(t)
	base := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)

	for i := 1; i <= recordQuerySeedCount; i++ {
		cid := recordQueryCID(i)

		require.NoError(t, db.gormDB.Create(&Record{
			CreatedAt:     base.Add(time.Duration(i) * time.Hour),
			RecordCID:     cid,
			Name:          fmt.Sprintf("record-query-%d", i),
			Version:       "1.0.0",
			SchemaVersion: "0.8.0",
			OASFCreatedAt: base.AddDate(0, 0, -i).Format(time.RFC3339),
			Skills: []Skill{
				{RecordCID: cid, SkillID: 1, Name: "nlp/generation"},
				{RecordCID: cid, SkillID: 2, Name: "nlp/summarization"},
			},
			Domains: []Domain{{RecordCID: cid, DomainID: 1, Name: "technology/software"}},
			Modules: []Module{{RecordCID: cid, ModuleID: 1, Name: "integration/mcp"}},
		}).Error)
	}

	return db
}

func pageCIDs(records []Record) []string {
	cids := make([]string, len(records))
	for i := range records {
		cids[i] = records[i].RecordCID
	}

	return cids
}

func TestPageRecords_HasMoreBoundary(t *testing.T) {
	db := seedRecordQueryDB(t)

	tests := []struct {
		limit       int
		wantLen     int
		wantHasMore bool
	}{
		{limit: recordQuerySeedCount - 1, wantLen: recordQuerySeedCount - 1, wantHasMore: true},
		{limit: recordQuerySeedCount, wantLen: recordQuerySeedCount, wantHasMore: false},
		{limit: recordQuerySeedCount + 1, wantLen: recordQuerySeedCount, wantHasMore: false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("limit=%d", tt.limit), func(t *testing.T) {
			cfg := &types.RecordFilters{Limit: tt.limit}

			records, hasMore, err := db.pageRecords(db.recordQuery(cfg), cfg, applyRecordOrder)
			require.NoError(t, err)
			assert.Len(t, records, tt.wantLen)
			assert.Equal(t, tt.wantHasMore, hasMore)
		})
	}
}

func TestPageRecords_JoinedFilterReturnsEachRecordOnce(t *testing.T) {
	db := seedRecordQueryDB(t)

	// Both skill rows of every record match, so the JOIN yields two rows each.
	cfg := &types.RecordFilters{Limit: 100, SkillNames: []string{"nlp/*"}}

	records, hasMore, err := db.pageRecords(db.recordQuery(cfg), cfg, applyRecordOrder)
	require.NoError(t, err)
	assert.False(t, hasMore)
	assert.Len(t, records, recordQuerySeedCount)

	count, err := countRecords(db.recordQuery(cfg))
	require.NoError(t, err)
	assert.Equal(t, uint32(recordQuerySeedCount), count)
}

func TestCountRecords_Filtered(t *testing.T) {
	db := seedRecordQueryDB(t)

	cfg := &types.RecordFilters{Names: []string{"record-query-1"}}

	count, err := countRecords(db.recordQuery(cfg))
	require.NoError(t, err)
	assert.Equal(t, uint32(1), count)
}

func TestPageRecords_Preloads(t *testing.T) {
	db := seedRecordQueryDB(t)

	cfg := &types.RecordFilters{Limit: 1}

	records, _, err := db.pageRecords(db.recordQuery(cfg), cfg, applyRecordOrder, "Skills", "Domains", "Modules")
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Len(t, records[0].Skills, 2)
	assert.NotEmpty(t, records[0].Domains)
	assert.NotEmpty(t, records[0].Modules)

	// Associations that were not requested stay unloaded.
	records, _, err = db.pageRecords(db.recordQuery(cfg), cfg, applyRecordOrder)
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Empty(t, records[0].Skills)
}

func TestPageRecords_OffsetPagesAreDisjointAndStable(t *testing.T) {
	db := seedRecordQueryDB(t)

	const pageSize = 2

	page := func(offset int) []string {
		cfg := &types.RecordFilters{Limit: pageSize, Offset: offset}

		records, _, err := db.pageRecords(db.recordQuery(cfg), cfg, applyRecordOrder)
		require.NoError(t, err)

		return pageCIDs(records)
	}

	seen := map[string]bool{}

	for offset := 0; offset < recordQuerySeedCount; offset += pageSize {
		cids := page(offset)
		assert.Equal(t, cids, page(offset), "page at offset %d is not stable", offset)

		for _, cid := range cids {
			assert.False(t, seen[cid], "record %s appears on more than one page", cid)
			seen[cid] = true
		}
	}

	assert.Len(t, seen, recordQuerySeedCount)
}

func TestPageRecords_HonoursCallerOrder(t *testing.T) {
	db := seedRecordQueryDB(t)

	cfg := &types.RecordFilters{Limit: recordQuerySeedCount}

	// Record order: newest insertion first.
	records, _, err := db.pageRecords(db.recordQuery(cfg), cfg, applyRecordOrder)
	require.NoError(t, err)
	assert.Equal(t, []string{
		recordQueryCID(5), recordQueryCID(4), recordQueryCID(3), recordQueryCID(2), recordQueryCID(1),
	}, pageCIDs(records))

	// Catalog order: newest OASF created_at first, which is the reverse here.
	records, _, err = db.pageRecords(db.recordQuery(cfg), cfg, applyCatalogOrder)
	require.NoError(t, err)
	assert.Equal(t, []string{
		recordQueryCID(1), recordQueryCID(2), recordQueryCID(3), recordQueryCID(4), recordQueryCID(5),
	}, pageCIDs(records))
}

func TestPageRecords_UsageMetricOrder(t *testing.T) {
	db := seedRecordQueryDB(t)

	require.NoError(t, db.gormDB.Create(&RecordUsageMetrics{RecordCID: recordQueryCID(2), PullCount: 10}).Error)

	cfg := &types.RecordFilters{
		Limit:   recordQuerySeedCount,
		OrderBy: []types.RecordOrderClause{{Column: "popularity_score", Desc: true}},
	}

	records, _, err := db.pageRecords(db.recordQuery(cfg), cfg, applyRecordOrder, "Skills")
	require.NoError(t, err)
	require.Len(t, records, recordQuerySeedCount)
	assert.Equal(t, recordQueryCID(2), records[0].RecordCID)
	assert.Equal(t, "record-query-2", records[0].Name)
	assert.Len(t, records[0].Skills, 2)
}

func TestPageRecords_OrderError(t *testing.T) {
	db := seedRecordQueryDB(t)

	cfg := &types.RecordFilters{
		Limit:   1,
		OrderBy: []types.RecordOrderClause{{Column: "bogus"}},
	}

	_, _, err := db.pageRecords(db.recordQuery(cfg), cfg, applyRecordOrder)
	require.Error(t, err)
}
