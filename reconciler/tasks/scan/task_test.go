// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package scan

import (
	"testing"

	"github.com/agntcy/dir/utils/scanner"
)

// TestNewTask_RunnerSet locks in the runner set NewTask builds. Live-endpoint
// scanning is a phase inside MCPRunner rather than a runner of its own, so the
// set here is unchanged by that work: a fourth "remote" entry appearing would
// mean the split runner had come back, along with the scanner-type gap it
// carried.
func TestNewTask_RunnerSet(t *testing.T) {
	t.Parallel()

	task, err := NewTask(Config{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantNames := []string{"mcp", "skill", "a2a"}
	if len(task.runners) != len(wantNames) {
		t.Fatalf("want %d runners, got %d", len(wantNames), len(task.runners))
	}

	for i, want := range wantNames {
		if got := task.runners[i].Name(); got != want {
			t.Errorf("runner %d: Name() = %q, want %q", i, got, want)
		}
	}
}

// TestNewTask_PassesEndpointScanConfig checks the two endpoint knobs reach the
// runner. They are security-relevant defaults: if the wiring silently dropped
// them, the reconciler would scan private-range endpoints regardless of
// configuration, and the config file would appear to work while doing nothing.
func TestNewTask_PassesEndpointScanConfig(t *testing.T) {
	t.Parallel()

	task, err := NewTask(Config{
		DisableEndpointScan:   true,
		AllowPrivateEndpoints: true,
	}, nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mcp, ok := task.runners[0].(*scanner.MCPRunner)
	if !ok {
		t.Fatalf("first runner should be the MCP runner, got %T", task.runners[0])
	}

	got := mcp.EndpointScanSettings()
	if !got.Disabled || !got.AllowPrivate {
		t.Errorf("config did not reach the runner: %+v, want both fields true", got)
	}
}
