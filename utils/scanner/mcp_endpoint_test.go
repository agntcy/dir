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

// --- endpoint phase / no endpoints ---

func TestRunEndpointScan_NilRecord_Skipped(t *testing.T) {
	t.Parallel()

	r := NewMCPRunner(MCPConfig{})

	got := r.runEndpointScan(context.Background(), nil)
	if !got.Skipped || got.SkippedReason == "" {
		t.Errorf("nil record should be skipped with a reason: %+v", got)
	}
}

func TestRunEndpointScan_NoConnections_Skipped(t *testing.T) {
	t.Parallel()

	r := NewMCPRunner(MCPConfig{})

	st, err := structpb.NewStruct(map[string]any{"schema_version": "1.0.0"})
	if err != nil {
		t.Fatalf("structpb.NewStruct: %v", err)
	}

	got := r.runEndpointScan(context.Background(), &corev1.Record{Data: st})
	if !got.Skipped {
		t.Error("record with no remote-capable connection should be skipped")
	}
}

// TestRun_DisableEndpointScan_SkipsEndpointPhase pins the toggle: with the
// endpoint phase off, a record whose endpoints WOULD produce findings comes
// back with none.
//
// The record has to be v1 and the CLI has to be the fake scanner, or the
// assertions pass for the wrong reason: extractEndpoints bails early on a
// non-v1 record, so the phase would find nothing to scan whether the toggle
// worked or not. With this fixture, deleting the toggle produces 4 findings
// and fails the test.
func TestRun_DisableEndpointScan_SkipsEndpointPhase(t *testing.T) {
	t.Parallel()

	r := NewMCPRunner(MCPConfig{CLIPath: fakeCLIPath(t), DisableEndpointScan: true})
	rec := recordWithEndpoints(t, "https://ok.example.com/mcp")

	got, err := r.Run(context.Background(), rec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got.Findings) != 0 {
		t.Errorf("endpoint phase is disabled, so there should be no findings, got %d: %+v", len(got.Findings), got.Findings)
	}

	if len(got.Analyzers) != 0 {
		t.Errorf("endpoint phase is disabled, so no endpoint analyzers should be reported, got %v", got.Analyzers)
	}

	if !got.Skipped {
		t.Fatalf("with no source locator and the endpoint phase off, the result should be skipped: %+v", got)
	}

	if !strings.Contains(got.SkippedReason, "no source-code locator") {
		t.Errorf("expected the source-phase skip reason, got %q", got.SkippedReason)
	}
}

