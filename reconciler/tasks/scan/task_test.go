// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package scan

import "testing"

// TestNewTask_WiresUpRemoteRunner locks in the runner set NewTask builds,
// specifically that RemoteRunner is now included alongside MCPRunner and
// SkillRunner (this PR's change) rather than being silently dropped.
func TestNewTask_WiresUpRemoteRunner(t *testing.T) {
	t.Parallel()

	task, err := NewTask(Config{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantNames := []string{"mcp", "remote", "skill"}
	if len(task.runners) != len(wantNames) {
		t.Fatalf("want %d runners, got %d", len(wantNames), len(task.runners))
	}

	for i, want := range wantNames {
		if got := task.runners[i].Name(); got != want {
			t.Errorf("runner %d: Name() = %q, want %q", i, got, want)
		}
	}
}
