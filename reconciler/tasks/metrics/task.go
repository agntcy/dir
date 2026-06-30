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
	"time"

	reconcilercfg "github.com/agntcy/dir/config/reconciler"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
)

var logger = logging.Logger("reconciler/metrics")

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
	config   reconcilercfg.Metrics
	db       types.UsageMetricsDatabaseAPI
	search   types.SearchDatabaseAPI
	counters ProviderCounterAPI
}

// NewTask creates a new usage-metrics reconciliation task.
func NewTask(config reconcilercfg.Metrics, db types.DatabaseAPI, counters ProviderCounterAPI) (*Task, error) {
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

	for _, cid := range cids {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled: %w", ctx.Err())
		default:
		}

		count, err := t.counters.GetProviderCount(ctx, cid)
		if err != nil {
			logger.Warn("Failed to get provider count", "cid", cid, "error", err)

			failed++

			continue
		}

		if count < 0 {
			count = 0
		}

		if err := t.db.SetProviderCount(cid, uint32(count)); err != nil {
			logger.Warn("Failed to set provider count", "cid", cid, "count", count, "error", err)

			failed++

			continue
		}

		updated++
	}

	logger.Info("Provider count refresh complete", "updated", updated, "failed", failed)

	return nil
}
