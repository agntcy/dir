// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sql

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	runtimev1 "github.com/agntcy/dir/runtime/api/runtime/v1"
	"github.com/agntcy/dir/runtime/store/types"
	"github.com/agntcy/dir/runtime/utils"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const defaultWatchPollInterval = 5 * time.Second

var logger = utils.NewLogger("store", "sql")

// store provides storage operations for gorm-backed storage.
type store struct {
	db *gorm.DB
}

// New creates a new store.
func New(db *gorm.DB) (types.Store, error) {
	if err := db.AutoMigrate(&workloadRecord{}); err != nil {
		return nil, fmt.Errorf("failed to migrate workload schema: %w", err)
	}

	return &store{db: db}, nil
}

// Close closes the connection.
func (s *store) Close() error {
	return nil
}

// RegisterWorkload stores a workload.
func (s *store) RegisterWorkload(ctx context.Context, workload *runtimev1.Workload) error {
	record, err := newWorkloadRecord(workload)
	if err != nil {
		return err
	}

	err = s.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{"payload", "updated_at"}),
		}).
		Create(record).Error
	if err != nil {
		return fmt.Errorf("failed to register workload: %w", err)
	}

	logger.Info("registered workload", "workload", workload.GetId())

	return nil
}

// DeregisterWorkload removes a workload.
func (s *store) DeregisterWorkload(ctx context.Context, workloadID string) error {
	if workloadID == "" {
		return fmt.Errorf("workload ID is empty")
	}

	result := s.db.WithContext(ctx).Where("id = ?", workloadID).Delete(&workloadRecord{})
	if result.Error != nil {
		return fmt.Errorf("failed to deregister workload: %w", result.Error)
	}

	logger.Info("deregistered workload", "workload", workloadID)

	return nil
}

// UpdateWorkload performs a full update of an existing workload in storage.
func (s *store) UpdateWorkload(ctx context.Context, workload *runtimev1.Workload) error {
	record, err := newWorkloadRecord(workload)
	if err != nil {
		return err
	}

	var existing workloadRecord

	err = s.db.WithContext(ctx).Where("id = ?", workload.GetId()).First(&existing).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Warn("workload not found for patch", "workload", workload.GetId())

			return nil
		}

		return fmt.Errorf("failed to get workload for update: %w", err)
	}

	err = s.db.WithContext(ctx).
		Model(&workloadRecord{}).
		Where("id = ?", workload.GetId()).
		Updates(map[string]any{"payload": record.Payload, "updated_at": time.Now().UTC()}).Error
	if err != nil {
		return fmt.Errorf("failed to update workload: %w", err)
	}

	logger.Info("patched workload", "workload", workload.GetId())

	return nil
}

// GetWorkload retrieves a workload by ID.
func (s *store) GetWorkload(ctx context.Context, workloadID string) (*runtimev1.Workload, error) {
	if workloadID == "" {
		return nil, fmt.Errorf("workload ID is empty")
	}

	var record workloadRecord

	err := s.db.WithContext(ctx).Where("id = ?", workloadID).First(&record).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("workload not found: %s", workloadID)
		}

		return nil, fmt.Errorf("failed to get workload: %w", err)
	}

	workload, err := record.ToRuntimeWorkload()
	if err != nil {
		return nil, err
	}

	return workload, nil
}

// ListWorkloadIDs returns all workload IDs in the store.
func (s *store) ListWorkloadIDs(ctx context.Context) (map[string]struct{}, error) {
	var ids []string

	err := s.db.WithContext(ctx).Model(&workloadRecord{}).Pluck("id", &ids).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list workload IDs: %w", err)
	}

	result := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		result[id] = struct{}{}
	}

	return result, nil
}

// ListWorkloads returns all workloads in the store.
func (s *store) ListWorkloads(ctx context.Context) ([]*runtimev1.Workload, error) {
	var records []workloadRecord

	err := s.db.WithContext(ctx).Find(&records).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list workloads: %w", err)
	}

	workloads := make([]*runtimev1.Workload, 0, len(records))
	for _, record := range records {
		workload, convErr := record.ToRuntimeWorkload()
		if convErr != nil {
			logger.Error("failed to parse workload", "workload", record.ID, "error", convErr)

			continue
		}

		workloads = append(workloads, workload)
	}

	return workloads, nil
}

// WatchWorkloads watches for workload changes.
func (s *store) WatchWorkloads(ctx context.Context, handler func(workload *runtimev1.Workload, deleted bool)) error {
	// Take an initial snapshot of the workloads in the store to compare against for changes.
	// We will use this to detect created, updated, and deleted workloads.
	prev, err := s.snapshot(ctx)
	if err != nil {
		return err
	}

	// Poll the store at regular intervals to detect changes.
	// We cannot rely on database triggers or notifications since they are not supported by SQL.
	ticker := time.NewTicker(defaultWatchPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			//nolint:wrapcheck
			return ctx.Err()
		case <-ticker.C:
			// Take a snapshot of the current workloads in the store. We will compare this with the previous snapshot to detect changes.
			current, snapErr := s.snapshot(ctx)
			if snapErr != nil {
				logger.Error("watch snapshot failed", "error", snapErr)

				continue
			}

			// If the workload exists in the current snapshot but is missing in the previous snapshot, it has been created.
			// If the workload exists in both snapshots but has different payloads, it has been updated.
			for id, currentWorkload := range current {
				prevWorkload, existed := prev[id]
				if existed && slices.Equal(prevWorkload.Payload, currentWorkload.Payload) {
					continue
				}

				// Fetch the workload from the store to pass to the handler. We cannot rely on the payload in the snapshot since it
				// may be outdated by the time we process it, and the handler may want to see the most up-to-date version of the workload.
				workload, getErr := s.GetWorkload(ctx, id)
				if getErr != nil {
					logger.Error("failed to fetch workload during watch", "workload", id, "error", getErr)

					continue
				}

				// Mark the workload as patched/fetched in the handler.
				handler(workload, false)
			}

			// If the workload existed in the previous snapshot but is missing in the current snapshot, it has been deleted.
			// We check for this after processing all current workloads to ensure we don't miss any updates that occur between snapshots.
			for id, prevWorkload := range prev {
				// Skip if the workload still exists
				if _, ok := current[id]; ok {
					continue
				}

				// Parse the workload from the previous snapshot to pass to the handler.
				// It cannot be fetched from the store since it has been deleted, but the handler may still want to know its details.
				workload, err := prevWorkload.ToRuntimeWorkload()
				if err != nil {
					logger.Error("failed to unmarshal deleted workload during watch", "workload", id, "error", err)

					continue
				}

				// Mark the workload as deleted in the handler.
				handler(workload, true)
			}

			// Update the previous snapshot to the current one for the next iteration.
			prev = current
		}
	}
}

// snapshot takes a snapshot of the current workloads in the store and returns a map of workload ID to payload.
func (s *store) snapshot(ctx context.Context) (map[string]*workloadRecord, error) {
	var records []*workloadRecord

	err := s.db.WithContext(ctx).Find(&records).Error
	if err != nil {
		return nil, fmt.Errorf("failed to create watch snapshot: %w", err)
	}

	snapshot := make(map[string]*workloadRecord, len(records))
	for _, record := range records {
		snapshot[record.ID] = record
	}

	return snapshot, nil
}
