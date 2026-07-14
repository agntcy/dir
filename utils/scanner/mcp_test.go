// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package scanner

import (
	"slices"
	"testing"

	typesv1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

// --- parseMCPOutput / trimToJSON / mapMCPSeverity ---

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
		t.Fatalf("trimToJSON should strip leading text: %v", err)
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

// --- trimToJSON ---

func TestTrimToJSON_AlreadyJSON(t *testing.T) {
	t.Parallel()

	in := []byte(`[{"a":1}]`)
	if got := string(trimToJSON(in)); got != string(in) {
		t.Errorf("clean JSON should be returned unchanged, got %q", got)
	}
}

func TestTrimToJSON_WithPreamble(t *testing.T) {
	t.Parallel()

	in := []byte("warning: something\n[1,2,3]")
	want := "[1,2,3]"

	if got := string(trimToJSON(in)); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTrimToJSON_NoArray(t *testing.T) {
	t.Parallel()

	in := []byte("no array here")
	if got := string(trimToJSON(in)); got != string(in) {
		t.Errorf("no '[' → should return input unchanged, got %q", got)
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
		"mcp_data": map[string]any{
			"repository": map[string]any{
				"subfolder": "src/server",
			},
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
