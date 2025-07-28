// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"context"
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	objectsv1 "github.com/agntcy/dir/api/objects/v1"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/sync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockStoreAPI is a mock implementation of types.StoreAPI for testing.
type MockStoreAPI struct {
	mock.Mock
}

func (m *MockStoreAPI) Push(ctx context.Context, record *corev1.Record) (*corev1.RecordRef, error) {
	args := m.Called(ctx, record)

	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // Mock should return exact error without wrapping
	}

	ref, ok := args.Get(0).(*corev1.RecordRef)
	if !ok {
		panic("MockStoreAPI.Push: expected *corev1.RecordRef, got different type")
	}

	return ref, args.Error(1) //nolint:wrapcheck // Mock should return exact error without wrapping
}

func (m *MockStoreAPI) Pull(ctx context.Context, ref *corev1.RecordRef) (*corev1.Record, error) {
	args := m.Called(ctx, ref)

	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // Mock should return exact error without wrapping
	}

	record, ok := args.Get(0).(*corev1.Record)
	if !ok {
		panic("MockStoreAPI.Pull: expected *corev1.Record, got different type")
	}

	return record, args.Error(1) //nolint:wrapcheck // Mock should return exact error without wrapping
}

func (m *MockStoreAPI) Lookup(ctx context.Context, ref *corev1.RecordRef) (*corev1.RecordMeta, error) {
	args := m.Called(ctx, ref)

	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // Mock should return exact error without wrapping
	}

	meta, ok := args.Get(0).(*corev1.RecordMeta)
	if !ok {
		panic("MockStoreAPI.Lookup: expected *corev1.RecordMeta, got different type")
	}

	return meta, args.Error(1) //nolint:wrapcheck // Mock should return exact error without wrapping
}

func (m *MockStoreAPI) Delete(ctx context.Context, ref *corev1.RecordRef) error {
	args := m.Called(ctx, ref)

	return args.Error(0) //nolint:wrapcheck // Mock should return exact error without wrapping
}

func TestCachedStore_Push(t *testing.T) {
	ctx := t.Context()

	// Create test record
	record := &corev1.Record{
		Data: &corev1.Record_V1{
			V1: &objectsv1.Agent{
				Name:        "test-agent",
				Description: "Test agent",
				Version:     "1.0.0",
			},
		},
	}

	recordCID := record.GetCid()
	require.NotEmpty(t, recordCID, "record should have a CID")

	expectedRef := &corev1.RecordRef{Cid: recordCID}

	// Create mock store and cache
	mockStore := &MockStoreAPI{}
	cache := sync.MutexWrap(datastore.NewMapDatastore())
	cachedStore, ok := Wrap(mockStore, cache).(*cachedStore)
	require.True(t, ok, "Wrap should return *cachedStore")

	// Mock the Push call
	mockStore.On("Push", ctx, record).Return(expectedRef, nil)

	// Test Push
	ref, err := cachedStore.Push(ctx, record)
	require.NoError(t, err)
	assert.Equal(t, expectedRef, ref)

	// Test that record can be pulled from cache (verifies it was cached)
	pulledRecord, err := cachedStore.Pull(ctx, expectedRef)
	require.NoError(t, err)
	assert.Equal(t, record.GetV1().GetName(), pulledRecord.GetV1().GetName())

	// Verify mock was called only once (push), not for the pull (cache hit)
	mockStore.AssertExpectations(t)
}

func TestCachedStore_Pull_CacheHit(t *testing.T) {
	ctx := t.Context()

	// Create test record
	record := &corev1.Record{
		Data: &corev1.Record_V1{
			V1: &objectsv1.Agent{
				Name:        "test-agent",
				Description: "Test agent",
				Version:     "1.0.0",
			},
		},
	}

	recordCID := record.GetCid()
	require.NotEmpty(t, recordCID, "record should have a CID")

	ref := &corev1.RecordRef{Cid: recordCID}

	// Create mock store and cache
	mockStore := &MockStoreAPI{}
	cache := sync.MutexWrap(datastore.NewMapDatastore())
	cachedStore, ok := Wrap(mockStore, cache).(*cachedStore)
	require.True(t, ok, "Wrap should return *cachedStore")

	// Pre-cache the record
	err := cachedStore.cacheRecord(ctx, record)
	require.NoError(t, err)

	// Test Pull - should hit cache and not call source store
	pulledRecord, err := cachedStore.Pull(ctx, ref)
	require.NoError(t, err)
	assert.Equal(t, record.GetV1().GetName(), pulledRecord.GetV1().GetName())

	// Verify mock was NOT called (cache hit)
	mockStore.AssertNotCalled(t, "Pull")
}

func TestCachedStore_Pull_CacheMiss(t *testing.T) {
	ctx := t.Context()

	// Create test record
	record := &corev1.Record{
		Data: &corev1.Record_V1{
			V1: &objectsv1.Agent{
				Name:        "test-agent",
				Description: "Test agent",
				Version:     "1.0.0",
			},
		},
	}

	recordCID := record.GetCid()
	ref := &corev1.RecordRef{Cid: recordCID}

	// Create mock store and cache
	mockStore := &MockStoreAPI{}
	cache := sync.MutexWrap(datastore.NewMapDatastore())
	cachedStore := Wrap(mockStore, cache)

	// Mock the Pull call
	mockStore.On("Pull", ctx, ref).Return(record, nil)

	// Test Pull - should miss cache and call source store
	pulledRecord, err := cachedStore.Pull(ctx, ref)
	require.NoError(t, err)
	assert.Equal(t, record.GetV1().GetName(), pulledRecord.GetV1().GetName())

	// Test that subsequent pull hits cache (verifies it was cached after first pull)
	pulledRecord2, err := cachedStore.Pull(ctx, ref)
	require.NoError(t, err)
	assert.Equal(t, record.GetV1().GetName(), pulledRecord2.GetV1().GetName())

	// Verify mock was called only once (first pull), not for the second pull (cache hit)
	mockStore.AssertExpectations(t)
}

