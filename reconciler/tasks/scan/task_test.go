// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package scan

import (
	"testing"
	"time"

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

// The endpoint-scan knobs are asserted behaviorally over in utils/scanner
// (TestRun_DisableEndpointScan_SkipsEndpointPhase and the validateEndpointURL
// cases) rather than read back off the runner here. Exporting an accessor just
// so this file could assert the wiring would widen the scanner package's
// public API for the sake of one test.

// A record cannot change, so a failure it caused backs off to the TTL rather
// than the cap that exists to get a fixed deployment scanning again quickly.
func TestScanSchedule_RetryCapDependsOnFault(t *testing.T) {
	t.Parallel()

	task := &Task{config: Config{
		TTL: 7 * 24 * time.Hour,
		// The interval doubles as the backoff base.
		Interval: 6 * time.Hour,
	}}

	for _, tc := range []struct {
		name         string
		reason       scanner.FailureReason
		wantRetryMax time.Duration
	}{
		{"private repository", scanner.FailureSourceUnreachable, 7 * 24 * time.Hour},
		{"bad endpoint URL", scanner.FailureEndpointURLInvalid, 7 * 24 * time.Hour},
		{"endpoint down", scanner.FailureEndpointUnreachable, 7 * 24 * time.Hour},
		{"scanner binary missing", scanner.FailureScannerUnavailable, 24 * time.Hour},
		{"scanner timed out", scanner.FailureScannerTimeout, 24 * time.Hour},
		{"no LLM configured", scanner.FailureLLMNotConfigured, 24 * time.Hour},
		{"verdict reached", scanner.FailureNone, 24 * time.Hour},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := task.scanSchedule(tc.reason)
			if got.RetryMax != tc.wantRetryMax {
				t.Errorf("RetryMax = %v, want %v", got.RetryMax, tc.wantRetryMax)
			}

			// The other two must survive the override untouched.
			if got.FreshFor != 7*24*time.Hour {
				t.Errorf("FreshFor = %v, want 168h", got.FreshFor)
			}

			if got.RetryBase != 6*time.Hour {
				t.Errorf("RetryBase = %v, want the 6h interval", got.RetryBase)
			}
		})
	}
}
