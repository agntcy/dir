// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package cache

import (
	"context"
	"encoding/json"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	"github.com/ipfs/go-datastore"
	"google.golang.org/protobuf/proto"
)

var logger = logging.Logger("store/cache")

// cachedStore wraps a StoreAPI with caching functionality
type cachedStore struct {
	source types.StoreAPI
	cache  types.Datastore
}

// Wrap creates a cached store that uses the provided datastore as a cache
func Wrap(source types.StoreAPI, cache types.Datastore) types.StoreAPI {
	return &cachedStore{
		source: source,
		cache:  cache,
	}
}

// Push pushes a record to the source store and caches it
func (s *cachedStore) Push(ctx context.Context, record *corev1.Record) (*corev1.RecordRef, error) {
	logger.Debug("Push: forwarding to source store")

	// Push to source store first
	ref, err := s.source.Push(ctx, record)
	if err != nil {
		return nil, err
	}

	// Cache the record after successful push
	if err := s.cacheRecord(ctx, record); err != nil {
		logger.Debug("Failed to cache record", "cid", ref.GetCid(), "error", err)
		// Don't fail the push operation if caching fails
	}

	return ref, nil
}

// Pull pulls a record from cache first, then from source store if not found
func (s *cachedStore) Pull(ctx context.Context, ref *corev1.RecordRef) (*corev1.Record, error) {
	cid := ref.GetCid()
	logger.Debug("Pull: checking cache first", "cid", cid)

	// Try to get from cache first
	if record, err := s.getRecordFromCache(ctx, cid); err == nil {
		logger.Debug("Pull: cache hit", "cid", cid)
		return record, nil
	}

	logger.Debug("Pull: cache miss, forwarding to source store", "cid", cid)

	// Not in cache, get from source store
	record, err := s.source.Pull(ctx, ref)
	if err != nil {
		return nil, err
	}

	// Cache the record for future requests
	if err := s.cacheRecord(ctx, record); err != nil {
		logger.Debug("Failed to cache record", "cid", cid, "error", err)
		// Don't fail the pull operation if caching fails
	}

	return record, nil
}

// Lookup looks up record metadata from cache first, then from source store if not found
func (s *cachedStore) Lookup(ctx context.Context, ref *corev1.RecordRef) (*corev1.RecordMeta, error) {
	cid := ref.GetCid()
	logger.Debug("Lookup: checking cache first", "cid", cid)

	// Try to get metadata from cache first
	if meta, err := s.getMetaFromCache(ctx, cid); err == nil {
		logger.Debug("Lookup: cache hit", "cid", cid)
		return meta, nil
	}

	logger.Debug("Lookup: cache miss, forwarding to source store", "cid", cid)

	// Not in cache, get from source store
	meta, err := s.source.Lookup(ctx, ref)
	if err != nil {
		return nil, err
	}

	// Cache the metadata for future requests
	if err := s.cacheMeta(ctx, meta); err != nil {
		logger.Debug("Failed to cache metadata", "cid", cid, "error", err)
		// Don't fail the lookup operation if caching fails
	}

	return meta, nil
}

// Delete removes a record from both cache and source store
func (s *cachedStore) Delete(ctx context.Context, ref *corev1.RecordRef) error {
	cid := ref.GetCid()
	logger.Debug("Delete: removing from cache and source store", "cid", cid)

	// Remove from cache first (don't fail if not in cache)
	s.removeFromCache(ctx, cid)

	// Delete from source store
	return s.source.Delete(ctx, ref)
}

// cacheRecord stores a record in the cache
func (s *cachedStore) cacheRecord(ctx context.Context, record *corev1.Record) error {
	cid := record.GetCid()
	if cid == "" {
		return fmt.Errorf("record has no CID")
	}

	// Marshal record to bytes
	data, err := proto.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal record: %w", err)
	}

	// Store in cache with record key
	key := datastore.NewKey(fmt.Sprintf("/record/%s", cid))
	return s.cache.Put(ctx, key, data)
}

// getRecordFromCache retrieves a record from the cache
func (s *cachedStore) getRecordFromCache(ctx context.Context, cid string) (*corev1.Record, error) {
	key := datastore.NewKey(fmt.Sprintf("/record/%s", cid))

	data, err := s.cache.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	// Unmarshal record from bytes
	var record corev1.Record
	if err := proto.Unmarshal(data, &record); err != nil {
		return nil, fmt.Errorf("failed to unmarshal record: %w", err)
	}

	return &record, nil
}

// cacheMeta stores record metadata in the cache
func (s *cachedStore) cacheMeta(ctx context.Context, meta *corev1.RecordMeta) error {
	cid := meta.GetCid()
	if cid == "" {
		return fmt.Errorf("metadata has no CID")
	}

	// Marshal metadata to JSON (since it's not a protobuf message)
	data, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Store in cache with metadata key
	key := datastore.NewKey(fmt.Sprintf("/meta/%s", cid))
	return s.cache.Put(ctx, key, data)
}

// getMetaFromCache retrieves record metadata from the cache
func (s *cachedStore) getMetaFromCache(ctx context.Context, cid string) (*corev1.RecordMeta, error) {
	key := datastore.NewKey(fmt.Sprintf("/meta/%s", cid))

	data, err := s.cache.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	// Unmarshal metadata from JSON
	var meta corev1.RecordMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &meta, nil
}

// removeFromCache removes both record and metadata from cache
func (s *cachedStore) removeFromCache(ctx context.Context, cid string) {
	recordKey := datastore.NewKey(fmt.Sprintf("/record/%s", cid))
	metaKey := datastore.NewKey(fmt.Sprintf("/meta/%s", cid))

	// Remove record (ignore errors)
	if err := s.cache.Delete(ctx, recordKey); err != nil {
		logger.Debug("Failed to remove record from cache", "cid", cid, "error", err)
	}

	// Remove metadata (ignore errors)
	if err := s.cache.Delete(ctx, metaKey); err != nil {
		logger.Debug("Failed to remove metadata from cache", "cid", cid, "error", err)
	}
}
