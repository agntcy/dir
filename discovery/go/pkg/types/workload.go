package types

import (
	"context"
	"encoding/json"
)

// WorkloadType represents the type of workload.
type WorkloadType string

const (
	WorkloadTypeContainer WorkloadType = "container"
	WorkloadTypePod       WorkloadType = "pod"
	WorkloadTypeService   WorkloadType = "service"
)

// WorkloadProcessor is the interface for metadata extraction logic for workloads.
type WorkloadProcessor interface {
	// Name returns the processor name (used as metadata key).
	Name() string

	// ShouldProcess returns whether the processor should process the workload.
	ShouldProcess(workload *Workload) bool

	// Process extracts metadata from the workload.
	Process(ctx context.Context, workload *Workload) (interface{}, error)
}

// Workload represents a unified workload across all runtimes.
type Workload struct {
	// Identity
	ID       string `json:"id"`
	Name     string `json:"name"`
	Hostname string `json:"hostname"`

	// Runtime info
	Runtime      RuntimeType       `json:"runtime"`
	WorkloadType WorkloadType      `json:"workload_type"`
	Labels       map[string]string `json:"labels,omitempty"`
	Annotations  map[string]string `json:"annotations,omitempty"`

	// Network
	Addresses       []string `json:"addresses"`
	IsolationGroups []string `json:"isolation_groups"`
	Ports           []string `json:"ports"`

	// Scraped metadata (populated async by inspector)
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ToJSON serializes the workload to JSON.
func (w *Workload) ToJSON() ([]byte, error) {
	return json.Marshal(w)
}

// FromJSON deserializes a workload from JSON.
func FromJSON(data []byte) (*Workload, error) {
	var w Workload
	if err := json.Unmarshal(data, &w); err != nil {
		return nil, err
	}
	return &w, nil
}

// ReachabilityResult represents the result of a reachability query.
type ReachabilityResult struct {
	Caller    *Workload   `json:"caller"`
	Reachable []*Workload `json:"reachable"`
	Count     int         `json:"count"`
}
