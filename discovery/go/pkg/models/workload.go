// Package models provides unified data models for multi-runtime service discovery.
package models

import (
	"encoding/json"
	"time"
)

// Runtime represents the container runtime type.
type Runtime string

const (
	RuntimeDocker     Runtime = "docker"
	RuntimeContainerd Runtime = "containerd"
	RuntimeKubernetes Runtime = "kubernetes"
)

// WorkloadType represents the type of workload.
type WorkloadType string

const (
	WorkloadTypeContainer WorkloadType = "container"
	WorkloadTypePod       WorkloadType = "pod"
	WorkloadTypeService   WorkloadType = "service"
)

// EventType represents types of workload events.
type EventType string

const (
	EventTypeAdded          EventType = "added"
	EventTypeModified       EventType = "modified"
	EventTypeDeleted        EventType = "deleted"
	EventTypePaused         EventType = "paused"
	EventTypeNetworkChanged EventType = "network_changed"
)

// Workload represents a unified workload across all runtimes.
type Workload struct {
	// Identity
	ID       string `json:"id"`
	Name     string `json:"name"`
	Hostname string `json:"hostname"`

	// Runtime info
	Runtime      Runtime      `json:"runtime"`
	WorkloadType WorkloadType `json:"workload_type"`

	// Location
	Node      string `json:"node,omitempty"`
	Namespace string `json:"namespace,omitempty"`

	// Network
	Addresses       []string `json:"addresses"`
	IsolationGroups []string `json:"isolation_groups"`
	Ports           []string `json:"ports"`

	// Discovery metadata
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`

	// Scraped metadata (populated async by inspector)
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// Internal tracking
	Registrar string `json:"registrar,omitempty"`

	// Global naming (Phase 1)
	Domain      string `json:"domain,omitempty"`
	Environment string `json:"environment,omitempty"`
	Version     string `json:"version,omitempty"`
	Protocol    string `json:"protocol,omitempty"`
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

// GlobalName returns a globally unique name for the workload.
func (w *Workload) GlobalName() string {
	domain := w.Domain
	if domain == "" {
		domain = "cluster.local"
	}
	env := w.Environment
	if env == "" {
		env = "default"
	}
	ns := w.Namespace
	if ns == "" {
		ns = "default"
	}
	return domain + "/" + env + "/" + ns + "/" + w.Name
}

// WorkloadEvent represents a workload change event.
type WorkloadEvent struct {
	Type     EventType
	Workload *Workload
}

// ReachabilityResult represents the result of a reachability query.
type ReachabilityResult struct {
	Caller    *Workload   `json:"caller"`
	Reachable []*Workload `json:"reachable"`
	Count     int         `json:"count"`
}

// HealthResult represents a health check result.
type HealthResult struct {
	Healthy        bool    `json:"healthy"`
	Endpoint       string  `json:"endpoint,omitempty"`
	StatusCode     int     `json:"status_code,omitempty"`
	ResponseTimeMs float64 `json:"response_time_ms,omitempty"`
	Error          string  `json:"error,omitempty"`
	CheckedAt      string  `json:"checked_at"`
}

// NewHealthResult creates a new health result with current timestamp.
func NewHealthResult(healthy bool) *HealthResult {
	return &HealthResult{
		Healthy:   healthy,
		CheckedAt: time.Now().UTC().Format(time.RFC3339),
	}
}

// OpenAPIResult represents an OpenAPI discovery result.
type OpenAPIResult struct {
	Available    bool     `json:"available"`
	Endpoint     string   `json:"endpoint,omitempty"`
	Title        string   `json:"title,omitempty"`
	Version      string   `json:"version,omitempty"`
	Description  string   `json:"description,omitempty"`
	Paths        []string `json:"paths,omitempty"`
	PathCount    int      `json:"path_count,omitempty"`
	Error        string   `json:"error,omitempty"`
	DiscoveredAt string   `json:"discovered_at"`
}

// NewOpenAPIResult creates a new OpenAPI result with current timestamp.
func NewOpenAPIResult() *OpenAPIResult {
	return &OpenAPIResult{
		Available:    false,
		DiscoveredAt: time.Now().UTC().Format(time.RFC3339),
	}
}
