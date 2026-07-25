// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package datastore

import (
	"fmt"
	"sync"
	"testing"

	ds "github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	"github.com/stretchr/testify/require"
)

func TestNewMemoryDatastoreConcurrentQueryAndPut(t *testing.T) {
	const (
		seedEntryCount = 1024
		workerCount    = 4
		iterationCount = 256
	)

	ctx := t.Context()

	store, err := New()
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, store.Close())
	})

	for i := 0; i < seedEntryCount; i++ {
		key := ds.NewKey(fmt.Sprintf("/skills/seed/%d", i))
		require.NoError(t, store.Put(ctx, key, []byte("seed")))
	}

	start := make(chan struct{})
	errCh := make(chan error, workerCount*2)

	var wg sync.WaitGroup

	for worker := 0; worker < workerCount; worker++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			<-start

			for i := 0; i < iterationCount; i++ {
				key := ds.NewKey(fmt.Sprintf("/skills/writer-%d/%d", worker, i))
				if err := store.Put(ctx, key, []byte("value")); err != nil {
					errCh <- fmt.Errorf("put %s: %w", key, err)

					return
				}
			}
		}()
	}

	for range workerCount {
		wg.Add(1)

		go func() {
			defer wg.Done()
			<-start

			for range iterationCount {
				results, err := store.Query(ctx, query.Query{Prefix: "/skills"})
				if err != nil {
					errCh <- fmt.Errorf("query labels: %w", err)

					return
				}

				_, readErr := results.Rest()
				closeErr := results.Close()

				if readErr != nil {
					errCh <- fmt.Errorf("read query results: %w", readErr)

					return
				}

				if closeErr != nil {
					errCh <- fmt.Errorf("close query results: %w", closeErr)

					return
				}
			}
		}()
	}

	close(start)
	wg.Wait()
	close(errCh)

	for err := range errCh {
		require.NoError(t, err)
	}
}
