// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package prune

import (
	"context"
	"errors"
	"testing"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNewTask_InvalidMinSeverity(t *testing.T) {
	t.Parallel()

	task, err := NewTask(Config{Enabled: true, Criteria: Criteria{MinSeverity: "MEDUIM"}}, nil, nil)
	require.Error(t, err)
	assert.Nil(t, task)
	assert.Contains(t, err.Error(), `invalid min_severity "MEDUIM"`)
}

func TestTask_Name_Interval_IsEnabled(t *testing.T) {
	t.Parallel()

	task, err := NewTask(Config{Enabled: true, Interval: 2 * time.Minute}, nil, nil)
	require.NoError(t, err)

	assert.Equal(t, "prune", task.Name())
	assert.Equal(t, 2*time.Minute, task.Interval())
	assert.True(t, task.IsEnabled())
}

func TestTask_IsEnabled_False(t *testing.T) {
	t.Parallel()

	task, err := NewTask(Config{Enabled: false}, nil, nil)
	require.NoError(t, err)
	assert.False(t, task.IsEnabled())
}

func TestTask_Interval_Zero_UsesDefault(t *testing.T) {
	t.Parallel()

	task, err := NewTask(Config{Enabled: true}, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, DefaultInterval, task.Interval())
}

func TestTask_Run_NoRecords_ReturnsNil(t *testing.T) {
	t.Parallel()

	db := &fakeIndex{
		getRecordCIDs: func(_ ...types.FilterOption) ([]string, error) {
			return nil, nil
		},
	}
	task, err := NewTask(Config{Enabled: true}, db, &fakeStore{})
	require.NoError(t, err)

	assert.NoError(t, task.Run(context.Background()))
	assert.Empty(t, db.removed)
}

func TestTask_Run_DBError_ReturnsError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("db unavailable")
	db := &fakeIndex{
		getRecordCIDs: func(_ ...types.FilterOption) ([]string, error) {
			return nil, wantErr
		},
	}
	task, err := NewTask(Config{Enabled: true}, db, &fakeStore{})
	require.NoError(t, err)

	err = task.Run(context.Background())
	require.Error(t, err)
	require.ErrorIs(t, err, wantErr)
	assert.Contains(t, err.Error(), "get records")
}

func TestTask_Run_AppliesTrustedAndScanSeverityFilters(t *testing.T) {
	t.Parallel()

	var got types.RecordFilters

	db := &fakeIndex{
		getRecordCIDs: func(opts ...types.FilterOption) ([]string, error) {
			for _, opt := range opts {
				opt(&got)
			}

			return nil, nil
		},
	}
	task, err := NewTask(Config{Enabled: true, Criteria: Criteria{MinSeverity: "HIGH"}, Limit: 25}, db, &fakeStore{})
	require.NoError(t, err)

	require.NoError(t, task.Run(context.Background()))
	require.NotNil(t, got.Trusted)
	assert.False(t, *got.Trusted)
	assert.Equal(t, []string{"HIGH"}, got.ScanSeverities)
	assert.Equal(t, 25, got.Limit)
	require.NotNil(t, got.IndexedBefore)
	assert.WithinDuration(t, time.Now().Add(-DefaultOlderThan), *got.IndexedBefore, 5*time.Second)
}

func TestTask_Run_AppliesConfiguredOlderThan(t *testing.T) {
	t.Parallel()

	var got types.RecordFilters

	db := &fakeIndex{
		getRecordCIDs: func(opts ...types.FilterOption) ([]string, error) {
			for _, opt := range opts {
				opt(&got)
			}

			return nil, nil
		},
	}
	task, err := NewTask(Config{Enabled: true, Criteria: Criteria{OlderThan: 24 * time.Hour}}, db, &fakeStore{})
	require.NoError(t, err)

	require.NoError(t, task.Run(context.Background()))
	require.NotNil(t, got.IndexedBefore)
	assert.WithinDuration(t, time.Now().Add(-24*time.Hour), *got.IndexedBefore, 5*time.Second)
}

func TestTask_Run_AppliesConfiguredTrusted(t *testing.T) {
	t.Parallel()

	var got types.RecordFilters

	db := &fakeIndex{
		getRecordCIDs: func(opts ...types.FilterOption) ([]string, error) {
			for _, opt := range opts {
				opt(&got)
			}

			return nil, nil
		},
	}
	task, err := NewTask(Config{Enabled: true, Criteria: Criteria{Trusted: true}}, db, &fakeStore{})
	require.NoError(t, err)

	require.NoError(t, task.Run(context.Background()))
	require.NotNil(t, got.Trusted)
	assert.True(t, *got.Trusted)
}

