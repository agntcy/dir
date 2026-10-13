// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package prune implements the prune reconciler task.
// It deletes records that match configured criteria (signature status, given
// directly or through trusted, minimum scan severity, and age). Defaults follow
// the dirctl prune-untrusted CronJob plus an age floor of 168h, except that a
// signed record without a verification result is never deleted.
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

	cids, err := t.db.GetRecordCIDs(t.filters(indexedBefore, t.config.GetSignatureStatuses())...)
	if err != nil {
		return fmt.Errorf("get records: %w", err)
	}

	if len(cids) == 0 {
		logger.Info("No records to prune")

		return nil
	}

	if t.config.DryRun {
		logger.Info("Dry run: not pruning records",
			append(t.criteriaLogArgs(olderThan),
				"count", len(cids),
				"count_by_signature_status", t.countBySignatureStatus(indexedBefore),
				"cids", cids,
			)...,
		)

		return nil
	}

	logger.Info("Pruning records", append(t.criteriaLogArgs(olderThan), "count", len(cids))...)

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

// filters returns the search filters for the configured criteria and the given
// signature statuses, falling back to the configured ones when none are given.
func (t *Task) filters(indexedBefore time.Time, signatureStatuses []string) []types.FilterOption {
	// An empty list is not "no signature filter": handleFilterOptions drops the
	// predicate entirely, which would select every record matching the
	// remaining criteria. Same hazard Validate guards against for min_severity.
	if len(signatureStatuses) == 0 {
		signatureStatuses = t.config.GetSignatureStatuses()
	}

	return []types.FilterOption{
		types.WithSignatureStatuses(signatureStatuses...),
		types.WithScanSeverities(t.config.GetMinSeverity()),
		types.WithIndexedBefore(indexedBefore),
		types.WithLimit(t.config.GetLimit()),
	}
}

// criteriaLogArgs returns the criteria as log key/value pairs: the signature
// statuses in effect, plus trusted when they were derived from it.
func (t *Task) criteriaLogArgs(olderThan time.Duration) []any {
	var args []any
	if !t.config.UsesSignatureStatuses() {
		args = append(args, "trusted", t.config.GetTrusted())
	}

	return append(args,
		"signature_statuses", t.config.GetSignatureStatuses(),
		"min_severity", t.config.GetMinSeverity(),
		"older_than", olderThan,
	)
}

// countBySignatureStatus counts the matching records per signature status in
// effect, so a dry run shows why records would be pruned. Counts are capped by
// Limit.
func (t *Task) countBySignatureStatus(indexedBefore time.Time) map[string]int {
	statuses := t.config.GetSignatureStatuses()
	counts := make(map[string]int, len(statuses))

	for _, s := range statuses {
		cids, err := t.db.GetRecordCIDs(t.filters(indexedBefore, []string{s})...)
		if err != nil {
			logger.Warn("Failed to count records by signature status", "signature_status", s, "error", err)

			continue
		}

		counts[s] = len(cids)
	}

	return counts
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
