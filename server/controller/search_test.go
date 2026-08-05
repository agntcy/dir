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
)

type fakeSearchDB struct {
	types.DatabaseAPI
	totalCount uint32
	err        error
	gotFilters types.RecordFilters
}

func (f *fakeSearchDB) CountRecords(opts ...types.FilterOption) (uint32, error) {
	for _, opt := range opts {
		if opt != nil {
			opt(&f.gotFilters)
		}
	}

	return f.totalCount, f.err
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