func TestCachedStore_Lookup_CacheHit(t *testing.T) {
	ctx := t.Context()

	recordCID := "test-cid-123"
	ref := &corev1.RecordRef{Cid: recordCID}

	meta := &corev1.RecordMeta{
		Cid:           recordCID,
		Annotations:   map[string]string{"test": "value"},
		SchemaVersion: "v0.3.1",
		CreatedAt:     "2023-01-01T00:00:00Z",
	}

	// Create mock store and cache
	mockStore := &MockStoreAPI{}
	cache := sync.MutexWrap(datastore.NewMapDatastore())
	cachedStore, ok := Wrap(mockStore, cache).(*cachedStore)
	require.True(t, ok, "Wrap should return *cachedStore")

	// Pre-cache the metadata
	err := cachedStore.cacheMeta(ctx, meta)
	require.NoError(t, err)

	// Test Lookup - should hit cache and not call source store
	lookedUpMeta, err := cachedStore.Lookup(ctx, ref)
	require.NoError(t, err)
	assert.Equal(t, meta.GetCid(), lookedUpMeta.GetCid())
	assert.Equal(t, meta.GetAnnotations(), lookedUpMeta.GetAnnotations())

	// Verify mock was NOT called (cache hit)
	mockStore.AssertNotCalled(t, "Lookup")
}

func TestCachedStore_Lookup_CacheMiss(t *testing.T) {
	ctx := t.Context()

	recordCID := "test-cid-123"
	ref := &corev1.RecordRef{Cid: recordCID}

	meta := &corev1.RecordMeta{
		Cid:           recordCID,
		Annotations:   map[string]string{"test": "value"},
		SchemaVersion: "v0.3.1",
		CreatedAt:     "2023-01-01T00:00:00Z",
	}

	// Create mock store and cache
	mockStore := &MockStoreAPI{}
	cache := sync.MutexWrap(datastore.NewMapDatastore())
	cachedStore := Wrap(mockStore, cache)

	// Mock the Lookup call
	mockStore.On("Lookup", ctx, ref).Return(meta, nil)

	// Test Lookup - should miss cache and call source store
	lookedUpMeta, err := cachedStore.Lookup(ctx, ref)
	require.NoError(t, err)
	assert.Equal(t, meta.GetCid(), lookedUpMeta.GetCid())
	assert.Equal(t, meta.GetAnnotations(), lookedUpMeta.GetAnnotations())

	// Test that subsequent lookup hits cache (verifies it was cached after first lookup)
	lookedUpMeta2, err := cachedStore.Lookup(ctx, ref)
	require.NoError(t, err)
	assert.Equal(t, meta.GetCid(), lookedUpMeta2.GetCid())

	// Verify mock was called only once (first lookup), not for the second lookup (cache hit)
	mockStore.AssertExpectations(t)
}

func TestCachedStore_Delete(t *testing.T) {
	ctx := t.Context()

	// Create test record and metadata
	record := &corev1.Record{
		Data: &corev1.Record_V1{
			V1: &objectsv1.Agent{
				Name:        "test-agent",
				Description: "Test agent",
				Version:     "1.0.0",
			},
		},
	}

	recordCID := record.GetCid()
	ref := &corev1.RecordRef{Cid: recordCID}

	meta := &corev1.RecordMeta{
		Cid:           recordCID,
		Annotations:   map[string]string{"test": "value"},
		SchemaVersion: "v0.3.1",
		CreatedAt:     "2023-01-01T00:00:00Z",
	}

	// Create mock store and cache
	mockStore := &MockStoreAPI{}
	cache := sync.MutexWrap(datastore.NewMapDatastore())
	cachedStore := Wrap(mockStore, cache)

	// First, populate cache by doing Pull and Lookup operations
	mockStore.On("Pull", ctx, ref).Return(record, nil)
	mockStore.On("Lookup", ctx, ref).Return(meta, nil)

	// Pull and Lookup to populate cache
	_, err := cachedStore.Pull(ctx, ref)
	require.NoError(t, err)
	_, err = cachedStore.Lookup(ctx, ref)
	require.NoError(t, err)

	// Mock the Delete call
	mockStore.On("Delete", ctx, ref).Return(nil)

	// Test Delete
	err = cachedStore.Delete(ctx, ref)
	require.NoError(t, err)

	// Verify items were removed from cache by attempting to pull/lookup
	// They should now hit the source store again (cache miss)
	mockStore.On("Pull", ctx, ref).Return(record, nil)
	mockStore.On("Lookup", ctx, ref).Return(meta, nil)

	_, err = cachedStore.Pull(ctx, ref)
	require.NoError(t, err)
	_, err = cachedStore.Lookup(ctx, ref)
	require.NoError(t, err)

	// Verify mock was called
	mockStore.AssertExpectations(t)
}
