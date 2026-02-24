// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package indexer

import (
	"errors"
	"testing"
	"time"

	ociconfig "github.com/agntcy/dir/server/store/oci/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTask_Name_Interval_IsEnabled(t *testing.T) {
	task, err := NewTask(
		Config{Enabled: true, Interval: 2 * time.Hour},
		ociconfig.Config{},
		nil,
		nil,
		nil,
	)
	require.NoError(t, err)
	require.NotNil(t, task)

	assert.Equal(t, "indexer", task.Name())
	assert.Equal(t, 2*time.Hour, task.Interval())
	assert.True(t, task.IsEnabled())
}

func TestTask_IsEnabled_False(t *testing.T) {
	task, err := NewTask(
		Config{Enabled: false, Interval: time.Hour},
		ociconfig.Config{},
		nil,
		nil,
		nil,
	)
	require.NoError(t, err)
	assert.False(t, task.IsEnabled())
}

func TestTask_Interval_Zero_UsesDefault(t *testing.T) {
	task, err := NewTask(
		Config{Enabled: true, Interval: 0},
		ociconfig.Config{},
		nil,
		nil,
		nil,
	)
	require.NoError(t, err)
	assert.Equal(t, DefaultInterval, task.Interval())
}

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
	task, _ := NewTask(Config{}, ociconfig.Config{}, nil, nil, nil)

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

func TestIsRepositoryNotFoundError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"404 name unknown", errors.New("404 name unknown"), true},
		{"404 repository name not known", errors.New("404 repository name not known"), true},
		{"404 not found", errors.New("404 not found"), true},
		{"no 404", errors.New("500 internal error"), false},
		{"404 without keyword", errors.New("404 something else"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isRepositoryNotFoundError(tt.err))
		})
	}
}
