// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package scanner

import (
	"slices"
	"strings"
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

// --- NewA2ARunner / Name ---

func TestNewA2ARunner_DefaultCLIPath(t *testing.T) {
	t.Parallel()

	r := NewA2ARunner(A2AConfig{})
	if r.cfg.CLIPath != DefaultA2ACLIPath {
		t.Errorf("empty CLIPath should default to %q, got %q", DefaultA2ACLIPath, r.cfg.CLIPath)
	}
}

func TestNewA2ARunner_CustomCLIPath(t *testing.T) {
	t.Parallel()

	r := NewA2ARunner(A2AConfig{CLIPath: "/opt/a2a-scanner"})
	if r.cfg.CLIPath != "/opt/a2a-scanner" {
		t.Errorf("custom CLIPath should be preserved, got %q", r.cfg.CLIPath)
	}
}

func TestA2ARunner_Name(t *testing.T) {
	t.Parallel()

	r := NewA2ARunner(A2AConfig{})
	if got := r.Name(); got != "a2a" {
		t.Errorf("Name() = %q, want %q", got, "a2a")
	}
}

// --- Run / no card found ---

func TestA2ARunner_Run_NilRecord_Skipped(t *testing.T) {
	t.Parallel()

	r := NewA2ARunner(A2AConfig{})

	got, err := r.Run(t.Context(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !got.Skipped || got.SkippedReason == "" {
		t.Errorf("nil record should be skipped with a reason: %+v", got)
	}
}

func TestA2ARunner_Run_NoA2AModule_Skipped(t *testing.T) {
	t.Parallel()

	r := NewA2ARunner(A2AConfig{})

	data, _ := structpb.NewStruct(map[string]any{
		"schema_version": "1.0.0",
		"name":           "no-a2a-here",
		"modules": []any{
			map[string]any{"name": "integration/mcp", "data": map[string]any{"name": "x"}},
		},
	})

	got, err := r.Run(t.Context(), &corev1.Record{Data: data})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !got.Skipped {
		t.Error("record with no a2a_data module should be skipped")
	}
}

// --- extractA2ACard ---

func TestExtractA2ACard_NilRecord(t *testing.T) {
	t.Parallel()

	card, ok := extractA2ACard(nil)
	if ok || card != nil {
		t.Errorf("nil record should return (nil, false), got (%v, %v)", card, ok)
	}
}

func TestExtractA2ACard_NoData(t *testing.T) {
	t.Parallel()

	card, ok := extractA2ACard(&corev1.Record{})
	if ok || card != nil {
		t.Errorf("record with no data should return (nil, false), got (%v, %v)", card, ok)
	}
}

func TestExtractA2ACard_Found(t *testing.T) {
	t.Parallel()

	data, _ := structpb.NewStruct(map[string]any{
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
					"card_schema_version": "v1.0.0",
				},
			},
		},
	})

	card, ok := extractA2ACard(&corev1.Record{Data: data})
	if !ok {
		t.Fatal("expected a card to be found")
	}

	if card["name"] != "burger_seller_agent" {
		t.Errorf("card[name] = %v, want burger_seller_agent", card["name"])
	}
}

// --- getNestedStructValue ---

func TestGetNestedStructValue_Nil(t *testing.T) {
	t.Parallel()

	if got := getNestedStructValue(nil, "a2a_data"); got != nil {
		t.Errorf("nil struct should return nil, got %v", got)
	}
}

func TestGetNestedStructValue_NoKeys(t *testing.T) {
	t.Parallel()

	s, _ := structpb.NewStruct(map[string]any{"a": 1})
	if got := getNestedStructValue(s); got != s {
		t.Errorf("no keys should return the input struct unchanged")
	}
}

func TestGetNestedStructValue_MissingKey(t *testing.T) {
	t.Parallel()

	s, _ := structpb.NewStruct(map[string]any{"other": map[string]any{}})
	if got := getNestedStructValue(s, "a2a_data"); got != nil {
		t.Errorf("missing key should return nil, got %v", got)
	}
}

func TestGetNestedStructValue_NotAStruct(t *testing.T) {
	t.Parallel()

	s, _ := structpb.NewStruct(map[string]any{"a2a_data": "not-a-struct"})
	if got := getNestedStructValue(s, "a2a_data"); got != nil {
		t.Errorf("non-struct value should return nil, got %v", got)
	}
}

func TestGetNestedStructValue_Found(t *testing.T) {
	t.Parallel()

	s, _ := structpb.NewStruct(map[string]any{
		"a2a_data": map[string]any{
			"card_data": map[string]any{"name": "x"},
		},
	})

	got := getNestedStructValue(s, "a2a_data", "card_data")
	if got == nil {
		t.Fatal("expected a struct value")
	}

	if got.AsMap()["name"] != "x" {
		t.Errorf("card_data.name = %v, want x", got.AsMap()["name"])
	}
}

// --- parseA2AOutput ---

