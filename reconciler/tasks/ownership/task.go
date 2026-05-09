// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package ownership implements the ownership claim reconciler task.
// It walks OwnershipClaim referrers from the store and keeps the DB owners table in sync.
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

// Task reconciles ownership claims from OCI referrers into the search DB.
type Task struct {
	config Config
	db     types.OwnershipDatabaseAPI
	store  types.ReferrerStoreAPI
	search types.SearchDatabaseAPI
}

// NewTask creates a new ownership reconciler task.
func NewTask(config Config, db types.OwnershipDatabaseAPI, store types.ReferrerStoreAPI, search types.SearchDatabaseAPI) (*Task, error) {
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

// Run walks all records and syncs their ownership referrers into the DB.
func (t *Task) Run(ctx context.Context) error {
	logger.Debug("Running ownership reconciliation")

	records, err := t.search.GetRecords()
	if err != nil {
		return fmt.Errorf("failed to get records: %w", err)
	}

	for _, record := range records {
		if err := t.reconcileRecord(ctx, record.GetCid()); err != nil {
			logger.Error("Failed to reconcile ownership for record", "cid", record.GetCid(), "error", err)
		}
	}

	return nil
}

func (t *Task) reconcileRecord(ctx context.Context, cid string) error {
	if err := t.store.WalkReferrers(ctx, cid, corev1.OwnershipClaimReferrerType, func(ref *corev1.RecordReferrer) error {
		claim := &ownershipv1.Claim{}
		if err := claim.UnmarshalReferrer(ref); err != nil {
			logger.Warn("Failed to unmarshal ownership claim referrer", "cid", cid, "error", err)

			return nil
		}

		if err := t.db.AddOwner(cid, claim.GetOwnerId(), claim.GetClaimedAt()); err != nil {
			return fmt.Errorf("add owner %s for %s: %w", claim.GetOwnerId(), cid, err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("walk ownership referrers for %s: %w", cid, err)
	}

	return nil
}
