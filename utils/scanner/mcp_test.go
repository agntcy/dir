// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package scanner

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	typesv1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

// --- parseMCPOutput / decodeMCPResults ---

func TestParseMCPOutput_EmptyArray(t *testing.T) {
	t.Parallel()

	got, err := parseMCPOutput([]byte(`[]`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !got.Safe {
		t.Error("empty result array should be safe")
	}
}

func TestParseMCPOutput_AllSafe(t *testing.T) {
	t.Parallel()

	raw := `[{"tool_name":"fetch","status":"ok","is_safe":true,"findings":{}}]`

	got, err := parseMCPOutput([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !got.Safe || len(got.Findings) != 0 {
		t.Errorf("all-safe tool should produce Safe=true with no findings: %+v", got)
	}
}

func TestParseMCPOutput_UnsafeWithFindings(t *testing.T) {
	t.Parallel()

	raw := `[{
		"tool_name": "exec",
		"status": "done",
		"is_safe": false,
		"findings": {
			"prompt_injection": {
				"severity": "HIGH",
				"threat_summary": "injects prompts",
				"threat_names": ["jailbreak"],
				"total_findings": 1
			}
		}
	}]`

	got, err := parseMCPOutput([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Safe {
		t.Error("unsafe tool should produce Safe=false")
	}

	if len(got.Findings) != 1 {
		t.Fatalf("want 1 finding, got %d", len(got.Findings))
	}

	f := got.Findings[0]
	if f.Severity != SeverityError {
		t.Errorf("HIGH severity should map to SeverityError, got %q", f.Severity)
	}

	if f.Message == "" {
		t.Error("finding message must not be empty")
	}
}

// Object-shaped findings arrive in a map, whose iteration order Go randomizes.
// Two scans of identical output have to produce identical reports, or every
// scan rewrites its stored result with the same findings in a new order.
func TestParseMCPOutput_FindingOrderIsStable(t *testing.T) {
	t.Parallel()

	raw := `[{
		"tool_name": "exec",
		"status": "done",
		"is_safe": false,
		"findings": {
			"gamma_analyzer": {"severity": "HIGH", "threat_summary": "g"},
			"alpha_analyzer": {"severity": "HIGH", "threat_summary": "a"},
			"beta_analyzer":  {"severity": "HIGH", "threat_summary": "b"}
		}
	}]`

	want := []string{
		"[alpha_analyzer] exec: a",
		"[beta_analyzer] exec: b",
		"[gamma_analyzer] exec: g",
	}

	// Repeated because an unsorted map would still land on the wanted order
	// some of the time.
	for range 10 {
		got, err := parseMCPOutput([]byte(raw))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		msgs := make([]string, 0, len(got.Findings))
		for _, f := range got.Findings {
			msgs = append(msgs, f.Message)
		}

		if !slices.Equal(msgs, want) {
			t.Fatalf("finding order = %q, want %q", msgs, want)
		}
	}
}

// Shape emitted by the `instructions` subcommand: findings is an array whose
// entries name their own analyzer, and the subject is a server, not a tool.
func TestParseMCPOutput_ArrayFindings(t *testing.T) {
	t.Parallel()

	raw := `[{
		"server_name": "DeepWiki",
		"status": "completed",
		"is_safe": false,
		"findings": [
			{"severity":"HIGH","summary":"Detected 1 threat: data exfiltration","analyzer":"YARA"}
		]
	}]`

	got, err := parseMCPOutput([]byte(raw))
	if err != nil {
		t.Fatalf("array-shaped findings must parse, got error: %v", err)
	}

	if got.Safe {
		t.Error("a result carrying findings must not be reported safe")
	}

	if len(got.Findings) != 1 {
		t.Fatalf("want 1 finding, got %d: %+v", len(got.Findings), got.Findings)
	}

	want := "[YARA] DeepWiki: Detected 1 threat: data exfiltration"
	if got.Findings[0].Message != want {
		t.Errorf("want message %q, got %q", want, got.Findings[0].Message)
	}
}

func TestParseMCPOutput_UnknownFindingsShape_Errors(t *testing.T) {
	t.Parallel()

	raw := `[{"tool_name":"exec","is_safe":false,"findings":"not-a-shape-we-know"}]`

	if _, err := parseMCPOutput([]byte(raw)); err == nil {
		t.Error("an unrecognized findings shape must error, not be silently dropped")
	}
}

func TestParseMCPOutput_ThreatNamesAppended(t *testing.T) {
	t.Parallel()

	raw := `[{
		"tool_name": "read_file",
		"is_safe": false,
		"findings": {
			"exfil": {
				"severity": "MEDIUM",
				"threat_summary": "leaks data",
				"threat_names": ["credential_theft","data_leak"],
				"total_findings": 2
			}
		}
	}]`

	got, err := parseMCPOutput([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got.Findings) != 1 {
		t.Fatalf("want 1 finding, got %d", len(got.Findings))
	}

	msg := got.Findings[0].Message
	for _, name := range []string{"credential_theft", "data_leak"} {
		if !containsStr(msg, name) {
			t.Errorf("message %q should contain threat name %q", msg, name)
		}
	}
}

func TestParseMCPOutput_SafeToolDoesNotProduceFindings(t *testing.T) {
	t.Parallel()

	raw := `[
		{"tool_name":"safe_tool","is_safe":true,"findings":{"x":{"severity":"HIGH","threat_summary":"ignored","threat_names":[],"total_findings":1}}},
		{"tool_name":"unsafe_tool","is_safe":false,"findings":{"y":{"severity":"LOW","threat_summary":"low risk","threat_names":[],"total_findings":1}}}
	]`

	got, err := parseMCPOutput([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only the unsafe tool should contribute findings.
	if len(got.Findings) != 1 {
		t.Errorf("want 1 finding (from unsafe tool only), got %d", len(got.Findings))
	}
}

func TestParseMCPOutput_LeadingTextStripped(t *testing.T) {
	t.Parallel()

	raw := "some preamble text\n[{\"tool_name\":\"t\",\"is_safe\":true,\"findings\":{}}]"

	got, err := parseMCPOutput([]byte(raw))
	if err != nil {
		t.Fatalf("leading text should be stripped: %v", err)
	}

	if !got.Safe {
		t.Error("want Safe=true")
	}
}

func TestParseMCPOutput_InvalidJSON(t *testing.T) {
	t.Parallel()

	_, err := parseMCPOutput([]byte(`not json at all`))
	if err == nil {
		t.Error("invalid JSON should return an error")
	}
}

// --- decodeMCPResults ---

// An ANSI colour escape puts a '[' ahead of the results.
func TestDecodeMCPResults_ANSIBannerBeforeResults(t *testing.T) {
	t.Parallel()

	raw := []byte("\x1b[1;31mGive Feedback / Get Help\x1b[0m\n" +
		`[{"tool_name":"exec","is_safe":false,"findings":{"yara":{"severity":"HIGH","threat_summary":"s"}}}]`)

	got, err := parseMCPOutput(raw)
	if err != nil {
		t.Fatalf("results preceded by an ANSI banner must still parse: %v", err)
	}

	if len(got.Findings) != 1 {
		t.Fatalf("want 1 finding, got %d: %+v", len(got.Findings), got.Findings)
	}
}

func TestDecodeMCPResults_TrailingLogLines(t *testing.T) {
	t.Parallel()

	raw := []byte(`[{"tool_name":"exec","is_safe":true,"findings":{}}]` + "\nINFO scan complete\n")

	if _, err := parseMCPOutput(raw); err != nil {
		t.Fatalf("log lines after the results must not break parsing: %v", err)
	}
}

// An empty array in a log line must not stand in for the real results.
func TestDecodeMCPResults_EmptyArrayInLogPrefersRealResults(t *testing.T) {
	t.Parallel()

	raw := []byte("INFO analyzers=[] starting\n" +
		`[{"tool_name":"exec","is_safe":false,"findings":{"yara":{"severity":"HIGH","threat_summary":"s"}}}]`)

	got, err := parseMCPOutput(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Safe {
		t.Error("want Safe=false: the real results carry a finding")
	}
}

// --- getNestedString ---

func TestGetNestedString_Nil(t *testing.T) {
	t.Parallel()

	if got := getNestedString(nil, "key"); got != "" {
		t.Errorf("nil struct should return empty string, got %q", got)
	}
}

func TestGetNestedString_NoKeys(t *testing.T) {
	t.Parallel()

	s, _ := structpb.NewStruct(map[string]any{"k": "v"})
	if got := getNestedString(s); got != "" {
		t.Errorf("no keys should return empty string, got %q", got)
	}
}

func TestGetNestedString_MissingKey(t *testing.T) {
	t.Parallel()

	s, _ := structpb.NewStruct(map[string]any{"other": "val"})
	if got := getNestedString(s, "missing"); got != "" {
		t.Errorf("missing key should return empty string, got %q", got)
	}
}

func TestGetNestedString_LeafValue(t *testing.T) {
	t.Parallel()

	s, _ := structpb.NewStruct(map[string]any{"name": "hello"})
	if got := getNestedString(s, "name"); got != "hello" {
		t.Errorf("want %q, got %q", "hello", got)
	}
}

func TestGetNestedString_NestedValue(t *testing.T) {
	t.Parallel()

	s, _ := structpb.NewStruct(map[string]any{
		"a": map[string]any{
			"b": map[string]any{
				"c": "deep",
			},
		},
	})

	if got := getNestedString(s, "a", "b", "c"); got != "deep" {
		t.Errorf("want %q, got %q", "deep", got)
	}
}

func TestGetNestedString_IntermediateNotStruct(t *testing.T) {
	t.Parallel()

	s, _ := structpb.NewStruct(map[string]any{"a": "not-a-struct"})
	if got := getNestedString(s, "a", "b"); got != "" {
		t.Errorf("non-struct intermediate should return empty string, got %q", got)
	}
}

// --- extractSourceCodeURL ---

func TestExtractSourceCodeURL_Empty(t *testing.T) {
	t.Parallel()

	if got := extractSourceCodeURL(nil); got != "" {
		t.Errorf("nil locators should return empty string, got %q", got)
	}
}

func TestExtractSourceCodeURL_NoMatchingType(t *testing.T) {
	t.Parallel()

	locs := []*typesv1.Locator{
		{Type: "website", Urls: []string{"https://example.com"}},
	}

	if got := extractSourceCodeURL(locs); got != "" {
		t.Errorf("no source_code locator should return empty string, got %q", got)
	}
}

func TestExtractSourceCodeURL_Found(t *testing.T) {
	t.Parallel()

	want := "https://github.com/example/repo"
	locs := []*typesv1.Locator{
		{Type: "website", Urls: []string{"https://example.com"}},
		{Type: "source_code", Urls: []string{want}},
	}

	if got := extractSourceCodeURL(locs); got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestExtractSourceCodeURL_SourceCodeNoURLs(t *testing.T) {
	t.Parallel()

	locs := []*typesv1.Locator{
		{Type: "source_code", Urls: []string{}},
	}

	if got := extractSourceCodeURL(locs); got != "" {
		t.Errorf("source_code with no URLs should return empty string, got %q", got)
	}
}

// --- extractSubfolder ---

func TestExtractSubfolder_Empty(t *testing.T) {
	t.Parallel()

	if got := extractSubfolder(nil); got != "" {
		t.Errorf("nil modules should return empty string, got %q", got)
	}
}

func TestExtractSubfolder_NoMCPData(t *testing.T) {
	t.Parallel()

	data, _ := structpb.NewStruct(map[string]any{"other_data": "x"})
	mods := []*typesv1.Module{{Data: data}}

	if got := extractSubfolder(mods); got != "" {
		t.Errorf("module without mcp_data should return empty string, got %q", got)
	}
}

func TestExtractSubfolder_Found(t *testing.T) {
	t.Parallel()

	data, _ := structpb.NewStruct(map[string]any{
		"repository": map[string]any{
			"subfolder": "src/server",
		},
	})
	mods := []*typesv1.Module{{Data: data}}

	if got := extractSubfolder(mods); got != "src/server" {
		t.Errorf("want %q, got %q", "src/server", got)
	}
}

// --- NewMCPRunner / Name ---

func TestNewMCPRunner_DefaultCLIPath(t *testing.T) {
	t.Parallel()

	r := NewMCPRunner(MCPConfig{})
	if r.cfg.CLIPath != DefaultMCPCLIPath {
		t.Errorf("empty CLIPath should default to %q, got %q", DefaultMCPCLIPath, r.cfg.CLIPath)
	}
}

func TestNewMCPRunner_CustomCLIPath(t *testing.T) {
	t.Parallel()

	r := NewMCPRunner(MCPConfig{CLIPath: "/usr/local/bin/mcp-scanner"})
	if r.cfg.CLIPath != "/usr/local/bin/mcp-scanner" {
		t.Errorf("custom CLIPath should be preserved, got %q", r.cfg.CLIPath)
	}
}

func TestMCPRunner_Name(t *testing.T) {
	t.Parallel()

	r := NewMCPRunner(MCPConfig{})
	if got := r.Name(); got != "mcp" {
		t.Errorf("Name() = %q, want %q", got, "mcp")
	}
}

// --- MCPRunner.Run (end to end: real `git clone` of a local repo + testdata/fakecli) ---

// localGitRepo creates a throwaway git repository under t.TempDir() with a
// single committed file, and returns its filesystem path. gitClone (git
// clone --depth=1) accepts a plain local path as the repo URL, so this lets
// MCPRunner.Run be exercised end to end without any network access.
func localGitRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()

	run := func(args ...string) {
		t.Helper()

		//nolint:noctx // test helper, no request-scoped context available
		cmd := exec.Command("git", args...)
		cmd.Dir = dir

		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}

	run("init", "--initial-branch=main")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "Test")

	readme := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readme, []byte("# fixture repo\n"), 0o600); err != nil {
		t.Fatalf("write fixture file: %v", err)
	}

	run("add", "README.md")
	run("commit", "-m", "initial commit")

	return dir
}

// sourceRecord builds a record whose source-code locator points at repoDir. A
// subfolder becomes part of the scan path, which is how a test steers the fake
// CLI, since the fake matches on argv.
func sourceRecord(t *testing.T, repoDir, subfolder string, endpoints ...string) *corev1.Record {
	t.Helper()

	moduleData := map[string]any{}

	if subfolder != "" {
		moduleData["repository"] = map[string]any{"subfolder": subfolder}
	}

	if len(endpoints) > 0 {
		conns := make([]any, 0, len(endpoints))
		for _, u := range endpoints {
			conns = append(conns, map[string]any{"type": "streamable-http", "url": u})
		}

		moduleData["connections"] = conns
	}

	data, err := structpb.NewStruct(map[string]any{
		"schema_version": "1.0.0",
		"locators": []any{
			map[string]any{"type": "source_code", "urls": []any{repoDir}},
		},
		"modules": []any{
			map[string]any{"name": "core/mcp", "data": moduleData},
		},
	})
	if err != nil {
		t.Fatalf("structpb.NewStruct: %v", err)
	}

	return &corev1.Record{Data: data}
}

func TestMCPRunner_Run_Success(t *testing.T) {
	// Cannot run in parallel: uses t.Setenv.
	clearAnalyzerCredentials(t)
	setAzureCredentials(t)

	r := NewMCPRunner(MCPConfig{CLIPath: fakeCLIPath(t)})

	got, err := r.Run(context.Background(), sourceRecord(t, localGitRepo(t), ""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Safe {
		t.Error("want Safe=false: the fake CLI reports one unsafe finding")
	}

	if len(got.Findings) != 1 {
		t.Fatalf("want 1 finding, got %d: %+v", len(got.Findings), got.Findings)
	}

	// behavioral is what the source subcommand runs, and the record declares
	// no endpoints, so nothing else joins the union.
	wantAnalyzers := []string{"behavioral"}
	if !slices.Equal(got.Analyzers, wantAnalyzers) {
		t.Errorf("want Analyzers=%v, got %v", wantAnalyzers, got.Analyzers)
	}
}

// behavioral is the only subcommand that takes a source tree and it exits
// non-zero without LLM credentials, so an unconfigured deployment must skip
// the phase rather than fail every record that declares source code.
func TestMCPRunner_Run_NoLLMConfigured_SourceSkippedNotError(t *testing.T) {
	// Cannot run in parallel: uses t.Setenv.
	clearAnalyzerCredentials(t)

	r := NewMCPRunner(MCPConfig{CLIPath: fakeCLIPath(t), DisableEndpointScan: true})

	got, err := r.Run(context.Background(), sourceRecord(t, localGitRepo(t), ""))
	if err != nil {
		t.Fatalf("a missing LLM configuration must not fail the scan: %v", err)
	}

	if !got.Skipped {
		t.Fatal("want the source phase skipped when no LLM is configured")
	}

	if !strings.Contains(got.SkippedReason, "LLM") {
		t.Errorf("want a reason naming the missing LLM configuration, got %q", got.SkippedReason)
	}
}

// A skipped source phase must not take the endpoint findings with it.
func TestMCPRunner_Run_NoLLMConfigured_KeepsEndpointFindings(t *testing.T) {
	// Cannot run in parallel: uses t.Setenv.
	clearAnalyzerCredentials(t)

	r := NewMCPRunner(MCPConfig{CLIPath: fakeCLIPath(t)})

	got, err := r.Run(context.Background(), sourceRecord(t, localGitRepo(t), "", "https://ok.example.com/mcp"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got.Findings) == 0 {
		t.Fatal("want the endpoint findings to survive a skipped source phase")
	}

	// Every finding must carry an endpoint tag: one from the source phase
	// would mean behavioral ran without an LLM configured for it.
	for _, f := range got.Findings {
		if !strings.Contains(f.Message, "ok.example.com") {
			t.Errorf("want only endpoint findings, got %q", f.Message)
		}
	}
}

// A scanner that cannot run at all is still a hard failure: unlike a missing
// credential, it leaves no basis for reporting on the record.
func TestMCPRunner_Run_SourceExecFailure_IsError(t *testing.T) {
	// Cannot run in parallel: uses t.Setenv.
	clearAnalyzerCredentials(t)
	setAzureCredentials(t)

	r := NewMCPRunner(MCPConfig{CLIPath: fakeCLIPath(t), DisableEndpointScan: true})

	// The subfolder lands in the scan path, which is what the fake CLI matches.
	_, err := r.Run(context.Background(), sourceRecord(t, localGitRepo(t), "fail-exec"))
	if err == nil {
		t.Fatal("want an error when mcp-scanner cannot run")
	}

	if !strings.Contains(err.Error(), "exited with error") {
		t.Errorf("want the scanner's exec failure, got %q", err)
	}
}

// --- mcpSourceArgs ---

// The behavioral subparser redeclares --output, so a leading one is overwritten
// by the subparser's default and the results never reach the file. Tidying
// these into the conventional flags-first shape fails silently: the scan still
// exits zero, having written nothing.
func TestMCPSourceArgs_OutputFollowsSubcommand(t *testing.T) {
	t.Parallel()

	got := mcpSourceArgs("/src/server", "/out/scan-result.json")
	want := []string{"behavioral", "/src/server", "--output", "/out/scan-result.json"}

	if !slices.Equal(got, want) {
		t.Errorf("mcpSourceArgs() = %q, want %q", got, want)
	}
}

// --- buildMCPScannerEnv ---

func TestBuildMCPScannerEnv_ContainsParentEnv(t *testing.T) {
	// Cannot run in parallel: uses t.Setenv.
	const marker = "TEST_BUILD_MCP_SCANNER_ENV_MARKER"
	t.Setenv(marker, "present")

	env := buildMCPScannerEnv()

	if slices.Contains(env, marker+"=present") {
		return
	}

	t.Errorf("buildMCPScannerEnv should inherit the parent process environment")
}

func TestBuildMCPScannerEnv_MapsAzureVars(t *testing.T) {
	// Cannot run in parallel: uses t.Setenv.
	t.Setenv("AZURE_OPENAI_API_KEY", "test-key")
	t.Setenv("AZURE_OPENAI_BASE_URL", "https://openai.example.com")
	t.Setenv("AZURE_OPENAI_DEPLOYMENT", "gpt-4")
	t.Setenv("AZURE_OPENAI_API_VERSION", "2024-02-01")

	// Ensure MCP vars are not set so they get derived.
	t.Setenv("MCP_SCANNER_LLM_API_KEY", "")
	t.Setenv("MCP_SCANNER_LLM_BASE_URL", "")
	t.Setenv("MCP_SCANNER_LLM_MODEL", "")
	t.Setenv("MCP_SCANNER_LLM_API_VERSION", "")

	env := buildMCPScannerEnv()
	envMap := make(map[string]string)

	for _, e := range env {
		if k, v, ok := splitEnvEntry(e); ok {
			envMap[k] = v
		}
	}

	cases := map[string]string{
		"MCP_SCANNER_LLM_API_KEY":     "test-key",
		"MCP_SCANNER_LLM_BASE_URL":    "https://openai.example.com",
		"MCP_SCANNER_LLM_MODEL":       "azure/gpt-4",
		"MCP_SCANNER_LLM_API_VERSION": "2024-02-01",
	}

	for k, want := range cases {
		if got := envMap[k]; got != want {
			t.Errorf("env[%s] = %q, want %q", k, got, want)
		}
	}
}

// A deployment is the only part of the model name that varies, so without one
// there is no model to name. Exporting the "azure/" prefix on its own hands the
// scanner a model it cannot resolve.
func TestBuildMCPScannerEnv_NoDeployment_OmitsModel(t *testing.T) {
	// Cannot run in parallel: uses t.Setenv.
	t.Setenv("AZURE_OPENAI_API_KEY", "test-key")
	t.Setenv("AZURE_OPENAI_BASE_URL", "https://openai.example.com")
	t.Setenv("AZURE_OPENAI_DEPLOYMENT", "")
	t.Setenv("MCP_SCANNER_LLM_MODEL", "")

	for _, e := range buildMCPScannerEnv() {
		if k, v, ok := splitEnvEntry(e); ok && k == "MCP_SCANNER_LLM_MODEL" && v != "" {
			t.Errorf("MCP_SCANNER_LLM_MODEL = %q, want it unset without a deployment", v)
		}
	}
}

func splitEnvEntry(e string) (string, string, bool) {
	for i, c := range e {
		if c == '=' {
			return e[:i], e[i+1:], true
		}
	}

	return "", "", false
}

// --- appendEnvIfMissing ---

func TestAppendEnvIfMissing_FallbackEmpty(t *testing.T) {
	t.Parallel()

	env := []string{"EXISTING=val"}
	got := appendEnvIfMissing(env, "NEW_KEY", "")

	if len(got) != len(env) {
		t.Error("empty fallback should leave env unchanged")
	}
}

func TestAppendEnvIfMissing_KeyAlreadySet(t *testing.T) {
	// Cannot run in parallel: modifies process env.
	const key = "TEST_APPEND_ENV_ALREADY_SET_1"
	t.Setenv(key, "existing")

	env := []string{}
	got := appendEnvIfMissing(env, key, "fallback")

	if len(got) != 0 {
		t.Error("key already in process env should leave env slice unchanged")
	}
}

func TestAppendEnvIfMissing_AppendsWhenMissing(t *testing.T) {
	// Cannot run in parallel: relies on key being absent from process env.
	const key = "TEST_APPEND_ENV_MISSING_2"

	env := []string{}
	got := appendEnvIfMissing(env, key, "injected")

	if len(got) != 1 || got[0] != key+"=injected" {
		t.Errorf("want [%s=injected], got %v", key, got)
	}
}

// --- helpers ---

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && stringContains(s, sub))
}

func stringContains(s, sub string) bool {
	for i := range s {
		if i+len(sub) <= len(s) && s[i:i+len(sub)] == sub {
			return true
		}
	}

	return false
}