func TestParseA2AOutput_EmptyBytes(t *testing.T) {
	t.Parallel()

	got, err := parseA2AOutput([]byte{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !got.Safe {
		t.Error("empty input should produce Safe=true")
	}
}

func TestParseA2AOutput_SafeNoFindings(t *testing.T) {
	t.Parallel()

	raw := `{"status":"completed","findings":[],"total_findings":0,"high_severity_count":0}`

	got, err := parseA2AOutput([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !got.Safe || len(got.Findings) != 0 {
		t.Errorf("zero-finding result: want Safe=true, Findings=0; got %+v", got)
	}
}

func TestParseA2AOutput_UnsafeWithFindings(t *testing.T) {
	t.Parallel()

	raw := `{
		"status": "completed",
		"findings": [
			{"threat_name":"unsafe_delegation","scanner_category":"delegation","severity":"HIGH","description":"unsafe delegation chain"}
		],
		"total_findings": 1,
		"high_severity_count": 1
	}`

	got, err := parseA2AOutput([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Safe {
		t.Error("a result with findings should produce Safe=false")
	}

	if len(got.Findings) != 1 {
		t.Fatalf("want 1 finding, got %d", len(got.Findings))
	}

	f := got.Findings[0]
	if f.Severity != SeverityError {
		t.Errorf("HIGH severity should map to SeverityError, got %q", f.Severity)
	}

	for _, part := range []string{"unsafe_delegation", "delegation", "unsafe delegation chain"} {
		if !strings.Contains(f.Message, part) {
			t.Errorf("message %q should contain %q", f.Message, part)
		}
	}
}

func TestParseA2AOutput_FindingsDeriveUnsafe(t *testing.T) {
	t.Parallel()

	// a2a-scanner reports no is_safe flag: any finding must derive Safe=false.
	// This finding also omits "description", exercising the summary fallback.
	raw := `{"status":"completed","findings":[
		{"threat_name":"prompt_injection","scanner_category":"injection","severity":"CRITICAL","summary":"injection detected"}
	],"total_findings":1,"high_severity_count":1}`

	got, err := parseA2AOutput([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Safe {
		t.Error("a result with findings must be Safe=false even without an is_safe flag")
	}

	if len(got.Findings) != 1 {
		t.Fatalf("finding must be surfaced; got %d", len(got.Findings))
	}

	if got.Findings[0].Severity != SeverityError {
		t.Errorf("CRITICAL severity should map to SeverityError, got %q", got.Findings[0].Severity)
	}

	if !strings.Contains(got.Findings[0].Message, "injection detected") {
		t.Errorf("message should fall back to summary when description is empty, got %q", got.Findings[0].Message)
	}
}

func TestParseA2AOutput_FindingsPresentButZeroCount_Unsafe(t *testing.T) {
	t.Parallel()

	// Defends the len(findings) clause of the safety conjunction: if the scanner
	// ever reports findings without a matching total_findings count, the presence
	// of a finding must still derive Safe=false.
	raw := `{"status":"completed","findings":[
		{"threat_name":"t","scanner_category":"c","severity":"HIGH","description":"d"}
	],"total_findings":0}`

	got, err := parseA2AOutput([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Safe {
		t.Error("a surfaced finding must force Safe=false even when total_findings is 0")
	}
}

func TestParseA2AOutput_NonzeroCountNoFindings_Unsafe(t *testing.T) {
	t.Parallel()

	// Defends the TotalFindings clause: a non-zero count with an empty findings
	// array must still derive Safe=false.
	raw := `{"status":"completed","findings":[],"total_findings":2,"high_severity_count":1}`

	got, err := parseA2AOutput([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Safe {
		t.Error("a non-zero total_findings must force Safe=false even with an empty findings array")
	}
}

func TestParseA2AOutput_InvalidJSON(t *testing.T) {
	t.Parallel()

	_, err := parseA2AOutput([]byte(`not json`))
	if err == nil {
		t.Error("invalid JSON should return an error")
	}
}

func TestParseA2AOutput_LeadingTextStripped(t *testing.T) {
	t.Parallel()

	raw := "progress output\n" + `{"status":"completed","findings":[],"total_findings":0}`

	got, err := parseA2AOutput([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !got.Safe {
		t.Error("leading non-JSON text should be stripped before parsing")
	}
}

// --- trimToA2AJSON ---

func TestTrimToA2AJSON_StartsWithObject(t *testing.T) {
	t.Parallel()

	in := []byte(`{"a":1}`)
	if got := string(trimToA2AJSON(in)); got != string(in) {
		t.Errorf("object JSON should be returned unchanged, got %q", got)
	}
}

func TestTrimToA2AJSON_LeadingTextBeforeObject(t *testing.T) {
	t.Parallel()

	in := []byte("progress output\n{\"a\":1}")
	want := `{"a":1}`

	if got := string(trimToA2AJSON(in)); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTrimToA2AJSON_NoJSON(t *testing.T) {
	t.Parallel()

	in := []byte("no json here")
	if got := string(trimToA2AJSON(in)); got != string(in) {
		t.Errorf("no JSON start should return input unchanged, got %q", got)
	}
}

// --- buildA2AScannerEnv ---

func TestBuildA2AScannerEnv_ContainsParentEnv(t *testing.T) {
	const marker = "TEST_BUILD_A2A_SCANNER_ENV_MARKER"
	t.Setenv(marker, "present")

	env := buildA2AScannerEnv()

	if !slices.Contains(env, marker+"=present") {
		t.Error("buildA2AScannerEnv should inherit the parent process environment")
	}
}

func TestBuildA2AScannerEnv_MapsAzureVars(t *testing.T) {
	testAzureEnvDerivation(t, "A2A_SCANNER", buildA2AScannerEnv)
}

func TestBuildA2AScannerEnv_ExistingA2AVarNotOverridden(t *testing.T) {
	t.Setenv("AZURE_OPENAI_API_KEY", "azure-key")
	t.Setenv("A2A_SCANNER_LLM_API_KEY", "already-set")

	env := buildA2AScannerEnv()
	envMap := make(map[string]string)

	for _, e := range env {
		if k, v, ok := splitEnvEntry(e); ok {
			envMap[k] = v
		}
	}

	if got := envMap["A2A_SCANNER_LLM_API_KEY"]; got != "already-set" {
		t.Errorf("pre-existing A2A_SCANNER_LLM_API_KEY should not be overridden; got %q", got)
	}
}
