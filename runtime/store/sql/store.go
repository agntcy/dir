// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sql

import (
	"context"
	"errors"
	"fmt"
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
	prev, err := s.snapshot(ctx)
	if err != nil {
		return err
	}

	ticker := time.NewTicker(defaultWatchPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			//nolint:wrapcheck
			return ctx.Err()
		case <-ticker.C:
			current, snapErr := s.snapshot(ctx)
			if snapErr != nil {
				logger.Error("watch snapshot failed", "error", snapErr)

				continue
			}

			for id, currentPayload := range current {
				prevPayload, existed := prev[id]
				if existed && prevPayload == currentPayload {
					continue
				}

				workload, getErr := s.GetWorkload(ctx, id)
				if getErr != nil {
					logger.Error("failed to fetch workload during watch", "workload", id, "error", getErr)

					continue
				}

				handler(workload, false)
			}

			for id := range prev {
				if _, ok := current[id]; ok {
					continue
				}

				handler(&runtimev1.Workload{Id: id}, true)
			}

			prev = current
		}
	}
}

func (s *store) snapshot(ctx context.Context) (map[string]string, error) {
	var records []workloadRecord

	err := s.db.WithContext(ctx).Find(&records).Error
	if err != nil {
		return nil, fmt.Errorf("failed to create watch snapshot: %w", err)
	}

	snapshot := make(map[string]string, len(records))
	for _, record := range records {
		snapshot[record.ID] = string(record.Payload)
	}

	return snapshot, nil
}
