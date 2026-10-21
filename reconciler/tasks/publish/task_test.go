// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package publish

import (
	"context"
	"errors"
	"testing"
	"time"

	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTask_Name_Interval_IsEnabled(t *testing.T) {
	t.Parallel()

	task, err := NewTask(Config{Enabled: true, Interval: 2 * time.Hour}, nil)
	require.NoError(t, err)

	assert.Equal(t, "publish", task.Name())
	assert.Equal(t, 2*time.Hour, task.Interval())
	assert.True(t, task.IsEnabled())
}

func TestTask_IsEnabled_False(t *testing.T) {
	t.Parallel()

	task, err := NewTask(Config{Enabled: false}, nil)
	require.NoError(t, err)
	assert.False(t, task.IsEnabled())
}

func TestTask_Interval_Zero_UsesDefault(t *testing.T) {
	t.Parallel()

	task, err := NewTask(Config{Enabled: true}, nil)
	require.NoError(t, err)
	assert.Equal(t, DefaultInterval, task.Interval())
}

func TestTask_Run_NoRecords_ReturnsNil(t *testing.T) {
	t.Parallel()

	db := &fakeStore{}
	task, err := NewTask(Config{Enabled: true}, db)
	require.NoError(t, err)

	assert.NoError(t, task.Run(context.Background()))
	assert.Empty(t, db.created)
}

func TestTask_Run_DBError_ReturnsError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("db unavailable")
	db := &fakeStore{
		getRecordCIDs: func(_ ...types.FilterOption) ([]string, error) {
			return nil, wantErr
		},
	}
	task, err := NewTask(Config{Enabled: true}, db)
	require.NoError(t, err)

	err = task.Run(context.Background())
	require.Error(t, err)
	require.ErrorIs(t, err, wantErr)
	assert.Contains(t, err.Error(), "get records")
	assert.Empty(t, db.created)
}

func TestTask_Run_AppliesDomainAndSkillFilters(t *testing.T) {
	t.Parallel()

	var got types.RecordFilters

	db := &fakeStore{
		getRecordCIDs: func(opts ...types.FilterOption) ([]string, error) {
			for _, opt := range opts {
				opt(&got)
			}

			return nil, nil
		},
	}
	task, err := NewTask(Config{
		Enabled:  true,
		Criteria: Criteria{Domains: []string{"research", "analytics"}, Skills: []string{"threat_detection", "nlp"}},
		Limit:    25,
	}, db)
	require.NoError(t, err)

	require.NoError(t, task.Run(context.Background()))
	assert.Equal(t, []string{"research", "analytics"}, got.DomainNames)
	assert.Equal(t, []string{"threat_detection", "nlp"}, got.SkillNames)
	assert.Equal(t, 25, got.Limit)
	assert.Equal(t, 0, got.Offset)
}

func TestTask_Run_EmptyDomains_DoesNotFilter(t *testing.T) {
	t.Parallel()

	var got types.RecordFilters

	db := &fakeStore{
		getRecordCIDs: func(opts ...types.FilterOption) ([]string, error) {
			for _, opt := range opts {
				opt(&got)
			}

			return nil, nil
		},
	}
	task, err := NewTask(Config{Enabled: true}, db)
	require.NoError(t, err)

	require.NoError(t, task.Run(context.Background()))
	assert.Empty(t, got.DomainNames)
	assert.Empty(t, got.SkillNames)
	assert.Nil(t, got.Trusted)
}

func TestTask_Run_TrustedFilter(t *testing.T) {
	t.Parallel()

	for _, want := range []bool{true, false} {
		t.Run(map[bool]string{true: "true", false: "false"}[want], func(t *testing.T) {
			t.Parallel()

			var got types.RecordFilters

			db := &fakeStore{
				getRecordCIDs: func(opts ...types.FilterOption) ([]string, error) {
					for _, opt := range opts {
						opt(&got)
					}

					return nil, nil
				},
			}
			trusted := want
			task, err := NewTask(Config{Enabled: true, Criteria: Criteria{Trusted: &trusted}}, db)
			require.NoError(t, err)

			require.NoError(t, task.Run(context.Background()))
			require.NotNil(t, got.Trusted)
			assert.Equal(t, want, *got.Trusted)
		})
	}
}

func TestTask_Run_DryRun_DoesNotCreatePublication(t *testing.T) {
	t.Parallel()

	db := &fakeStore{
		pages: [][]string{{"cid-1", "cid-2"}},
	}
	task, err := NewTask(Config{Enabled: true, DryRun: true}, db)
	require.NoError(t, err)

	require.NoError(t, task.Run(context.Background()))
	assert.Empty(t, db.created)
}

func TestTask_Run_CreatesPublicationPerPage(t *testing.T) {
	t.Parallel()

	db := &fakeStore{
		pages: [][]string{
			{"cid-1", "cid-2"},
			{"cid-3"},
		},
	}
	task, err := NewTask(Config{Enabled: true, Limit: 2}, db)
	require.NoError(t, err)

	require.NoError(t, task.Run(context.Background()))
	require.Len(t, db.created, 2)
	assert.Equal(t, []string{"cid-1", "cid-2"}, db.created[0])
	assert.Equal(t, []string{"cid-3"}, db.created[1])
	assert.Equal(t, []int{0, 2}, db.offsets)
}

func TestTask_Run_CreateError_ReturnsError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("queue down")
	db := &fakeStore{
		pages: [][]string{{"cid-1"}},
		err:   wantErr,
	}
	task, err := NewTask(Config{Enabled: true}, db)
	require.NoError(t, err)

	err = task.Run(context.Background())
	require.Error(t, err)
	require.ErrorIs(t, err, wantErr)
	assert.Contains(t, err.Error(), "create publication")
}

func TestTask_Run_Canceled_ReturnsError(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	db := &fakeStore{pages: [][]string{{"cid-1"}}}
	task, err := NewTask(Config{Enabled: true}, db)
	require.NoError(t, err)

	err = task.Run(ctx)
	require.Error(t, err)
	require.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, db.created)
}

type fakeStore struct {
	getRecordCIDs func(opts ...types.FilterOption) ([]string, error)
	pages         [][]string
	err           error
	created       [][]string
	offsets       []int
}

func (f *fakeStore) GetRecordCIDs(opts ...types.FilterOption) ([]string, error) {
	if f.getRecordCIDs != nil {
		return f.getRecordCIDs(opts...)
	}

	var filters types.RecordFilters
	for _, opt := range opts {
		opt(&filters)
	}

	f.offsets = append(f.offsets, filters.Offset)

	page := filters.Offset / max(filters.Limit, 1)
	if page >= len(f.pages) {
		return nil, nil
	}

	return f.pages[page], nil
}

func (f *fakeStore) CreatePublication(request *routingv1.PublishRequest) (string, error) {
	if f.err != nil {
		return "", f.err
	}

	refs := request.GetRecordRefs().GetRefs()

	cids := make([]string, 0, len(refs))
	for _, ref := range refs {
		cids = append(cids, ref.GetCid())
	}

	f.created = append(f.created, cids)

	return "pub-1", nil
}
