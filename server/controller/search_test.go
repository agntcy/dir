// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"testing"

	searchv1 "github.com/agntcy/dir/api/search/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type fakeSearchDB struct {
	types.DatabaseAPI
	totalCount uint32
	err        error
	gotFilters types.RecordFilters

	summaries    []*searchv1.RecordSummary
	hasMore      bool
	summariesErr error
	countErr     error
	gotPage      types.RecordFilters
	pageCalls    int
	countCalls   int

	fieldValues []types.FilterFieldValues
	gotFields   []searchv1.RecordQueryType
	fieldsCalls int
}

func (f *fakeSearchDB) CountRecords(opts ...types.FilterOption) (uint32, error) {
	f.countCalls++

	if f.countErr != nil {
		return 0, f.countErr
	}

	for _, opt := range opts {
		if opt != nil {
			opt(&f.gotFilters)
		}
	}

	return f.totalCount, f.err
}

func (f *fakeSearchDB) ListRecordSummaries(opts ...types.FilterOption) ([]*searchv1.RecordSummary, bool, error) {
	f.pageCalls++

	for _, opt := range opts {
		if opt != nil {
			opt(&f.gotPage)
		}
	}

	if f.summariesErr != nil {
		return nil, false, f.summariesErr
	}

	return f.summaries, f.hasMore, f.err
}

func (f *fakeSearchDB) ListFilterValues(fields []searchv1.RecordQueryType) ([]types.FilterFieldValues, error) {
	f.gotFields = fields
	f.fieldsCalls++

	return f.fieldValues, f.err
}

