// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"context"

	runtimev1 "github.com/agntcy/dir/runtime/api/runtime/v1"
)

// RuntimeType represents the type of runtime environment.
type RuntimeType string

// RuntimeEventType represents types of workload events.
type RuntimeEventType string

const (
	RuntimeEventTypeAdded    RuntimeEventType = "added"
	RuntimeEventTypeModified RuntimeEventType = "modified"
	RuntimeEventTypeDeleted  RuntimeEventType = "deleted"
	RuntimeEventTypePaused   RuntimeEventType = "paused"
)

// RuntimeEvent represents a workload change event.
type RuntimeEvent struct {
	Type     RuntimeEventType
	Workload *runtimev1.Workload
}

// RuntimeAdapter is the interface for runtime adapters.
type RuntimeAdapter interface {
	// Type returns the type of the runtime.
	Type() RuntimeType

	// Close closes the adapter and releases resources.
	Close() error

	// ListWorkloads returns all discoverable workloads.
	ListWorkloads(ctx context.Context) ([]*runtimev1.Workload, error)

	// WatchEvents watches for workload events and sends them to the channel.
	WatchEvents(ctx context.Context, events chan<- *RuntimeEvent) error
}
