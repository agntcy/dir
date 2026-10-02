// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package indexer

import (
	"context"
	"errors"
	"testing"

	typesv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	coretypes "github.com/agntcy/dir/api/core/types"
	corev1 "github.com/agntcy/dir/api/core/v1"
	ociconfig "github.com/agntcy/dir/server/store/oci/config"
	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateContentHash(t *testing.T) {
	tests := []struct {
		name string
		tags []string
	}{
		{"empty", []string{}},
		{"single", []string{"bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi"}},
		{"multiple", []string{"cid1", "cid2", "cid3"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := createContentHash(tt.tags)
			assert.NotEmpty(t, h)
			// Same input must produce same hash
			assert.Equal(t, h, createContentHash(tt.tags))
		})
	}
}

func TestCreateContentHash_OrderMatters(t *testing.T) {
	h1 := createContentHash([]string{"a", "b"})
	h2 := createContentHash([]string{"b", "a"})
	assert.NotEqual(t, h1, h2)
}

func TestDetectNewTags(t *testing.T) {
	task, _ := NewTask(Config{}, ociconfig.Config{}, nil, nil, nil, nil)

	tests := []struct {
		name        string
		oldSnapshot *registrySnapshot
		newSnapshot *registrySnapshot
		wantCount   int
	}{
		{
			name:        "same hash no new",
			oldSnapshot: &registrySnapshot{Tags: []string{"a", "b"}, ContentHash: createContentHash([]string{"a", "b"})},
			newSnapshot: &registrySnapshot{Tags: []string{"a", "b"}, ContentHash: createContentHash([]string{"a", "b"})},
			wantCount:   0,
		},
		{
			name:        "new tags",
			oldSnapshot: &registrySnapshot{Tags: []string{"a"}, ContentHash: createContentHash([]string{"a"})},
			newSnapshot: &registrySnapshot{Tags: []string{"a", "b", "c"}, ContentHash: createContentHash([]string{"a", "b", "c"})},
			wantCount:   2,
		},
		{
			name:        "empty old",
			oldSnapshot: &registrySnapshot{Tags: []string{}, ContentHash: createContentHash([]string{})},
			newSnapshot: &registrySnapshot{Tags: []string{"x"}, ContentHash: createContentHash([]string{"x"})},
			wantCount:   1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := task.detectNewTags(tt.oldSnapshot, tt.newSnapshot)
			assert.Len(t, got, tt.wantCount)
		})
	}
}

func TestIsDuplicateRecordError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"duplicate", errors.New("duplicate key value"), true},
		{"already exists", errors.New("record already exists"), true},
		{"unique constraint", errors.New("UNIQUE constraint failed"), true},
		{"primary key", errors.New("primary key violation"), true},
		{"other", errors.New("some other error"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isDuplicateRecordError(tt.err))
		})
	}
}

// --- OnIndexed ---

// registryOf lists the tags it is given.
type registryOf struct{ tags []string }

func (r *registryOf) Tags(_ context.Context, _ string, fn func([]string) error) error {
	return fn(r.tags)
}

// storeOf serves a record for each tag it is given, or fails the pull.
type storeOf struct {
	types.StoreAPI

	pullErr error
}

func (s *storeOf) Pull(context.Context, *corev1.RecordRef) (*corev1.Record, error) {
	if s.pullErr != nil {
		return nil, s.pullErr
	}

	return corev1.New(&typesv1alpha1.Record{Name: "agent", SchemaVersion: "0.7.0"}), nil
}

// searchDB accepts every record it is given.
type searchDB struct {
	types.SearchDatabaseAPI

	added int
}

func (d *searchDB) AddRecord(coretypes.Record) error {
	d.added++

	return nil
}

const indexedCID = "baeareidp4vt6jw7tirdvk6qlcuqndobz24yejxovcjgcuv3qnnhwzqz4mi"

func TestRun_TellsWhatDependsOnTheIndexWhenItIndexedARecord(t *testing.T) {
	t.Parallel()

	db := &searchDB{}

	task, err := NewTask(Config{Enabled: true}, ociconfig.Config{}, &storeOf{}, &registryOf{tags: []string{indexedCID}}, db, nil)
	require.NoError(t, err)

	var told int

	task.OnIndexed(func() { told++ })

	require.NoError(t, task.Run(t.Context()))
	assert.Equal(t, 1, db.added)
	assert.Equal(t, 1, told, "a run that indexed a record tells the policy task to look")

	// Nothing new in the registry: nothing to tell.
	require.NoError(t, task.Run(t.Context()))
	assert.Equal(t, 1, told, "a run that found nothing new says nothing")
}

func TestRun_SaysNothingWhenNothingWasIndexed(t *testing.T) {
	t.Parallel()

	t.Run("an empty registry", func(t *testing.T) {
		t.Parallel()

		task, err := NewTask(Config{Enabled: true}, ociconfig.Config{}, &storeOf{}, &registryOf{}, &searchDB{}, nil)
		require.NoError(t, err)

		told := false

		task.OnIndexed(func() { told = true })

		require.NoError(t, task.Run(t.Context()))
		assert.False(t, told)
	})

	t.Run("a record that could not be indexed", func(t *testing.T) {
		t.Parallel()

		db := &searchDB{}
		store := &storeOf{pullErr: errors.New("registry unavailable")}

		task, err := NewTask(Config{Enabled: true}, ociconfig.Config{}, store, &registryOf{tags: []string{indexedCID}}, db, nil)
		require.NoError(t, err)

		told := false

		task.OnIndexed(func() { told = true })

		require.NoError(t, task.Run(t.Context()))
		assert.Zero(t, db.added)
		assert.False(t, told, "a failure indexes nothing, so there is nothing new to evaluate")
	})

	t.Run("no one is listening", func(t *testing.T) {
		t.Parallel()

		task, err := NewTask(Config{Enabled: true}, ociconfig.Config{}, &storeOf{}, &registryOf{tags: []string{indexedCID}}, &searchDB{}, nil)
		require.NoError(t, err)

		assert.NotPanics(t, func() { _ = task.Run(t.Context()) })
	})
}
