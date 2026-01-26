// Package processor provides metadata extraction processors.
package processor

import (
	"context"

	"github.com/agntcy/dir/discovery/pkg/models"
)

// Processor is the interface for metadata extraction processors.
type Processor interface {
	// Name returns the processor name (used as metadata key).
	Name() string

	// Enabled returns whether the processor is enabled.
	Enabled() bool

	// ShouldProcess returns whether the processor should process the workload.
	ShouldProcess(workload *models.Workload) bool

	// Process extracts metadata from the workload.
	Process(ctx context.Context, workload *models.Workload) (interface{}, error)
}
