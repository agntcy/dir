// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"context"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"testing"
	"time"

	runtimev1 "github.com/agntcy/dir/runtime/api/runtime/v1"
)

func TestAdapter_ListWorkloads_All(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()

	runtimeAdapter, err := NewAdapter(Config{})
	if err != nil {
		t.Fatalf("NewAdapter() error = %v", err)
	}

	workloads, err := runtimeAdapter.ListWorkloads(ctx)
	if err != nil {
		t.Fatalf("ListWorkloads() error = %v", err)
	}

	if len(workloads) == 0 {
		t.Fatal("ListWorkloads() returned no workloads")
	}

	targetPID := strconv.Itoa(os.Getpid())

	var found *runtimev1.Workload

	// Must find self in the list of workloads
	for _, workload := range workloads {
		if workload.GetId() == targetPID {
			found = workload

			break
		}
	}

	if found == nil {
		t.Fatalf("failed to find workload for current process pid %s", targetPID)
	}

	if found.GetRuntime() != string(RuntimeType) {
		t.Fatalf("workload runtime = %q, want %q", found.GetRuntime(), string(RuntimeType))
	}

	wantType := runtimev1.WorkloadType_WORKLOAD_TYPE_PROCESS.GetName()
	if found.GetType() != wantType {
		t.Fatalf("workload type = %q, want %q", found.GetType(), wantType)
	}

	if found.GetName() == "" {
		t.Fatal("workload name is empty")
	}
}

func TestAdapter_ListWorkloads_Single(t *testing.T) {
	t.Parallel()

	// Start Python server processes in the background for discovery
	cmd := exec.CommandContext(t.Context(), "sh", "-c", `
	__AGNTCY_DISCOVERY_TESTS__=true \
	python3 -m http.server 0
	`)
	if err := cmd.Start(); err != nil {
		t.Errorf("failed to run background sleep command: %s: %v", cmd.ProcessState.String(), err)
	}

	// Add cleanup to kill the process after the test
	defer func() {
		if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
			t.Logf("failed to signal background sleep command: %v", err)
		}

		if err := cmd.Process.Kill(); err != nil {
			t.Logf("failed to kill background sleep command: %v", err)
		}
	}()

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	runtimeAdapter, err := NewAdapter(Config{
		EnvSelector: "__AGNTCY_DISCOVERY_TESTS__=true",
	})
	if err != nil {
		t.Fatalf("NewAdapter() error = %v", err)
	}

	workloads, err := runtimeAdapter.ListWorkloads(ctx)
	if err != nil {
		t.Fatalf("ListWorkloads() error = %v", err)
	}

	if len(workloads) == 0 {
		t.Fatal("ListWorkloads() returned no workloads")
	}

	targetPID := strconv.Itoa(cmd.Process.Pid)

	var found *runtimev1.Workload

	// Must find created process in the list of workloads
	for _, workload := range workloads {
		if workload.GetId() == targetPID {
			found = workload

			break
		}
	}

	if found == nil {
		t.Fatalf("failed to find workload for current process pid %s", targetPID)
	}
}
