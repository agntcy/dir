// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package scanner

import (
	"context"
	"slices"
	"strings"
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

// --- NewRemoteRunner / Name ---

func TestNewRemoteRunner_DefaultCLIPath(t *testing.T) {
	t.Parallel()

	r := NewRemoteRunner(RemoteConfig{})
	if r.cfg.CLIPath != DefaultMCPCLIPath {
		t.Errorf("empty CLIPath should default to %q, got %q", DefaultMCPCLIPath, r.cfg.CLIPath)
	}
}

func TestNewRemoteRunner_CustomCLIPath(t *testing.T) {
	t.Parallel()

	r := NewRemoteRunner(RemoteConfig{CLIPath: "/usr/local/bin/mcp-scanner"})
	if r.cfg.CLIPath != "/usr/local/bin/mcp-scanner" {
		t.Errorf("custom CLIPath should be preserved, got %q", r.cfg.CLIPath)
	}
}

func TestRemoteRunner_Name(t *testing.T) {
	t.Parallel()

	r := NewRemoteRunner(RemoteConfig{})
	if got := r.Name(); got != "remote" {
		t.Errorf("Name() = %q, want %q", got, "remote")
	}
}

// --- Run / no endpoints ---

func TestRemoteRunner_Run_NilRecord_Skipped(t *testing.T) {
	t.Parallel()

	r := NewRemoteRunner(RemoteConfig{})

	got, err := r.Run(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !got.Skipped || got.SkippedReason == "" {
		t.Errorf("nil record should be skipped with a reason: %+v", got)
	}
}

func TestRemoteRunner_Run_NoConnections_Skipped(t *testing.T) {
	t.Parallel()

	r := NewRemoteRunner(RemoteConfig{})

	st, err := structpb.NewStruct(map[string]any{"schema_version": "1.0.0"})
	if err != nil {
		t.Fatalf("structpb.NewStruct: %v", err)
	}

	got, err := r.Run(context.Background(), &corev1.Record{Data: st})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !got.Skipped {
		t.Error("record with no remote-capable connection should be skipped")
	}
}

// --- extractConnectionURLs ---

func TestExtractConnectionURLs_Nil(t *testing.T) {
	t.Parallel()

	if got := extractConnectionURLs(nil); got != nil {
		t.Errorf("nil struct should return nil, got %v", got)
	}
}

func TestExtractConnectionURLs_NoConnections(t *testing.T) {
	t.Parallel()

	data, _ := structpb.NewStruct(map[string]any{"other_data": "x"})
	if got := extractConnectionURLs(data); got != nil {
		t.Errorf("data without a connections field should return nil, got %v", got)
	}
}

func TestExtractConnectionURLs_StdioOnly_Excluded(t *testing.T) {
	t.Parallel()

	data, _ := structpb.NewStruct(map[string]any{
		"connections": []any{
			map[string]any{"type": "stdio", "command": "python server.py"},
		},
	})

	if got := extractConnectionURLs(data); len(got) != 0 {
		t.Errorf("stdio-only connection should produce no URLs, got %v", got)
	}
}

func TestExtractConnectionURLs_SSEFound(t *testing.T) {
	t.Parallel()

	want := "https://example.com/sse"

	data, _ := structpb.NewStruct(map[string]any{
		"connections": []any{
			map[string]any{"type": "sse", "url": want},
		},
	})

	got := extractConnectionURLs(data)
	if len(got) != 1 || got[0] != want {
		t.Errorf("want [%q], got %v", want, got)
	}
}

func TestExtractConnectionURLs_StreamableHTTPFound(t *testing.T) {
	t.Parallel()

	want := "https://example.com/mcp"

	data, _ := structpb.NewStruct(map[string]any{
		"connections": []any{
			map[string]any{"type": "streamable-http", "url": want},
		},
	})

	got := extractConnectionURLs(data)
	if len(got) != 1 || got[0] != want {
		t.Errorf("want [%q], got %v", want, got)
	}
}

func TestExtractConnectionURLs_MixedTransports_OnlyRemoteReturned(t *testing.T) {
	t.Parallel()

	want := "https://example.com/mcp"

	data, _ := structpb.NewStruct(map[string]any{
		"connections": []any{
			map[string]any{"type": "stdio", "command": "python server.py"},
			map[string]any{"type": "streamable-http", "url": want},
		},
	})

	got := extractConnectionURLs(data)
	if len(got) != 1 || got[0] != want {
		t.Errorf("want only the remote-capable URL [%q], got %v", want, got)
	}
}

func TestExtractConnectionURLs_RemoteTypeWithoutURL_Excluded(t *testing.T) {
	t.Parallel()

	data, _ := structpb.NewStruct(map[string]any{
		"connections": []any{
			map[string]any{"type": "sse"},
		},
	})

	if got := extractConnectionURLs(data); len(got) != 0 {
		t.Errorf("sse connection without a url should produce no URLs, got %v", got)
	}
}

func TestExtractConnectionURLs_MultipleRemoteConnections(t *testing.T) {
	t.Parallel()

	data, _ := structpb.NewStruct(map[string]any{
		"connections": []any{
			map[string]any{"type": "sse", "url": "https://a.example.com/sse"},
			map[string]any{"type": "streamable-http", "url": "https://b.example.com/mcp"},
		},
	})

	got := extractConnectionURLs(data)
	if len(got) != 2 {
		t.Fatalf("want 2 URLs, got %d: %v", len(got), got)
	}
}

func TestExtractConnectionURLs_NonStructConnection_Skipped(t *testing.T) {
	t.Parallel()

	// The connections list is untyped (structpb.ListValue), so a malformed
	// entry that isn't an object at all (here, a bare string) must be
	// skipped via the `if conn == nil { continue }` guard rather than
	// panicking on a nil GetStructValue(), while a valid entry later in the
	// same list is still picked up.
	want := "https://valid.example.com/mcp"

	data, err := structpb.NewStruct(map[string]any{
		"connections": []any{
			"not-an-object",
			map[string]any{"type": "streamable-http", "url": want},
		},
	})
	if err != nil {
		t.Fatalf("structpb.NewStruct: %v", err)
	}

	got := extractConnectionURLs(data)
	if len(got) != 1 || got[0] != want {
		t.Errorf("want [%q], got %v", want, got)
	}
}

// --- tagFindings ---

func TestTagFindings_Empty(t *testing.T) {
	t.Parallel()

	if got := tagFindings("remote", "https://example.com", nil); got != nil {
		t.Errorf("empty findings should return nil, got %v", got)
	}
}

func TestTagFindings_PrefixesMessage(t *testing.T) {
	t.Parallel()

	in := []Finding{{Severity: SeverityError, Message: "prompt injection detected"}}

	got := tagFindings("prompts", "https://example.com/mcp", in)
	if len(got) != 1 {
		t.Fatalf("want 1 finding, got %d", len(got))
	}

	want := "[prompts https://example.com/mcp] prompt injection detected"
	if got[0].Message != want {
		t.Errorf("want %q, got %q", want, got[0].Message)
	}

	if got[0].Severity != SeverityError {
		t.Errorf("severity should be preserved, got %q", got[0].Severity)
	}
}

func TestTagFindings_PreservesOrderAndCount(t *testing.T) {
	t.Parallel()

	in := []Finding{
		{Severity: SeverityError, Message: "first"},
		{Severity: SeverityWarning, Message: "second"},
	}

	got := tagFindings("resources", "https://example.com", in)
	if len(got) != len(in) {
		t.Fatalf("want %d findings, got %d", len(in), len(got))
	}

	for i, f := range got {
		if f.Severity != in[i].Severity {
			t.Errorf("finding %d: severity changed, want %q got %q", i, in[i].Severity, f.Severity)
		}
	}
}

// --- extractRemoteEndpoints ---
//
// The per-connection logic is unit tested directly above via
// extractConnectionURLs against hand-built structpb.Struct values (mirroring
// extractSubfolder's tests in mcp_test.go). The decode-and-delegate wrapper
// itself is exercised both indirectly (via the Run() tests above) and
// directly below, including its two error returns (decode failure, non-v1
// schema) and the module-walking loop, which none of the tests above touch
// since they use records with no modules at all.

func TestExtractRemoteEndpoints_NilRecord(t *testing.T) {
	t.Parallel()

	if got := extractRemoteEndpoints(nil); got != nil {
		t.Errorf("nil record should return nil, got %v", got)
	}
}

func TestExtractRemoteEndpoints_DecodeError_ReturnsNil(t *testing.T) {
	t.Parallel()

	// No "schema_version" field at all: record.Decode() fails inside
	// decoder.GetRecordSchemaVersion, so extractRemoteEndpoints must hit its
	// `if err != nil { return nil }` branch rather than panicking or the
	// nil-decoded-record propagating further.
	data, err := structpb.NewStruct(map[string]any{"name": "no-schema-version"})
	if err != nil {
		t.Fatalf("structpb.NewStruct: %v", err)
	}

	got := extractRemoteEndpoints(&corev1.Record{Data: data})
	if got != nil {
		t.Errorf("record with undecodable data should return nil, got %v", got)
	}
}

func TestExtractRemoteEndpoints_NonV1Schema_ReturnsNil(t *testing.T) {
	t.Parallel()

	// schema_version 0.7.x decodes successfully as OASF v1alpha1, so
	// decoded.HasV1() is false and extractRemoteEndpoints must return nil
	// without attempting to read v1-shaped modules.
	data, err := structpb.NewStruct(map[string]any{"schema_version": "0.7.0"})
	if err != nil {
		t.Fatalf("structpb.NewStruct: %v", err)
	}

	got := extractRemoteEndpoints(&corev1.Record{Data: data})
	if got != nil {
		t.Errorf("non-v1 record should return nil, got %v", got)
	}
}

func TestExtractRemoteEndpoints_WalksModulesAndCollectsURLs(t *testing.T) {
	t.Parallel()

	// A real v1 record with two modules, only one of which carries a
	// remote-capable MCP connection, so the loop body in
	// extractRemoteEndpoints (append(urls, extractConnectionURLs(...)...))
	// actually runs across more than zero modules.
	data, err := structpb.NewStruct(map[string]any{
		"schema_version": "1.0.0",
		"modules": []any{
			map[string]any{
				"name": "core/other",
				"data": map[string]any{"unrelated": "x"},
			},
			map[string]any{
				"name": "core/mcp",
				"data": map[string]any{
					"connections": []any{
						map[string]any{"type": "sse", "url": "https://example.com/sse"},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("structpb.NewStruct: %v", err)
	}

	got := extractRemoteEndpoints(&corev1.Record{Data: data})
	if len(got) != 1 || got[0] != "https://example.com/sse" {
		t.Errorf("want [%q], got %v", "https://example.com/sse", got)
	}
}

// --- runMCPScannerRemote ---
//
// These exercise the real exec.CommandContext invocation using
// testdata/fakecli as a stand-in mcp-scanner binary (see fakecli_test.go).

func TestRunMCPScannerRemote_Success_ReturnsStdout(t *testing.T) {
	t.Parallel()

	cli := fakeCLIPath(t)

	out, err := runMCPScannerRemote(context.Background(), cli, "remote", "https://ok.example.com/mcp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(string(out), `"tool_name":"remote"`) {
		t.Errorf("stdout should contain the fake CLI's canned finding tagged with the subcommand, got %q", out)
	}
}

func TestRunMCPScannerRemote_ExecFailure_WrapsStderr(t *testing.T) {
	t.Parallel()

	cli := fakeCLIPath(t)

	_, err := runMCPScannerRemote(context.Background(), cli, "remote", "https://fail-exec.example.com/mcp")
	if err == nil {
		t.Fatal("want an error when the mcp-scanner process exits non-zero")
	}

	if !strings.Contains(err.Error(), "simulated exec failure") {
		t.Errorf("error should surface the process's stderr, got %q", err)
	}

	if !strings.Contains(err.Error(), "exited with error") {
		t.Errorf("error should describe the failure as a non-zero exit, got %q", err)
	}
}

// --- RemoteRunner.runOne ---

func TestRemoteRunner_RunOne_Success_TagsAndSetsAnalyzer(t *testing.T) {
	t.Parallel()

	r := NewRemoteRunner(RemoteConfig{CLIPath: fakeCLIPath(t)})

	got := r.runOne(context.Background(), "prompts", "https://ok.example.com/mcp")

	if got.Skipped {
		t.Fatalf("want a non-skipped result, got skipped: %s", got.SkippedReason)
	}

	if len(got.Findings) != 1 {
		t.Fatalf("want 1 finding, got %d: %+v", len(got.Findings), got.Findings)
	}

	wantPrefix := "[prompts https://ok.example.com/mcp]"
	if !strings.HasPrefix(got.Findings[0].Message, wantPrefix) {
		t.Errorf("finding message should be tagged with subcommand+url, want prefix %q, got %q", wantPrefix, got.Findings[0].Message)
	}

	if len(got.Analyzers) != 1 || got.Analyzers[0] != "prompts" {
		t.Errorf("want Analyzers=[prompts], got %v", got.Analyzers)
	}
}

func TestRemoteRunner_RunOne_ExecFailure_SkippedNotError(t *testing.T) {
	t.Parallel()

	r := NewRemoteRunner(RemoteConfig{CLIPath: fakeCLIPath(t)})

	got := r.runOne(context.Background(), "remote", "https://fail-exec.example.com/mcp")

	if !got.Skipped {
		t.Fatal("an unreachable endpoint must be skipped, not surfaced as a hard error")
	}

	for _, want := range []string{"remote", "fail-exec.example.com", "simulated exec failure"} {
		if !strings.Contains(got.SkippedReason, want) {
			t.Errorf("SkippedReason should contain %q for traceability, got %q", want, got.SkippedReason)
		}
	}
}

func TestRemoteRunner_RunOne_UnparsableOutput_SkippedNotError(t *testing.T) {
	t.Parallel()

	r := NewRemoteRunner(RemoteConfig{CLIPath: fakeCLIPath(t)})

	got := r.runOne(context.Background(), "resources", "https://bad-json.example.com/mcp")

	if !got.Skipped {
		t.Fatal("unparsable mcp-scanner output must be skipped, not surfaced as a hard error")
	}

	if !strings.Contains(got.SkippedReason, "unparsable output") {
		t.Errorf("SkippedReason should mention unparsable output, got %q", got.SkippedReason)
	}
}

func TestRemoteRunner_RunOne_EmptySafeOutput_NoFindings(t *testing.T) {
	t.Parallel()

	r := NewRemoteRunner(RemoteConfig{CLIPath: fakeCLIPath(t)})

	got := r.runOne(context.Background(), "instructions", "https://empty-safe.example.com/mcp")

	if got.Skipped {
		t.Fatalf("want a non-skipped, safe result, got skipped: %s", got.SkippedReason)
	}

	if !got.Safe || len(got.Findings) != 0 {
		t.Errorf("empty mcp-scanner output should produce Safe=true with no findings: %+v", got)
	}

	if len(got.Analyzers) != 1 || got.Analyzers[0] != "instructions" {
		t.Errorf("want Analyzers=[instructions], got %v", got.Analyzers)
	}
}

// --- RemoteRunner.Run (end to end, via testdata/fakecli) ---

// remoteRecordWithEndpoints builds a v1 record whose sole MCP module
// declares one remote-capable ("streamable-http") connection per URL given.
func remoteRecordWithEndpoints(t *testing.T, urls ...string) *corev1.Record {
	t.Helper()

	conns := make([]any, 0, len(urls))
	for _, u := range urls {
		conns = append(conns, map[string]any{"type": "streamable-http", "url": u})
	}

	data, err := structpb.NewStruct(map[string]any{
		"schema_version": "1.0.0",
		"modules": []any{
			map[string]any{
				"name": "core/mcp",
				"data": map[string]any{"connections": conns},
			},
		},
	})
	if err != nil {
		t.Fatalf("structpb.NewStruct: %v", err)
	}

	return &corev1.Record{Data: data}
}

func TestRemoteRunner_Run_MergesFindingsAcrossEndpointsAndSubcommands(t *testing.T) {
	t.Parallel()

	r := NewRemoteRunner(RemoteConfig{CLIPath: fakeCLIPath(t)})
	rec := remoteRecordWithEndpoints(t, "https://a.example.com/mcp", "https://b.example.com/mcp")

	got, err := r.Run(context.Background(), rec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Skipped {
		t.Fatalf("want a non-skipped result when endpoints succeed, got skipped: %s", got.SkippedReason)
	}

	// 2 endpoints * 4 subcommands (remote, prompts, resources, instructions),
	// each producing exactly 1 tagged finding from the fake CLI.
	const wantFindings = 2 * 4

	if len(got.Findings) != wantFindings {
		t.Errorf("want %d merged findings, got %d: %+v", wantFindings, len(got.Findings), got.Findings)
	}

	if got.Safe {
		t.Error("merged result should be Safe=false: every sub-scan reported an unsafe finding")
	}

	wantAnalyzers := []string{"remote", "prompts", "resources", "instructions"}
	for _, a := range wantAnalyzers {
		if !slices.Contains(got.Analyzers, a) {
			t.Errorf("merged Analyzers missing %q, got %v", a, got.Analyzers)
		}
	}
}

func TestRemoteRunner_Run_AllEndpointsUnreachable_SkippedNotError(t *testing.T) {
	t.Parallel()

	r := NewRemoteRunner(RemoteConfig{CLIPath: fakeCLIPath(t)})
	rec := remoteRecordWithEndpoints(t, "https://fail-exec.example.com/mcp")

	got, err := r.Run(context.Background(), rec)
	if err != nil {
		t.Fatalf("network failures must not be surfaced as a hard error, got: %v", err)
	}

	if !got.Skipped {
		t.Fatal("want the merged result to be Skipped when every sub-scan failed to reach its endpoint")
	}

	if !strings.Contains(got.SkippedReason, "fail-exec.example.com") {
		t.Errorf("SkippedReason should reference the unreachable endpoint, got %q", got.SkippedReason)
	}
}

func TestRemoteRunner_Run_MixedReachability_KeepsSuccessfulFindings(t *testing.T) {
	t.Parallel()

	r := NewRemoteRunner(RemoteConfig{CLIPath: fakeCLIPath(t)})
	rec := remoteRecordWithEndpoints(t, "https://ok.example.com/mcp", "https://fail-exec.example.com/mcp")

	got, err := r.Run(context.Background(), rec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Skipped {
		t.Fatal("one reachable endpoint should be enough for the merged result to not be Skipped")
	}

	// Only the reachable endpoint's 4 subcommands contribute findings; the
	// unreachable one is dropped as a skip, not merged in as an error.
	const wantFindings = 4
	if len(got.Findings) != wantFindings {
		t.Errorf("want %d findings (from the reachable endpoint only), got %d", wantFindings, len(got.Findings))
	}
}

func TestRemoteRunner_Run_UnparsableEndpoint_SkippedNotError(t *testing.T) {
	t.Parallel()

	r := NewRemoteRunner(RemoteConfig{CLIPath: fakeCLIPath(t)})
	rec := remoteRecordWithEndpoints(t, "https://bad-json.example.com/mcp")

	got, err := r.Run(context.Background(), rec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !got.Skipped {
		t.Fatal("want the merged result to be Skipped when every sub-scan produced unparsable output")
	}

	if !strings.Contains(got.SkippedReason, "unparsable output") {
		t.Errorf("SkippedReason should mention unparsable output, got %q", got.SkippedReason)
	}
}
