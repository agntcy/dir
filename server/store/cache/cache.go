// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"io"

	storetypes "github.com/agntcy/dir/server/store/types"
	"github.com/agntcy/dir/server/types"
	"github.com/ipfs/go-datastore"
)

type store struct {
	cache  types.Datastore
	source types.StoreAPI
}

// TODO: benchmark cached vs non-cached.
func Wrap(source types.StoreAPI, cache types.Datastore) types.StoreAPI {
	if cache == nil {
		return source
	}

	return &store{
		cache:  cache,
		source: source,
	}
}

func (s *store) Push(ctx context.Context, ref types.Object, reader io.Reader) (types.Object, error) {
	// push data
	ref, err := s.source.Push(ctx, ref, reader)
	if err != nil {
		return nil, err
	}

	// write cache
	_ = s.cacheWrite(ctx, ref)

	return ref, nil
}

func (s *store) Pull(ctx context.Context, ref types.ObjectRef) (io.ReadCloser, error) {
	return s.source.Pull(ctx, ref)
}

func (s *store) Lookup(ctx context.Context, ref types.ObjectRef) (types.Object, error) {
	// read cache
	found, cachedRef, _ := s.cacheRead(ctx, ref)
	if found {
		return cachedRef, nil
	}

	// fetch from source
	sourceRef, err := s.source.Lookup(ctx, ref)
	if err != nil {
		return nil, err
	}

	// write cache
	_ = s.cacheWrite(ctx, sourceRef)

	return sourceRef, nil
}

func (s *store) Delete(ctx context.Context, ref types.ObjectRef) error {
	// delete
	if err := s.source.Delete(ctx, ref); err != nil {
		return err
	}

	// remove cache key
	_ = s.cache.Delete(ctx, getCacheKey(ref))

	return nil
}

func (s *store) cacheRead(ctx context.Context, ref types.ObjectRef) (bool, types.Object, error) {
	cacheKey := getCacheKey(ref)

	// check cache
	exists, err := s.cache.Has(ctx, cacheKey)
	if err != nil {
		return false, nil, err
	}

	if !exists {
		return false, nil, errors.New("not found")
	}

	// read cache
	cachedData, err := s.cache.Get(ctx, cacheKey)
	if err != nil {
		return false, nil, err
	}

	// convert object
	cachedRef := &storetypes.Object{}
	if err := json.Unmarshal(cachedData, &cachedRef); err != nil {
		return false, nil, err
	}

	// return cache
	return true, cachedRef, nil
}

func (s *store) cacheWrite(ctx context.Context, ref types.Object) error {
	// convert object
	toCache, err := json.Marshal(ref)
	if err != nil {
		return err
	}

	// write cache
	cacheKey := getCacheKey(ref)

	err = s.cache.Put(ctx, cacheKey, toCache)
	if err != nil {
		return err
	}

	return nil
}

func getCacheKey(ref types.ObjectRef) datastore.Key {
	return datastore.KeyWithNamespaces([]string{"store", ref.CID()})
}