func TestTask_Run_DeletesMatchingRecords(t *testing.T) {
	t.Parallel()

	db := &fakeIndex{
		getRecordCIDs: func(_ ...types.FilterOption) ([]string, error) {
			return []string{"cid-a", "cid-b"}, nil
		},
	}
	store := &fakeStore{}
	task, err := NewTask(Config{Enabled: true, RecordTimeout: time.Second, DryRun: false}, db, store)
	require.NoError(t, err)

	require.NoError(t, task.Run(context.Background()))
	assert.Equal(t, []string{"cid-a", "cid-b"}, store.deleted)
	assert.Equal(t, []string{"cid-a", "cid-b"}, db.removed)
}

func TestTask_Run_DryRun_DoesNotDelete(t *testing.T) {
	t.Parallel()

	db := &fakeIndex{
		getRecordCIDs: func(_ ...types.FilterOption) ([]string, error) {
			return []string{"cid-a", "cid-b"}, nil
		},
	}
	store := &fakeStore{}
	task, err := NewTask(Config{Enabled: true, DryRun: true}, db, store)
	require.NoError(t, err)

	require.NoError(t, task.Run(context.Background()))
	assert.Empty(t, store.deleted)
	assert.Empty(t, db.removed)
}

func TestTask_Run_StoreNotFound_StillRemovesIndex(t *testing.T) {
	t.Parallel()

	db := &fakeIndex{
		getRecordCIDs: func(_ ...types.FilterOption) ([]string, error) {
			return []string{"missing"}, nil
		},
	}
	store := &fakeStore{
		delete: func(_ context.Context, _ *corev1.RecordRef) error {
			return status.Error(codes.NotFound, "record not found")
		},
	}
	task, err := NewTask(Config{Enabled: true}, db, store)
	require.NoError(t, err)

	require.NoError(t, task.Run(context.Background()))
	assert.Equal(t, []string{"missing"}, db.removed)
}

func TestTask_Run_StoreError_DoesNotRemoveIndex(t *testing.T) {
	t.Parallel()

	db := &fakeIndex{
		getRecordCIDs: func(_ ...types.FilterOption) ([]string, error) {
			return []string{"cid-a", "cid-b"}, nil
		},
	}
	store := &fakeStore{
		delete: func(_ context.Context, ref *corev1.RecordRef) error {
			if ref.GetCid() == "cid-a" {
				return errors.New("oci unavailable")
			}

			return nil
		},
	}
	task, err := NewTask(Config{Enabled: true}, db, store)
	require.NoError(t, err)

	require.NoError(t, task.Run(context.Background()))
	assert.Equal(t, []string{"cid-b"}, db.removed)
}

func TestTask_Run_CanceledContext_Stops(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	db := &fakeIndex{
		getRecordCIDs: func(_ ...types.FilterOption) ([]string, error) {
			return []string{"cid-a"}, nil
		},
	}
	task, err := NewTask(Config{Enabled: true}, db, &fakeStore{})
	require.NoError(t, err)

	err = task.Run(ctx)
	require.Error(t, err)
	require.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, db.removed)
}

type fakeIndex struct {
	getRecordCIDs func(opts ...types.FilterOption) ([]string, error)
	removeRecord  func(cid string) error
	removed       []string
}

func (f *fakeIndex) GetRecordCIDs(opts ...types.FilterOption) ([]string, error) {
	if f.getRecordCIDs != nil {
		return f.getRecordCIDs(opts...)
	}

	return nil, nil
}

func (f *fakeIndex) RemoveRecord(cid string) error {
	f.removed = append(f.removed, cid)
	if f.removeRecord != nil {
		return f.removeRecord(cid)
	}

	return nil
}

type fakeStore struct {
	delete  func(ctx context.Context, ref *corev1.RecordRef) error
	deleted []string
}

func (f *fakeStore) Delete(ctx context.Context, ref *corev1.RecordRef) error {
	f.deleted = append(f.deleted, ref.GetCid())
	if f.delete != nil {
		return f.delete(ctx, ref)
	}

	return nil
}
