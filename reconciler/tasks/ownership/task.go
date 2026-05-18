// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package ownership implements the reconciliation task for indexing ownership claims.
// It walks OwnershipClaimReferrerType referrers from the OCI store and ensures the
// owners table in the search database reflects the current referrer state.
package ownership

import (
	"context"
	"fmt"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	ownershipv1 "github.com/agntcy/dir/api/ownership/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
)

var logger = logging.Logger("reconciler/ownership")

// Task reconciles ownership claim referrers into the search database.
type Task struct {
	config Config
	db     types.OwnershipDatabaseAPI
	store  types.ReferrerStoreAPI
	search types.SearchDatabaseAPI
}

// NewTask creates a new ownership reconciliation task.
func NewTask(config Config, db types.OwnershipDatabaseAPI, search types.SearchDatabaseAPI, store types.ReferrerStoreAPI) (*Task, error) {
	return &Task{
		config: config,
		db:     db,
		store:  store,
		search: search,
	}, nil
}

// Name returns the task name.
func (t *Task) Name() string {
	return "ownership"
}

// Interval returns how often this task should run.
func (t *Task) Interval() time.Duration {
	return t.config.GetInterval()
}

// IsEnabled returns whether this task is enabled.
func (t *Task) IsEnabled() bool {
	return t.config.Enabled
}

// Run walks all records in the search DB, then for each record walks its ownership
// referrers in the OCI store and upserts them into the owners table.
func (t *Task) Run(ctx context.Context) error {
	logger.Debug("Running ownership reconciliation")

	// Get all record CIDs from the search database.
	cids, err := t.search.GetRecordCIDs()
	if err != nil {
		return fmt.Errorf("get record CIDs: %w", err)
	}

	if len(cids) == 0 {
		logger.Debug("No records to reconcile ownership for")

		return nil
	}

	logger.Info("Reconciling ownership claims", "record_count", len(cids))

	var indexed, failed int

	for _, cid := range cids {
		if err := t.reconcileRecord(ctx, cid); err != nil {
			logger.Warn("Failed to reconcile ownership for record", "cid", cid, "error", err)

			failed++

			continue
		}

		indexed++
	}

	logger.Info("Ownership reconciliation complete", "indexed", indexed, "failed", failed)

	return nil
}

// reconcileRecord walks ownership referrers for one record and upserts into the owners table.
func (t *Task) reconcileRecord(ctx context.Context, cid string) error {
	if err := t.store.WalkReferrers(ctx, cid, corev1.OwnershipClaimReferrerType, func(ref *corev1.RecordReferrer) error {
		claim := &ownershipv1.OwnershipClaim{}
		if err := claim.UnmarshalReferrer(ref); err != nil {
			logger.Warn("Failed to unmarshal ownership claim referrer", "cid", cid, "error", err)

			return nil // skip malformed referrers; continue walk
		}

		if claim.GetOwnerId() == "" {
			return nil
		}

		if err := t.db.AddOwner(cid, claim.GetOwnerId(), claim.GetClaimedAt()); err != nil {
			return fmt.Errorf("add owner %s: %w", claim.GetOwnerId(), err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("walk ownership referrers for %s: %w", cid, err)
	}

	return nil
}
