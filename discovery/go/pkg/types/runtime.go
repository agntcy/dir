package types

import (
	"context"
)

// Runtime represents the container runtime type.
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
	Workload *Workload
}

// RuntimeAdapter is the interface for runtime adapters.
type RuntimeAdapter interface {
	// Type returns the type of the runtime.
	Type() RuntimeType

	// Close closes the adapter and releases resources.
	Close() error

	// ListWorkloads returns all discoverable workloads.
	ListWorkloads(ctx context.Context) ([]*Workload, error)

	// WatchEvents watches for workload events and sends them to the channel.
	WatchEvents(ctx context.Context, events chan<- *RuntimeEvent) error
}
