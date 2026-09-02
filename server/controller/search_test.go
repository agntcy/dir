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
)

type fakeSearchDB struct {
	types.DatabaseAPI
	totalCount uint32
	err        error
	gotFilters types.RecordFilters

	fieldValues []types.RecordFieldValues
	gotFields   []searchv1.RecordQueryType
	fieldsCalls int
}

func (f *fakeSearchDB) CountRecords(opts ...types.FilterOption) (uint32, error) {
	for _, opt := range opts {
		if opt != nil {
			opt(&f.gotFilters)
		}
	}

	return f.totalCount, f.err
}

func (f *fakeSearchDB) ListRecordValues(fields []searchv1.RecordQueryType) ([]types.RecordFieldValues, error) {
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

func TestListRecordValues(t *testing.T) {
	db := &fakeSearchDB{
		fieldValues: []types.RecordFieldValues{
			{
				Field:  searchv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL_NAME,
				Values: []string{"nlp/summarization", "nlp/translation"},
			},
		},
	}
	ctrl := NewSearchController(db, nil)

	resp, err := ctrl.ListRecordValues(context.Background(), &searchv1.ListRecordValuesRequest{
		Fields: []searchv1.RecordQueryType{searchv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL_NAME},
	})
	require.NoError(t, err)

	assert.Equal(t, []searchv1.RecordQueryType{searchv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL_NAME}, db.gotFields)
	require.Len(t, resp.GetFields(), 1)
	assert.Equal(t, searchv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL_NAME, resp.GetFields()[0].GetField())
	assert.Equal(t, []string{"nlp/summarization", "nlp/translation"}, resp.GetFields()[0].GetValues())
}

func TestListRecordValues_UnsupportedFieldIsRejectedBeforeQuerying(t *testing.T) {
	db := &fakeSearchDB{}
	ctrl := NewSearchController(db, nil)

	_, err := ctrl.ListRecordValues(context.Background(), &searchv1.ListRecordValuesRequest{
		Fields: []searchv1.RecordQueryType{searchv1.RecordQueryType_RECORD_QUERY_TYPE_AUTHOR},
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Zero(t, db.fieldsCalls)
}

func TestListRecordValues_DatabaseError(t *testing.T) {
	ctrl := NewSearchController(&fakeSearchDB{err: assert.AnError}, nil)

	_, err := ctrl.ListRecordValues(context.Background(), &searchv1.ListRecordValuesRequest{})
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to list record values")
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
