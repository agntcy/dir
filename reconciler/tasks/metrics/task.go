// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package metrics implements the reconciliation task for refreshing computed
// usage metrics. It currently refreshes provider counts by querying the routing
// layer for all locally known CIDs and persisting the number of distinct
// announcing peers into the record_usage_metrics table. Additional derived
// metrics (e.g. blended popularity score) can be added here as the epic
// progresses.
package metrics

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
)

var logger = logging.Logger("reconciler/metrics")

// providerCountWorkers bounds how many provider lookups are in flight at once.
const providerCountWorkers = 8

// maxProviderCount is an upper bound for a stored provider count. A real DHT
// lookup returns orders of magnitude fewer peers; the cap exists so the
// conversion to the column's width cannot wrap.
const maxProviderCount = 1 << 20

func clampProviderCount(count int) uint32 {
	if count < 0 {
		return 0
	}

	if count > maxProviderCount {
		return maxProviderCount
	}

	return uint32(count)
}

// ProviderCounterAPI is the minimal interface required by the metrics task to
// query provider counts. It is satisfied by types.RoutingAPI (daemon mode, where
// the routing layer is shared in-process) and by GRPCProviderCounter (standalone
// reconciler mode, where the routing layer is reached over gRPC).
//
// Note on the routing datastore: the routing layer uses an embedded Badger
// key-value store that does NOT support concurrent multi-process access. Sharing
// the datastore directory via a volume mount between the reconciler and the server
// is not safe. The gRPC client (GRPCProviderCounter) is the correct mechanism for
// the standalone reconciler to query provider counts from the server.
type ProviderCounterAPI interface {
	GetProviderCount(ctx context.Context, cid string) (int, error)
}

// Task implements the usage-metrics reconciliation task.
type Task struct {
	config   Config
	db       types.UsageMetricsDatabaseAPI
	search   types.SearchDatabaseAPI
	counters ProviderCounterAPI
}

// NewTask creates a new usage-metrics reconciliation task.
func NewTask(config Config, db types.DatabaseAPI, counters ProviderCounterAPI) (*Task, error) {
	return &Task{
		config:   config,
		db:       db,
		search:   db,
		counters: counters,
	}, nil
}

// Name returns the task name.
func (t *Task) Name() string {
	return "metrics"
}

// Interval returns how often this task should run.
func (t *Task) Interval() time.Duration {
	return t.config.GetInterval()
}

// IsEnabled returns whether this task is enabled.
func (t *Task) IsEnabled() bool {
	return t.config.Enabled
}

// Run refreshes all computed usage metrics for locally known records.
func (t *Task) Run(ctx context.Context) error {
	logger.Debug("Running usage-metrics reconciliation")

	if err := t.refreshProviderCounts(ctx); err != nil {
		return err
	}

	return nil
}

// refreshProviderCounts queries the routing layer for each locally known CID
// and persists the number of distinct announcing peers as provider_count.
func (t *Task) refreshProviderCounts(ctx context.Context) error {
	cids, err := t.search.GetRecordCIDs()
	if err != nil {
		return fmt.Errorf("failed to list local record CIDs: %w", err)
	}

	if len(cids) == 0 {
		logger.Debug("No local records to update provider counts for")

		return nil
	}

	logger.Info("Refreshing provider counts", "records", len(cids))

	var updated, failed int

	// Persist serially while the lookups run concurrently. The lookups are the
	// slow part; the metrics table serialises writers regardless.
	for res := range t.countProviders(ctx, cids) {
		if res.err != nil {
			logger.Warn("Failed to get provider count", "cid", res.cid, "error", res.err)

			failed++

			continue
		}

		count := clampProviderCount(res.count)

		if err := t.db.SetProviderCount(res.cid, count); err != nil {
			logger.Warn("Failed to set provider count", "cid", res.cid, "count", count, "error", err)

			failed++

			continue
		}

		updated++
	}

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context cancelled: %w", err)
	}

	logger.Info("Provider count refresh complete", "updated", updated, "failed", failed)

	return nil
}

type providerCountResult struct {
	cid   string
	count int
	err   error
}

// countProviders fans the CID list out over a bounded worker pool and streams
// results back as they land. Each GetProviderCount is a DHT lookup, so running
// one CID at a time would exceed the reconciliation interval on any sizeable
// corpus.
func (t *Task) countProviders(ctx context.Context, cids []string) <-chan providerCountResult {
	jobs := make(chan string)
	results := make(chan providerCountResult)
	workers := min(providerCountWorkers, len(cids))

	var wg sync.WaitGroup

	for range workers {
		wg.Go(func() {
			for cid := range jobs {
				count, err := t.counters.GetProviderCount(ctx, cid)

				select {
				case results <- providerCountResult{cid: cid, count: count, err: err}:
				case <-ctx.Done():
					return
				}
			}
		})
	}

	go func() {
		defer close(jobs)

		for _, cid := range cids {
			select {
			case jobs <- cid:
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	return results
}
