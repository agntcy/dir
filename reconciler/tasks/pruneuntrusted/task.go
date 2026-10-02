// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package pruneuntrusted implements the prune-untrusted reconciler task.
// It deletes records that are not signature-trusted and whose highest scan
// severity meets or exceeds a configured threshold (MEDIUM by default),
// matching the dirctl prune-untrusted CronJob.
package pruneuntrusted

import (
	"context"
	"fmt"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var logger = logging.Logger("reconciler/prune-untrusted")

// recordIndex is the search-index surface this task needs.
type recordIndex interface {
	GetRecordCIDs(opts ...types.FilterOption) ([]string, error)
	RemoveRecord(cid string) error
}

// contentStore is the object-store surface this task needs.
type contentStore interface {
	Delete(ctx context.Context, ref *corev1.RecordRef) error
}

// Task implements the prune-untrusted reconciler task.
type Task struct {
	config Config
	db     recordIndex
	store  contentStore
}

// NewTask creates a new prune-untrusted task.
func NewTask(config Config, db recordIndex, store contentStore) (*Task, error) {
	return &Task{
		config: config,
		db:     db,
		store:  store,
	}, nil
}

// Name returns the task name.
func (t *Task) Name() string {
	return "prune-untrusted"
}

// Interval returns how often this task should run.
func (t *Task) Interval() time.Duration {
	return t.config.GetInterval()
}

// IsEnabled returns whether this task is enabled.
func (t *Task) IsEnabled() bool {
	return t.config.Enabled
}

// Run finds untrusted records at or above the scan-severity threshold and
// deletes each from the store and the search index.
func (t *Task) Run(ctx context.Context) error {
	logger.Debug("Running prune-untrusted")

	cids, err := t.db.GetRecordCIDs(
		types.WithTrusted(false),
		types.WithScanSeverities(t.config.GetScanSeverity()),
	)
	if err != nil {
		return fmt.Errorf("get untrusted records: %w", err)
	}

	if len(cids) == 0 {
		logger.Info("No untrusted records to prune")

		return nil
	}

	logger.Info("Pruning untrusted records", "count", len(cids), "scan_severity", t.config.GetScanSeverity())

	var succeeded, failed int

	for _, cid := range cids {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("prune-untrusted canceled: %w", err)
		}

		if cid == "" {
			continue
		}

		if err := t.prune(ctx, cid); err != nil {
			logger.Warn("Failed to prune record", "cid", cid, "error", err)

			failed++

			continue
		}

		succeeded++
	}

	logger.Info("Prune-untrusted complete", "succeeded", succeeded, "failed", failed)

	return nil
}

func (t *Task) prune(ctx context.Context, cid string) error {
	recordCtx, cancel := context.WithTimeout(ctx, t.config.GetRecordTimeout())
	defer cancel()

	err := t.store.Delete(recordCtx, &corev1.RecordRef{Cid: cid})
	if err != nil && status.Code(err) != codes.NotFound {
		return fmt.Errorf("delete record from store: %w", err)
	}

	// Search index is secondary — storage is the source of truth.
	if err := t.db.RemoveRecord(cid); err != nil {
		logger.Error("Failed to remove record from search index", "error", err, "cid", cid)
	}

	logger.Info("Record pruned", "cid", cid)

	return nil
}
