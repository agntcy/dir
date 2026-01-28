package oasf

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/agntcy/dir/discovery/pkg/types"
)

// processor probes workloads for OASF data.
type processor struct {
	timeout  time.Duration
	client   *http.Client
	labelKey string
}

// NewProcessor creates a new OASF processor.
func NewProcessor(cfg Config) types.WorkloadProcessor {
	return &processor{
		timeout: cfg.Timeout,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
		labelKey: cfg.LabelKey,
	}
}

// Name returns the processor name.
func (p *processor) Name() string {
	return "oasf"
}

// ShouldProcess returns whether to process the workload.
func (p *processor) ShouldProcess(workload *types.Workload) bool {
	// If workload does not have a label key, skip it
	if _, ok := workload.Labels[p.labelKey]; !ok {
		return false
	}

	return true
}

// Process probes health endpoints on the workload.
func (p *processor) Process(ctx context.Context, workload *types.Workload) (interface{}, error) {
	return nil, fmt.Errorf("not implemented")
}
