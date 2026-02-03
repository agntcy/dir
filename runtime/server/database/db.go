// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package database

import (
	"context"
	"fmt"
	"regexp"

	v1 "github.com/agntcy/dir/runtime/api/runtime/v1"
	"github.com/agntcy/dir/runtime/store"
	storeconfig "github.com/agntcy/dir/runtime/store/config"
	storetypes "github.com/agntcy/dir/runtime/store/types"
)

type Database struct {
	store storetypes.StoreReader
}

func NewDatabase(cfg storeconfig.Config) (*Database, error) {
	store, err := store.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create store: %w", err)
	}

	return &Database{
		store: store,
	}, nil
}

func (d *Database) Close() error {
	if err := d.store.Close(); err != nil {
		return fmt.Errorf("failed to close database store: %w", err)
	}

	return nil
}

func (d *Database) Count(ctx context.Context) int {
	ids, err := d.store.ListWorkloadIDs(ctx)
	if err != nil {
		return 0
	}

	return len(ids)
}

func (d *Database) Get(ctx context.Context, id string) (*v1.Workload, error) {
	// Fetch workload from store
	workload, err := d.store.GetWorkload(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get workload by ID: %w", err)
	}

	return workload, nil
}

func (d *Database) List(ctx context.Context, labelFilter map[string]string) ([]*v1.Workload, error) {
	// Fetch all workloads
	result, err := d.store.ListWorkloads(ctx)
	if err != nil {
		return []*v1.Workload{}, fmt.Errorf("failed to list workloads: %w", err)
	}

	// Filter workloads by labels
	var filtered []*v1.Workload

	for _, workload := range result {
		if matchesLabelFilter(workload, labelFilter) {
			filtered = append(filtered, workload)
		}
	}

	return filtered, nil
}

// matchesLabelFilter checks if a workload matches the provided label filters.
// Label values support regular expression syntax.
func matchesLabelFilter(workload *v1.Workload, filters map[string]string) bool {
	if len(filters) == 0 {
		return true
	}

	for key, pattern := range filters {
		value, exists := workload.GetLabels()[key]
		if !exists {
			return false
		}

		// Try to match as regex
		matched, err := regexp.MatchString(pattern, value)
		if err != nil {
			// If pattern is invalid, fall back to exact match
			if value != pattern {
				return false
			}
		} else if !matched {
			return false
		}
	}

	return true
}
