// Package runtime provides runtime adapters for container orchestrators.
package runtime

import (
	"context"

	"github.com/agntcy/dir/discovery/pkg/models"
)

// Adapter is the interface for runtime adapters.
type Adapter interface {
	// ListWorkloads returns all discoverable workloads.
	ListWorkloads(ctx context.Context) ([]*models.Workload, error)

	// WatchEvents watches for workload events and sends them to the channel.
	WatchEvents(ctx context.Context, events chan<- *models.WorkloadEvent) error

	// Close closes the adapter and releases resources.
	Close() error
}
