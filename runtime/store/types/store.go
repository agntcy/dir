// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"context"

	runtimev1 "github.com/agntcy/dir/runtime/api/runtime/v1"
)

type StoreType string

// StoreWriter is the interface for writing workload data to storage.
// Used by the discovery component to register/deregister workloads.
type StoreWriter interface {
	// RegisterWorkload writes a workload to storage.
	RegisterWorkload(ctx context.Context, workload *runtimev1.Workload) error

	// DeregisterWorkload removes a workload from storage.
	DeregisterWorkload(ctx context.Context, workloadID string) error

	// UpdateWorkload performs a full update of an existing workload in storage.
	UpdateWorkload(ctx context.Context, workload *runtimev1.Workload) error

	// Close closes the storage connection.
	Close() error
}

// StoreReader is the interface for reading workload data from storage.
// Used by the server component to rebuild database state.
type StoreReader interface {
	// ListWorkloadIDs returns all workload IDs from storage.
	ListWorkloadIDs(ctx context.Context) (map[string]struct{}, error)

	// ListWorkloads returns all workloads from storage.
	ListWorkloads(ctx context.Context) ([]*runtimev1.Workload, error)

	// WatchWorkloads watches for workload changes.
	WatchWorkloads(ctx context.Context, handler func(workload *runtimev1.Workload, deleted bool)) error

	// Close closes the storage connection.
	Close() error
}

type Store interface {
	StoreWriter
	StoreReader
}