func TestCountRecords(t *testing.T) {
	db := &fakeSearchDB{totalCount: 7}
	ctrl := NewSearchController(db, nil)

	resp, err := ctrl.CountRecords(context.Background(), &searchv1.CountRecordsRequest{
		Queries: []*searchv1.RecordQuery{
			{
				Type:  searchv1.RecordQueryType_RECORD_QUERY_TYPE_NAME,
				Value: "*assistant*",
			},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, uint32(7), resp.GetTotalCount())
	assert.Equal(t, []string{"*assistant*"}, db.gotFilters.Names)
	assert.Zero(t, db.gotFilters.Limit)
	assert.Zero(t, db.gotFilters.Offset)
	assert.Empty(t, db.gotFilters.OrderBy)
}

func TestCountRecords_InvalidQuery(t *testing.T) {
	ctrl := NewSearchController(&fakeSearchDB{}, nil)

	_, err := ctrl.CountRecords(context.Background(), &searchv1.CountRecordsRequest{
		Queries: []*searchv1.RecordQuery{
			{
				Type:  searchv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL_ID,
				Value: "not-a-number",
			},
		},
	})
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to create filter options")
}

func TestCountRecords_DatabaseError(t *testing.T) {
	ctrl := NewSearchController(&fakeSearchDB{err: assert.AnError}, nil)

	_, err := ctrl.CountRecords(context.Background(), &searchv1.CountRecordsRequest{})
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to count records")
}

func TestListFilterValues(t *testing.T) {
	db := &fakeSearchDB{
		fieldValues: []types.FilterFieldValues{
			{
				Field:  searchv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL_NAME,
				Values: []string{"nlp/summarization", "nlp/translation"},
			},
		},
	}
	ctrl := NewSearchController(db, nil)

	resp, err := ctrl.ListFilterValues(context.Background(), &searchv1.ListFilterValuesRequest{
		Fields: []searchv1.RecordQueryType{searchv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL_NAME},
	})
	require.NoError(t, err)

	assert.Equal(t, []searchv1.RecordQueryType{searchv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL_NAME}, db.gotFields)
	require.Len(t, resp.GetFields(), 1)
	assert.Equal(t, searchv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL_NAME, resp.GetFields()[0].GetField())
	assert.Equal(t, []string{"nlp/summarization", "nlp/translation"}, resp.GetFields()[0].GetValues())
}

func TestListFilterValues_UnsupportedFieldIsRejectedBeforeQuerying(t *testing.T) {
	db := &fakeSearchDB{}
	ctrl := NewSearchController(db, nil)

	_, err := ctrl.ListFilterValues(context.Background(), &searchv1.ListFilterValuesRequest{
		Fields: []searchv1.RecordQueryType{searchv1.RecordQueryType_RECORD_QUERY_TYPE_ANNOTATION},
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Zero(t, db.fieldsCalls)
}

// Repeating a field has no useful meaning, and each occurrence would cost
// another full scan and another copy of its values in the response.
func TestListFilterValues_DuplicateFieldIsRejectedBeforeQuerying(t *testing.T) {
	db := &fakeSearchDB{}
	ctrl := NewSearchController(db, nil)

	_, err := ctrl.ListFilterValues(context.Background(), &searchv1.ListFilterValuesRequest{
		Fields: []searchv1.RecordQueryType{
			searchv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL_NAME,
			searchv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL_NAME,
		},
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Zero(t, db.fieldsCalls)
}

func TestListFilterValues_DatabaseError(t *testing.T) {
	ctrl := NewSearchController(&fakeSearchDB{err: assert.AnError}, nil)

	_, err := ctrl.ListFilterValues(context.Background(), &searchv1.ListFilterValuesRequest{})
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to list filter values")
}

func TestCountRecords_NegatedQuery(t *testing.T) {
	db := &fakeSearchDB{totalCount: 2}
	ctrl := NewSearchController(db, nil)

	resp, err := ctrl.CountRecords(context.Background(), &searchv1.CountRecordsRequest{
		Queries: []*searchv1.RecordQuery{
			{
				Type:   searchv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL_NAME,
				Value:  "nlp",
				Negate: true,
			},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, uint32(2), resp.GetTotalCount())
	assert.Equal(t, []string{"nlp"}, db.gotFilters.Excluded.SkillNames)
	assert.Empty(t, db.gotFilters.SkillNames)
}

func TestListRecordSummaries(t *testing.T) {
	db := &fakeSearchDB{
		totalCount: 120,
		hasMore:    true,
		summaries:  []*searchv1.RecordSummary{{Cid: "cid-1", Name: "a"}, {Cid: "cid-2", Name: "b"}},
	}
	ctrl := NewSearchController(db, nil)

	resp, err := ctrl.ListRecordSummaries(context.Background(), &searchv1.ListRecordSummariesRequest{
		Queries: []*searchv1.RecordQuery{
			{Type: searchv1.RecordQueryType_RECORD_QUERY_TYPE_NAME, Value: "*assistant*"},
		},
		Limit:  proto.Uint32(10),
		Offset: proto.Uint32(20),
	})
	require.NoError(t, err)

	require.Len(t, resp.GetRecords(), 2)
	assert.Equal(t, "cid-1", resp.GetRecords()[0].GetCid())
	assert.True(t, resp.GetHasMore())
	assert.Equal(t, uint32(120), resp.GetTotalCount())

	assert.Equal(t, []string{"*assistant*"}, db.gotPage.Names)
	assert.Equal(t, 10, db.gotPage.Limit)
	assert.Equal(t, 20, db.gotPage.Offset)

	// The count uses the same filters but no paging or ordering.
	assert.Equal(t, 1, db.countCalls)
	assert.Equal(t, []string{"*assistant*"}, db.gotFilters.Names)
	assert.Zero(t, db.gotFilters.Limit)
	assert.Zero(t, db.gotFilters.Offset)
	assert.Empty(t, db.gotFilters.OrderBy)
}

func TestListRecordSummaries_PageSize(t *testing.T) {
	tests := []struct {
		name  string
		limit *uint32
		want  int
	}{
		{"unset defaults", nil, types.DefaultRecordSummaryPageSize},
		{"zero defaults", proto.Uint32(0), types.DefaultRecordSummaryPageSize},
		{"above max clamped", proto.Uint32(500), types.MaxRecordSummaryPageSize},
		{"explicit kept", proto.Uint32(10), 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &fakeSearchDB{}
			ctrl := NewSearchController(db, nil)

			_, err := ctrl.ListRecordSummaries(context.Background(), &searchv1.ListRecordSummariesRequest{Limit: tt.limit})
			require.NoError(t, err)
			assert.Equal(t, tt.want, db.gotPage.Limit)
		})
	}
}

func TestListRecordSummaries_SortMode(t *testing.T) {
	db := &fakeSearchDB{}
	ctrl := NewSearchController(db, nil)

	_, err := ctrl.ListRecordSummaries(context.Background(), &searchv1.ListRecordSummariesRequest{
		SortMode: searchv1.SortMode_SORT_MODE_POPULARITY,
	})
	require.NoError(t, err)
	assert.Equal(t, []types.RecordOrderClause{{Column: "popularity_score", Desc: true}}, db.gotPage.OrderBy)
	assert.Empty(t, db.gotFilters.OrderBy)
}

func TestListRecordSummaries_InvalidQuery(t *testing.T) {
	db := &fakeSearchDB{}
	ctrl := NewSearchController(db, nil)

	_, err := ctrl.ListRecordSummaries(context.Background(), &searchv1.ListRecordSummariesRequest{
		Queries: []*searchv1.RecordQuery{
			{Type: searchv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL_ID, Value: "not-a-number"},
		},
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "failed to create filter options")
	assert.Zero(t, db.pageCalls)
	assert.Zero(t, db.countCalls)
}

func TestListRecordSummaries_DatabaseErrors(t *testing.T) {
	t.Run("list error", func(t *testing.T) {
		ctrl := NewSearchController(&fakeSearchDB{summariesErr: assert.AnError}, nil)

		_, err := ctrl.ListRecordSummaries(context.Background(), &searchv1.ListRecordSummariesRequest{})
		require.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to list record summaries")
	})

	t.Run("count error", func(t *testing.T) {
		ctrl := NewSearchController(&fakeSearchDB{countErr: assert.AnError}, nil)

		_, err := ctrl.ListRecordSummaries(context.Background(), &searchv1.ListRecordSummariesRequest{})
		require.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to count records")
	})
}