// The counterpart: with the toggle off, the same record does produce endpoint
// findings. Together these two fail if the toggle is deleted in either
// direction.
func TestRun_EndpointScanEnabled_ProducesFindings(t *testing.T) {
	t.Parallel()

	r := NewMCPRunner(MCPConfig{CLIPath: fakeCLIPath(t)})
	rec := recordWithEndpoints(t, "https://ok.example.com/mcp")

	got, err := r.Run(context.Background(), rec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 1 endpoint x 4 subcommands, one finding each from the fake CLI.
	const wantFindings = 4

	if len(got.Findings) != wantFindings {
		t.Errorf("want %d findings with the endpoint phase on, got %d: %+v", wantFindings, len(got.Findings), got.Findings)
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

// --- extractEndpoints ---
//
// The per-connection logic is unit tested directly above via
// extractConnectionURLs against hand-built structpb.Struct values (mirroring
// extractSubfolder's tests in mcp_test.go). The decode-and-delegate wrapper
// itself is exercised both indirectly (via the Run() tests above) and
// directly below, including its two error returns (decode failure, non-v1
// schema) and the module-walking loop, which none of the tests above touch
// since they use records with no modules at all.

func TestExtractEndpoints_NilRecord(t *testing.T) {
	t.Parallel()

	if got := extractEndpoints(nil); got != nil {
		t.Errorf("nil record should return nil, got %v", got)
	}
}

func TestExtractEndpoints_DecodeError_ReturnsNil(t *testing.T) {
	t.Parallel()

	// No "schema_version" field at all: record.Decode() fails inside
	// decoder.GetRecordSchemaVersion, so extractEndpoints must hit its
	// `if err != nil { return nil }` branch rather than panicking or the
	// nil-decoded-record propagating further.
	data, err := structpb.NewStruct(map[string]any{"name": "no-schema-version"})
	if err != nil {
		t.Fatalf("structpb.NewStruct: %v", err)
	}

	got := extractEndpoints(&corev1.Record{Data: data})
	if got != nil {
		t.Errorf("record with undecodable data should return nil, got %v", got)
	}
}

func TestExtractEndpoints_NonV1Schema_ReturnsNil(t *testing.T) {
	t.Parallel()

	// schema_version 0.7.x decodes successfully as OASF v1alpha1, so
	// decoded.HasV1() is false and extractEndpoints must return nil
	// without attempting to read v1-shaped modules.
	data, err := structpb.NewStruct(map[string]any{"schema_version": "0.7.0"})
	if err != nil {
		t.Fatalf("structpb.NewStruct: %v", err)
	}

	got := extractEndpoints(&corev1.Record{Data: data})
	if got != nil {
		t.Errorf("non-v1 record should return nil, got %v", got)
	}
}

func TestExtractEndpoints_WalksModulesAndCollectsURLs(t *testing.T) {
	t.Parallel()

	// A real v1 record with two modules, only one of which carries a
	// remote-capable MCP connection, so the loop body in
	// extractEndpoints (append(urls, extractConnectionURLs(...)...))
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

	got := extractEndpoints(&corev1.Record{Data: data})
	if len(got) != 1 || got[0] != "https://example.com/sse" {
		t.Errorf("want [%q], got %v", "https://example.com/sse", got)
	}
}

// --- runMCPScannerEndpoint ---
//
// These exercise the real exec.CommandContext invocation using
// testdata/fakecli as a stand-in mcp-scanner binary (see fakecli_test.go).

func TestRunMCPScannerEndpoint_Success_ReturnsStdout(t *testing.T) {
	t.Parallel()

	cli := fakeCLIPath(t)

	out, err := runMCPScannerEndpoint(context.Background(), cli, "remote", "https://ok.example.com/mcp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(string(out), `"tool_name":"remote"`) {
		t.Errorf("stdout should contain the fake CLI's canned finding tagged with the subcommand, got %q", out)
	}
}

func TestRunMCPScannerEndpoint_ExecFailure_WrapsStderr(t *testing.T) {
	t.Parallel()

	cli := fakeCLIPath(t)

	_, err := runMCPScannerEndpoint(context.Background(), cli, "remote", "https://fail-exec.example.com/mcp")
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

// --- endpoint analyzer selection ---

// clearAnalyzerCredentials makes the analyzer set independent of whichever
// credentials the machine running the test happens to export.
func clearAnalyzerCredentials(t *testing.T) {
	t.Helper()

	for _, k := range []string{
		"MCP_SCANNER_LLM_API_KEY",
		"MCP_SCANNER_LLM_MODEL",
		"AZURE_OPENAI_API_KEY",
		"AZURE_OPENAI_BASE_URL",
		"AZURE_OPENAI_DEPLOYMENT",
		"MCP_SCANNER_API_KEY",
	} {
		t.Setenv(k, "")
	}
}

// setAzureCredentials configures a complete LLM setup the way the chart and
// the docker env file do, which is the only form any shipped deployment uses.
func setAzureCredentials(t *testing.T) {
	t.Helper()

	t.Setenv("AZURE_OPENAI_API_KEY", "test-key")
	t.Setenv("AZURE_OPENAI_BASE_URL", "https://example.openai.azure.com")
	t.Setenv("AZURE_OPENAI_DEPLOYMENT", "gpt-4o")
}

func TestEndpointAnalyzers_NoCredentials_CredentialFreeOnly(t *testing.T) {
	// Cannot run in parallel: uses t.Setenv.
	clearAnalyzerCredentials(t)

	want := []string{"yara", "readiness"}
	if got := endpointAnalyzers(); !slices.Equal(got, want) {
		t.Errorf("want %v with no credentials, got %v", want, got)
	}
}

func TestEndpointAnalyzers_AzureConfig_AddsLLM(t *testing.T) {
	// Cannot run in parallel: uses t.Setenv.
	clearAnalyzerCredentials(t)
	setAzureCredentials(t)

	want := []string{"yara", "readiness", "llm"}
	if got := endpointAnalyzers(); !slices.Equal(got, want) {
		t.Errorf("want %v when Azure is fully configured, got %v", want, got)
	}
}

// buildMCPScannerEnv lets a pre-set MCP_SCANNER_LLM_* value win over the
// Azure-derived one, so the analyzer set has to recognise that form too.
func TestEndpointAnalyzers_DirectConfig_AddsLLM(t *testing.T) {
	// Cannot run in parallel: uses t.Setenv.
	clearAnalyzerCredentials(t)
	t.Setenv("MCP_SCANNER_LLM_API_KEY", "test-key")
	t.Setenv("MCP_SCANNER_LLM_MODEL", "openai/test-model")

	want := []string{"yara", "readiness", "llm"}
	if got := endpointAnalyzers(); !slices.Equal(got, want) {
		t.Errorf("want %v when the scanner LLM vars are set directly, got %v", want, got)
	}
}

// A key with no model passes the scanner's own validation and then fails at
// request time, so it must not count as a configured LLM.
func TestEndpointAnalyzers_KeyWithoutModel_OmitsLLM(t *testing.T) {
	// Cannot run in parallel: uses t.Setenv.
	clearAnalyzerCredentials(t)
	t.Setenv("AZURE_OPENAI_API_KEY", "test-key")

	want := []string{"yara", "readiness"}
	if got := endpointAnalyzers(); !slices.Equal(got, want) {
		t.Errorf("want %v when only a key is configured, got %v", want, got)
	}
}

// The api analyzer is opt-in rather than excluded: an operator who configures
// a Cisco AI Defense key gets it, which is what #1775 intended by "opt-in".
func TestEndpointAnalyzers_APIKey_AddsAPI(t *testing.T) {
	// Cannot run in parallel: uses t.Setenv.
	clearAnalyzerCredentials(t)
	t.Setenv("MCP_SCANNER_API_KEY", "test-key")

	want := []string{"yara", "readiness", "api"}
	if got := endpointAnalyzers(); !slices.Equal(got, want) {
		t.Errorf("want %v when an api key is configured, got %v", want, got)
	}
}

func TestEndpointAnalyzers_AllCredentials_RequestsEverything(t *testing.T) {
	// Cannot run in parallel: uses t.Setenv.
	clearAnalyzerCredentials(t)
	setAzureCredentials(t)
	t.Setenv("MCP_SCANNER_API_KEY", "test-key")

	want := []string{"yara", "readiness", "llm", "api"}
	if got := endpointAnalyzers(); !slices.Equal(got, want) {
		t.Errorf("want %v when every key is configured, got %v", want, got)
	}
}

// --- mcpEndpointArgs ---

// The live-server subparsers do not redeclare --analyzers or --raw, so either
// one placed after the subcommand is rejected as unrecognized. That is the
// reverse of mcpSourceArgs, where --output has to trail the subcommand, and the
// two orderings cannot be made to match.
func TestMCPEndpointArgs_GlobalFlagsPrecedeSubcommand(t *testing.T) {
	t.Parallel()

	got := mcpEndpointArgs("prompts", "https://mcp.example.com/mcp", []string{"yara", "readiness"})
	want := []string{
		"--analyzers", "yara,readiness",
		"--raw", "prompts",
		"--server-url", "https://mcp.example.com/mcp",
	}

	if !slices.Equal(got, want) {
		t.Errorf("mcpEndpointArgs() = %q, want %q", got, want)
	}
}

// --- MCPRunner endpoint subcommand ---

func TestMCPRunner_EndpointSubcommand_Success_TagsAndSetsAnalyzer(t *testing.T) {
	// Cannot run in parallel: uses t.Setenv.
	clearAnalyzerCredentials(t)

	r := NewMCPRunner(MCPConfig{CLIPath: fakeCLIPath(t)})

	got := r.runEndpointSubcommand(context.Background(), "prompts", "https://ok.example.com/mcp")

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

	// The analyzers that ran, not the subcommand: "prompts" is already carried
	// by the finding tag asserted above.
	wantAnalyzers := []string{"yara", "readiness"}
	if !slices.Equal(got.Analyzers, wantAnalyzers) {
		t.Errorf("want Analyzers=%v, got %v", wantAnalyzers, got.Analyzers)
	}
}

func TestMCPRunner_EndpointSubcommand_ExecFailure_SkippedNotError(t *testing.T) {
	t.Parallel()

	r := NewMCPRunner(MCPConfig{CLIPath: fakeCLIPath(t)})

	got := r.runEndpointSubcommand(context.Background(), "remote", "https://fail-exec.example.com/mcp")

	if !got.Skipped {
		t.Fatal("an unreachable endpoint must be skipped, not surfaced as a hard error")
	}

	for _, want := range []string{"remote", "fail-exec.example.com", "simulated exec failure"} {
		if !strings.Contains(got.SkippedReason, want) {
			t.Errorf("SkippedReason should contain %q for traceability, got %q", want, got.SkippedReason)
		}
	}
}

func TestMCPRunner_EndpointSubcommand_UnparsableOutput_SkippedNotError(t *testing.T) {
	t.Parallel()

	r := NewMCPRunner(MCPConfig{CLIPath: fakeCLIPath(t)})

	got := r.runEndpointSubcommand(context.Background(), "resources", "https://bad-json.example.com/mcp")

	if !got.Skipped {
		t.Fatal("unparsable mcp-scanner output must be skipped, not surfaced as a hard error")
	}

	if !strings.Contains(got.SkippedReason, "unparsable output") {
		t.Errorf("SkippedReason should mention unparsable output, got %q", got.SkippedReason)
	}
}

func TestMCPRunner_EndpointSubcommand_EmptySafeOutput_NoFindings(t *testing.T) {
	// Cannot run in parallel: uses t.Setenv.
	clearAnalyzerCredentials(t)

	r := NewMCPRunner(MCPConfig{CLIPath: fakeCLIPath(t)})

	got := r.runEndpointSubcommand(context.Background(), "instructions", "https://empty-safe.example.com/mcp")

	if got.Skipped {
		t.Fatalf("want a non-skipped, safe result, got skipped: %s", got.SkippedReason)
	}

	if !got.Safe || len(got.Findings) != 0 {
		t.Errorf("empty mcp-scanner output should produce Safe=true with no findings: %+v", got)
	}

	wantAnalyzers := []string{"yara", "readiness"}
	if !slices.Equal(got.Analyzers, wantAnalyzers) {
		t.Errorf("want Analyzers=%v, got %v", wantAnalyzers, got.Analyzers)
	}
}

// --- MCPRunner endpoint phase, end to end via testdata/fakecli ---

// recordWithEndpoints builds a v1 record whose sole MCP module
// declares one remote-capable ("streamable-http") connection per URL given.
func recordWithEndpoints(t *testing.T, urls ...string) *corev1.Record {
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

func TestMCPRunner_EndpointScan_MergesFindingsAcrossEndpointsAndSubcommands(t *testing.T) {
	// Cannot run in parallel: uses t.Setenv.
	clearAnalyzerCredentials(t)

	r := NewMCPRunner(MCPConfig{CLIPath: fakeCLIPath(t)})

	rec := recordWithEndpoints(t, "https://a.example.com/mcp", "https://b.example.com/mcp")

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

	// Every sub-scan runs the same analyzers, so the union is deduplicated and
	// sorted rather than one entry per sub-scan. That all four subcommands ran
	// is what the finding count above establishes.
	wantAnalyzers := []string{"readiness", "yara"}
	if !slices.Equal(got.Analyzers, wantAnalyzers) {
		t.Errorf("want merged Analyzers=%v, got %v", wantAnalyzers, got.Analyzers)
	}
}

func TestMCPRunner_EndpointScan_AllEndpointsUnreachable_SkippedNotError(t *testing.T) {
	t.Parallel()

	r := NewMCPRunner(MCPConfig{CLIPath: fakeCLIPath(t)})
	rec := recordWithEndpoints(t, "https://fail-exec.example.com/mcp")

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

func TestMCPRunner_EndpointScan_MixedReachability_KeepsSuccessfulFindings(t *testing.T) {
	t.Parallel()

	r := NewMCPRunner(MCPConfig{CLIPath: fakeCLIPath(t)})
	rec := recordWithEndpoints(t, "https://ok.example.com/mcp", "https://fail-exec.example.com/mcp")

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

func TestMCPRunner_EndpointScan_UnparsableEndpoint_SkippedNotError(t *testing.T) {
	t.Parallel()

	r := NewMCPRunner(MCPConfig{CLIPath: fakeCLIPath(t)})
	rec := recordWithEndpoints(t, "https://bad-json.example.com/mcp")

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
