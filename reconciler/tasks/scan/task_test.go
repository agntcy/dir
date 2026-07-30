// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package scan

import (
	"testing"
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

// The endpoint-scan knobs are asserted behaviorally over in utils/scanner
// (TestRun_DisableEndpointScan_SkipsEndpointPhase and the validateEndpointURL
// cases) rather than read back off the runner here. Exporting an accessor just
// so this file could assert the wiring would widen the scanner package's
// public API for the sake of one test.
