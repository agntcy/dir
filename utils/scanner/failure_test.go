// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package scanner

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"testing"
	"time"
)

// --- FailureReason classification ---

func TestFailureReason_IsFailure(t *testing.T) {
	t.Parallel()

	cases := []struct {
		reason FailureReason
		want   bool
	}{
		{FailureNone, false},
		// Nothing declared is not a failure. Every MCP-only record hits
		// FailureArtifactMissing for the skill and a2a runners.
		{FailureSourceNotDeclared, false},
		{FailureEndpointNotDeclared, false},
		{FailureArtifactMissing, false},
		{FailureSourceUnreachable, true},
		{FailureEndpointURLInvalid, true},
		{FailureEndpointUnreachable, true},
		{FailureScannerTimeout, true},
		{FailureScannerCrashed, true},
		{FailureScannerUnavailable, true},
		{FailureScannerFailed, true},
		{FailureScannerUnparsable, true},
		{FailureLLMNotConfigured, true},
	}

	for _, tc := range cases {
		t.Run(string(tc.reason), func(t *testing.T) {
			t.Parallel()

			if got := tc.reason.IsFailure(); got != tc.want {
				t.Errorf("%q.IsFailure() = %v, want %v", tc.reason, got, tc.want)
			}
		})
	}
}

func TestFailureReason_IsPublisherFault(t *testing.T) {
	t.Parallel()

	cases := []struct {
		reason FailureReason
		want   bool
	}{
		{FailureSourceUnreachable, true},
		{FailureEndpointURLInvalid, true},
		{FailureEndpointUnreachable, true},
		// Our own infrastructure: an OOM-killed scanner or an expired
		// per-record deadline must never penalise the record.
		{FailureScannerTimeout, false},
		{FailureScannerCrashed, false},
		{FailureScannerUnavailable, false},
		{FailureScannerFailed, false},
		{FailureScannerUnparsable, false},
		{FailureLLMNotConfigured, false},
		{FailureNone, false},
		{FailureSourceNotDeclared, false},
	}

	for _, tc := range cases {
		t.Run(string(tc.reason), func(t *testing.T) {
			t.Parallel()

			if got := tc.reason.IsPublisherFault(); got != tc.want {
				t.Errorf("%q.IsPublisherFault() = %v, want %v", tc.reason, got, tc.want)
			}
		})
	}
}

// --- ClassifyError ---

func TestClassifyError_Nil(t *testing.T) {
	t.Parallel()

	if got := ClassifyError(context.Background(), nil); got != FailureNone {
		t.Errorf("nil error should classify as FailureNone, got %q", got)
	}
}

// exec.CommandContext SIGKILLs its child on deadline, so a timeout and a crash
// look alike from the error alone; ctx has to be consulted first.
func TestClassifyError_ExpiredContextBeatsSignalDeath(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()

	<-ctx.Done()

	killed := runForError(t, "kill -9 $$")

	if got := ClassifyError(ctx, killed); got != FailureScannerTimeout {
		t.Errorf("signal death under an expired context should be a timeout, got %q", got)
	}
}

func TestClassifyError_SignalDeathIsCrash(t *testing.T) {
	t.Parallel()

	err := runForError(t, "kill -9 $$")

	if got := ClassifyError(context.Background(), err); got != FailureScannerCrashed {
		t.Errorf("signal death should classify as a crash, got %q", got)
	}
}

func TestClassifyError_NonZeroExitIsScannerFailure(t *testing.T) {
	t.Parallel()

	err := runForError(t, "exit 3")

	if got := ClassifyError(context.Background(), err); got != FailureScannerFailed {
		t.Errorf("a plain non-zero exit should classify as scanner-failed, got %q", got)
	}
}

// The reason survives the fmt.Errorf wrapping every call site applies.
func TestClassifyError_UnwrapsWrappedExitError(t *testing.T) {
	t.Parallel()

	wrapped := fmt.Errorf("mcp-scanner exited with error: %w", runForError(t, "exit 3"))

	if got := ClassifyError(context.Background(), wrapped); got != FailureScannerFailed {
		t.Errorf("wrapped exit error should still classify, got %q", got)
	}
}

func TestClassifyError_MissingBinaryIsUnavailable(t *testing.T) {
	t.Parallel()

	err := exec.CommandContext(t.Context(), "dir-scanner-that-does-not-exist").Run()
	if err == nil {
		t.Fatal("expected running a nonexistent binary to fail")
	}

	if got := ClassifyError(context.Background(), err); got != FailureScannerUnavailable {
		t.Errorf("a missing binary should classify as unavailable, got %q", got)
	}
}

func TestClassifyError_UnrecognisedErrorIsScannerFailure(t *testing.T) {
	t.Parallel()

	if got := ClassifyError(context.Background(), errors.New("something else")); got != FailureScannerFailed {
		t.Errorf("unrecognised error should classify as scanner-failed, got %q", got)
	}
}

// classifyEndpointError attributes a clean non-zero exit to the endpoint
// rather than to our scanner, but leaves a dead process attributed to us.
func TestClassifyEndpointError(t *testing.T) {
	t.Parallel()

	if got := classifyEndpointError(context.Background(), runForError(t, "exit 3")); got != FailureEndpointUnreachable {
		t.Errorf("a scanner that ran and exited non-zero means the endpoint failed, got %q", got)
	}

	if got := classifyEndpointError(context.Background(), runForError(t, "kill -9 $$")); got != FailureScannerCrashed {
		t.Errorf("a dead scanner process stays our fault, got %q", got)
	}
}

