// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	runtimev1 "github.com/agntcy/dir/runtime/api/runtime/v1"
	"github.com/agntcy/dir/runtime/discovery/types"
	gprocess "github.com/shirou/gopsutil/v4/process"
)

// errWorkloadIgnored is returned when a workload is ignored due to discovery selectors.
var errWorkloadIgnored = fmt.Errorf("workload ignored by discovery selectors")

// adapter implements the RuntimeAdapter interface for local process discovery.
type adapter struct {
	hostname  string
	selectors []ProcessSelector
}

// NewAdapter creates a new adapter.
func NewAdapter(cfg Config, selectors ...ProcessSelector) (types.RuntimeAdapter, error) {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "localhost"
	}

	return &adapter{
		hostname: hostname,
		selectors: append(
			selectors,                         // Add passed selectors
			ProcessSelectorNonSystem,          // Add non-system process selector
			ProcessSelectorDiscoveryFunc(cfg), // Add config-based discovery selector
		),
	}, nil
}

// Type returns the runtime type.
func (a *adapter) Type() types.RuntimeType {
	return RuntimeType
}

// Close closes the adapter.
func (a *adapter) Close() error {
	return nil
}

// TODO: paralellize this once fetching is reliable.
func (k *adapter) ListWorkloads(ctx context.Context) ([]*runtimev1.Workload, error) {
	// Get all process IDs on the system
	pids, err := gprocess.PidsWithContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list processes: %w", err)
	}

	var workloads []*runtimev1.Workload

	for _, pid := range pids {
		workload, err := k.getWorkloadFromPID(ctx, pid)
		if errors.Is(err, errWorkloadIgnored) {
			logger.Debug("ignoring workload from PID %d", pid)

			continue
		}

		if err != nil {
			logger.Error("error processing PID %d: %v", pid, err)

			continue
		}

		workloads = append(workloads, workload)
	}

	return workloads, nil
}

// WatchEvents is a no-op for local runtime. Local discovery currently supports
// snapshot listing only via ListWorkloads.
func (k *adapter) WatchEvents(ctx context.Context, eventChan chan<- *types.RuntimeEvent) error {
	panic("WatchEvents is not implemented for local runtime")
}

// getWorkloadFromPID builds a Workload object for the given PID by fetching process
// details and associated metadata.
//
// It converts process information into a Workload format, including labels and annotations for key process attributes.
// It does not include environment variables to avoid potential security risks.
// Discovery selector is applied for each process.
func (k *adapter) getWorkloadFromPID(ctx context.Context, pid int32) (*runtimev1.Workload, error) {
	// Fetch process details
	process, err := GetProcess(ctx, pid)
	if err != nil {
		return nil, fmt.Errorf("failed to get process details for PID %d: %w", pid, err)
	}

	// Apply discovery selectors to determine if this process should be included in the discovery results.
	for _, selector := range k.selectors {
		if allowed, err := selector.Apply(process); err != nil {
			return nil, fmt.Errorf("failed to apply discovery selectors for PID %d: %w", pid, err)
		} else if !allowed {
			return nil, errWorkloadIgnored
		}
	}

	// Map process details to a Workload object.
	return &runtimev1.Workload{
		Id:       fmt.Sprintf("%d", pid),
		Name:     process.Name,
		Hostname: k.hostname,
		Runtime:  string(RuntimeType),
		Type:     runtimev1.WorkloadType_WORKLOAD_TYPE_PROCESS.GetName(),
		Labels:   make(map[string]string),
		Annotations: map[string]string{
			"process.pid":       fmt.Sprintf("%d", process.Pid),
			"process.ppid":      fmt.Sprintf("%d", process.ParentPid),
			"process.exe":       process.Exe,
			"process.cmdline":   process.Cmdline,
			"process.username":  process.Username,
			"process.createdAt": process.CreatedAt,
			"process.pipes":     strings.Join(process.Pipes, ","),
			"process.sockets":   strings.Join(process.Sockets, ","),
		},
		Addresses: process.Addresses,
		Ports: func() []string {
			var ports []string
			for _, port := range process.Ports {
				ports = append(ports, fmt.Sprintf("%d", port))
			}

			return ports
		}(),
		IsolationGroups: []string{
			"user:" + process.Username,
		},
	}, nil
}
