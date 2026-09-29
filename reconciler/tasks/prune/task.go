// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package prune implements the prune reconciler task.
// It deletes records that match configured criteria (trusted status, minimum
// scan severity, and age). Defaults match the dirctl prune-untrusted CronJob
// plus an age floor of 168h.
package prune

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

var logger = logging.Logger("reconciler/prune")

// recordIndex is the search-index surface this task needs.
type recordIndex interface {
	GetRecordCIDs(opts ...types.FilterOption) ([]string, error)
	RemoveRecord(cid string) error
}

// contentStore is the object-store surface this task needs.
type contentStore interface {
	Delete(ctx context.Context, ref *corev1.RecordRef) error
}

// Task implements the prune reconciler task.
type Task struct {
	config Config
	db     recordIndex
	store  contentStore
}

// NewTask creates a new prune task.
func NewTask(config Config, db recordIndex, store contentStore) (*Task, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Task{
		config: config,
		db:     db,
		store:  store,
	}, nil
}

// Name returns the task name.
func (t *Task) Name() string {
	return "prune"
}

// Interval returns how often this task should run.
func (t *Task) Interval() time.Duration {
	return t.config.GetInterval()
}

// IsEnabled returns whether this task is enabled.
func (t *Task) IsEnabled() bool {
	return t.config.Enabled
}

// Run finds records matching Criteria and deletes each from the store and
// the search index.
func (t *Task) Run(ctx context.Context) error {
	logger.Debug("Running prune")

	olderThan := t.config.GetOlderThan()
	indexedBefore := time.Now().Add(-olderThan)

	cids, err := t.db.GetRecordCIDs(
		types.WithTrusted(t.config.Criteria.Trusted),
		types.WithScanSeverities(t.config.GetMinSeverity()),
		types.WithIndexedBefore(indexedBefore),
		types.WithLimit(t.config.GetLimit()),
	)
	if err != nil {
		return fmt.Errorf("get records: %w", err)
	}

	if len(cids) == 0 {
		logger.Info("No records to prune")

		return nil
	}

	if t.config.DryRun {
		logger.Info("Dry run: not pruning records",
			"count", len(cids),
			"trusted", t.config.Criteria.Trusted,
			"min_severity", t.config.GetMinSeverity(),
			"older_than", olderThan,
			"cids", cids,
		)

		return nil
	}

	logger.Info("Pruning records",
		"count", len(cids),
		"trusted", t.config.Criteria.Trusted,
		"min_severity", t.config.GetMinSeverity(),
		"older_than", olderThan,
	)

	var succeeded, failed int

	for _, cid := range cids {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("prune canceled: %w", err)
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

	logger.Info("Prune complete", "succeeded", succeeded, "failed", failed)

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