// runForError returns the error from a shell snippet expected to fail, giving
// tests a genuine *exec.ExitError. os.ProcessState cannot be faked.
func runForError(t *testing.T, script string) error {
	t.Helper()

	err := exec.CommandContext(t.Context(), "sh", "-c", script).Run()
	if err == nil {
		t.Fatalf("expected %q to fail", script)
	}

	// Deliberately unwrapped: callers classify this error, and the wrapped
	// case has its own test.
	return err //nolint:wrapcheck
}

// --- pickFailureReason ---

func TestPickFailureReason_PrefersRealFailureOverNotApplicable(t *testing.T) {
	t.Parallel()

	// Taking the earliest reason regardless would report
	// "source-not-declared", which is not a failure, and the endpoint failure
	// would never be persisted.
	got := pickFailureReason([]*ScanResult{
		{Skipped: true, FailureReason: FailureSourceNotDeclared},
		{Skipped: true, FailureReason: FailureEndpointUnreachable},
	})

	if got != FailureEndpointUnreachable {
		t.Errorf("got %q, want %q", got, FailureEndpointUnreachable)
	}
}

func TestPickFailureReason_EarliestFailureWins(t *testing.T) {
	t.Parallel()

	// MCPRunner.Run passes the source phase first, which is how source
	// failures take precedence over endpoint ones.
	got := pickFailureReason([]*ScanResult{
		{Skipped: true, FailureReason: FailureSourceUnreachable},
		{Skipped: true, FailureReason: FailureEndpointUnreachable},
	})

	if got != FailureSourceUnreachable {
		t.Errorf("got %q, want %q", got, FailureSourceUnreachable)
	}
}

func TestPickFailureReason_FallsBackToNotApplicable(t *testing.T) {
	t.Parallel()

	got := pickFailureReason([]*ScanResult{
		nil,
		{Skipped: true, FailureReason: FailureSourceNotDeclared},
	})

	if got != FailureSourceNotDeclared {
		t.Errorf("got %q, want %q", got, FailureSourceNotDeclared)
	}
}

func TestPickFailureReason_NoneWhenNothingFailed(t *testing.T) {
	t.Parallel()

	if got := pickFailureReason([]*ScanResult{{Safe: true}, {Safe: true}}); got != FailureNone {
		t.Errorf("got %q, want empty", got)
	}
}

// --- merge: partial coverage ---

// The case the change exists for: a record whose source clone failed but whose
// endpoints scanned cleanly used to persist an ordinary passing report.
func TestMerge_OneSkippedOneScanned_IsPartial(t *testing.T) {
	t.Parallel()

	got := merge([]*ScanResult{
		{Skipped: true, SkippedReason: "git clone failed", FailureReason: FailureSourceUnreachable},
		{Safe: true},
	})

	if got.Skipped {
		t.Error("a phase produced a result, so the merge must not be skipped")
	}

	if !got.Partial {
		t.Error("a phase was skipped, so the merge must be partial")
	}

	if got.FailureReason != FailureSourceUnreachable {
		t.Errorf("FailureReason = %q, want %q", got.FailureReason, FailureSourceUnreachable)
	}

	if got.SkippedReason == "" {
		t.Error("detail text should survive onto a partial result")
	}

	if !got.Safe {
		t.Error("a partial scan reports the verdict the phases that ran produced")
	}
}

func TestMerge_AllScanned_IsNotPartial(t *testing.T) {
	t.Parallel()

	got := merge([]*ScanResult{{Safe: true}, {Safe: true}})

	if got.Partial {
		t.Error("nothing was skipped, so the merge must not be partial")
	}

	if got.FailureReason != FailureNone {
		t.Errorf("FailureReason = %q, want empty", got.FailureReason)
	}
}

// Reduced coverage has to survive nesting: the endpoint phase merges its
// per-endpoint sub-scans before MCPRunner.Run merges it with the source phase.
func TestMerge_NestedPartialPropagates(t *testing.T) {
	t.Parallel()

	endpointPhase := merge([]*ScanResult{
		{Safe: true},
		{Skipped: true, SkippedReason: "remote https://x: refused", FailureReason: FailureEndpointUnreachable},
	})

	if !endpointPhase.Partial {
		t.Fatal("inner merge should be partial")
	}

	got := merge([]*ScanResult{{Safe: true}, endpointPhase})

	if !got.Partial {
		t.Error("a partial input must keep the outer merge partial")
	}

	if got.FailureReason != FailureEndpointUnreachable {
		t.Errorf("FailureReason = %q, want %q", got.FailureReason, FailureEndpointUnreachable)
	}

	if got.SkippedReason == "" {
		t.Error("nested detail text should be carried up")
	}
}

func TestMerge_AllSkipped_CarriesReasonAndIsNotPartial(t *testing.T) {
	t.Parallel()

	got := merge([]*ScanResult{
		{Skipped: true, SkippedReason: "git clone failed", FailureReason: FailureSourceUnreachable},
		{Skipped: true, SkippedReason: "no remote MCP endpoint found", FailureReason: FailureEndpointNotDeclared},
	})

	if !got.Skipped {
		t.Error("every phase skipped, so the merge is skipped")
	}

	if got.Partial {
		t.Error("nothing ran, so the merge is not partial")
	}

	if got.Safe {
		t.Error("a skipped merge must not be marked safe")
	}

	if got.FailureReason != FailureSourceUnreachable {
		t.Errorf("FailureReason = %q, want the real failure %q", got.FailureReason, FailureSourceUnreachable)
	}
}

// A skip with no detail text must still make the merge partial, hence the skip
// count tracked separately from the reason strings.
func TestMerge_SkipWithoutDetailIsStillPartial(t *testing.T) {
	t.Parallel()

	got := merge([]*ScanResult{{Skipped: true}, {Safe: true}})

	if !got.Partial {
		t.Error("a detail-less skip should still produce a partial merge")
	}
}
