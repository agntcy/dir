// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package scanner

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

// These tests exercise A2ARunner.Run's happy path and runA2AScanner without a
// real a2a-scanner binary. A small stand-in CLI (testdata/fakescanner) is built
// into a temp dir and its path is used as CLIPath; the fake's behaviour is
// driven through FAKE_A2A_* env vars, which reach the child because
// buildA2AScannerEnv forwards the parent environment. Building the fake from
// testdata (rather than re-execing the test binary) keeps its path out of the
// os.Args taint chain, so the runner's exec call is not flagged by gosec.

// fakeScannerBin builds testdata/fakescanner into the test's temp dir and
// returns the binary path.
func fakeScannerBin(t *testing.T) string {
	t.Helper()

	bin := filepath.Join(t.TempDir(), "fakescanner")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}

	out, err := exec.CommandContext(t.Context(), "go", "build", "-o", bin, "./testdata/fakescanner").CombinedOutput() //nolint:gosec
	if err != nil {
		t.Fatalf("build fake scanner: %v: %s", err, out)
	}

	return bin
}

// a2aRecordWithCard builds a record whose a2a module carries a card_data object,
// so extractA2ACard returns a card and Run proceeds to invoke the CLI.
func a2aRecordWithCard(t *testing.T) *corev1.Record {
	t.Helper()

	data, err := structpb.NewStruct(map[string]any{
		"schema_version": "1.0.0",
		"name":           "burger_seller_agent",
		"modules": []any{
			map[string]any{
				"name": "integration/a2a",
				"data": map[string]any{
					"card_data": map[string]any{
						"name":    "burger_seller_agent",
						"version": "1.0.0",
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("structpb.NewStruct: %v", err)
	}

	return &corev1.Record{Data: data}
}

func TestA2ARunner_Run_Safe(t *testing.T) {
	// Not parallel: uses t.Setenv.
	bin := fakeScannerBin(t)
	// No FAKE_A2A_OUTPUT set: the fake emits its default zero-finding payload.

	r := NewA2ARunner(A2AConfig{CLIPath: bin})

	got, err := r.Run(t.Context(), a2aRecordWithCard(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Skipped {
		t.Fatalf("record with a card should not be skipped: %+v", got)
	}

	if !got.Safe {
		t.Errorf("zero-finding output should produce Safe=true, got %+v", got)
	}

	if len(got.Findings) != 0 {
		t.Errorf("safe scan should have no findings, got %d", len(got.Findings))
	}

	wantAnalyzers := []string{"heuristic", "spec", "yara"}
	if strings.Join(got.Analyzers, ",") != strings.Join(wantAnalyzers, ",") {
		t.Errorf("Analyzers = %v, want %v", got.Analyzers, wantAnalyzers)
	}

	if got.Version != "9.9.9-test" {
		t.Errorf("Version = %q, want %q (parsed from --version)", got.Version, "9.9.9-test")
	}
}

func TestA2ARunner_Run_UnsafeWithFindings(t *testing.T) {
	// Not parallel: uses t.Setenv.
	t.Setenv("FAKE_A2A_OUTPUT", `{"status":"completed","findings":[`+
		`{"threat_name":"unsafe_delegation","scanner_category":"delegation","severity":"HIGH","description":"unsafe delegation chain"}],`+
		`"total_findings":1,"high_severity_count":1}`)

	r := NewA2ARunner(A2AConfig{CLIPath: fakeScannerBin(t)})

	got, err := r.Run(t.Context(), a2aRecordWithCard(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Safe {
		t.Error("output with findings should produce Safe=false")
	}

	if len(got.Findings) != 1 {
		t.Fatalf("want 1 finding, got %d", len(got.Findings))
	}

	if got.Findings[0].Severity != SeverityError {
		t.Errorf("HIGH severity should map to SeverityError, got %q", got.Findings[0].Severity)
	}
}

func TestA2ARunner_Run_ScannerExitError(t *testing.T) {
	// Not parallel: uses t.Setenv.
	t.Setenv("FAKE_A2A_FAIL", "1")

	r := NewA2ARunner(A2AConfig{CLIPath: fakeScannerBin(t)})

	_, err := r.Run(t.Context(), a2aRecordWithCard(t))
	if err == nil {
		t.Fatal("a non-zero scanner exit should surface an error")
	}

	if !strings.Contains(err.Error(), "a2a-scanner") {
		t.Errorf("error should be wrapped with a2a-scanner context, got %v", err)
	}
}

func TestA2ARunner_Run_MissingOutputFile(t *testing.T) {
	// Not parallel: uses t.Setenv.
	t.Setenv("FAKE_A2A_NO_FILE", "1")

	r := NewA2ARunner(A2AConfig{CLIPath: fakeScannerBin(t)})

	_, err := r.Run(t.Context(), a2aRecordWithCard(t))
	if err == nil {
		t.Fatal("a scanner that writes no output file should surface a read error")
	}
}

func TestA2ARunner_Run_InvalidScannerOutput(t *testing.T) {
	// Not parallel: uses t.Setenv.
	t.Setenv("FAKE_A2A_OUTPUT", "not valid json")

	r := NewA2ARunner(A2AConfig{CLIPath: fakeScannerBin(t)})

	_, err := r.Run(t.Context(), a2aRecordWithCard(t))
	if err == nil {
		t.Fatal("unparseable scanner output should surface an error")
	}
}

// --- runA2AScanner directly ---

func TestRunA2AScanner_WritesOutput(t *testing.T) {
	// Not parallel: uses t.Setenv.
	t.Setenv("FAKE_A2A_OUTPUT", `{"status":"completed","findings":[],"total_findings":0}`)

	bin := fakeScannerBin(t)
	dir := t.TempDir()
	cardPath := filepath.Join(dir, "card.json")
	outputPath := filepath.Join(dir, "out.json")

	if err := os.WriteFile(cardPath, []byte(`{"name":"x"}`), 0o600); err != nil {
		t.Fatalf("write card: %v", err)
	}

	if err := runA2AScanner(t.Context(), bin, cardPath, outputPath); err != nil {
		t.Fatalf("runA2AScanner returned error: %v", err)
	}

	body, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("output file not written: %v", err)
	}

	if !strings.Contains(string(body), "total_findings") {
		t.Errorf("unexpected output body: %s", body)
	}
}

func TestRunA2AScanner_ExitError(t *testing.T) {
	// Not parallel: uses t.Setenv.
	t.Setenv("FAKE_A2A_FAIL", "1")

	bin := fakeScannerBin(t)
	dir := t.TempDir()
	cardPath := filepath.Join(dir, "card.json")

	if err := os.WriteFile(cardPath, []byte(`{}`), 0o600); err != nil {
		t.Fatalf("write card: %v", err)
	}

	if err := runA2AScanner(t.Context(), bin, cardPath, filepath.Join(dir, "out.json")); err == nil {
		t.Error("non-zero CLI exit should return an error")
	}
}
