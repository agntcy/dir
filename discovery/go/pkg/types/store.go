// Package types provides common types for the discovery service.
package types

import "context"

// StoreWriter is the interface for writing workload data to storage.
// Used by the discovery component to register/deregister workloads and metadata.
type StoreWriter interface {
	// RegisterWorkload writes a workload to storage.
	RegisterWorkload(ctx context.Context, workload *Workload) error

	// DeregisterWorkload removes a workload and its metadata from storage.
	DeregisterWorkload(ctx context.Context, workloadID string) error

	// SetMetadata writes processor metadata for a workload.
	SetMetadata(ctx context.Context, workloadID, processorKey string, data interface{}) error

	// ListWorkloadIDs returns all workload IDs from storage.
	ListWorkloadIDs(ctx context.Context) (map[string]struct{}, error)

	// Close closes the storage connection.
	Close() error
}

// StoreReader is the interface for reading workload data from storage.
// Used by the server component to query workloads and metadata.
type StoreReader interface {
	// Get returns a workload by an identifier.
	Get(id string) (*Workload, error)

	// List returns all workloads, optionally filtered.
	List(runtime RuntimeType, labelFilter map[string]string) []*Workload

	// FindReachable returns workloads reachable from a source workload.
	FindReachable(identifier string) (*ReachabilityResult, error)

	// Count returns the total number of workloads.
	Count() int

	// Close closes the storage connection.
	Close() error
}

// ReachabilityResult represents the result of a reachability query.
type ReachabilityResult struct {
	Caller    *Workload   `json:"caller"`
	Reachable []*Workload `json:"reachable"`
	Count     int         `json:"count"`
}
