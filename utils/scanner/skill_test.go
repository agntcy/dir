// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package scanner

import (
	"encoding/base64"
	"slices"
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

// --- parseSkillOutput / trimToSkillJSON / parseSkillJSON ---

func TestParseSkillOutput_EmptyBytes(t *testing.T) {
	t.Parallel()

	got, err := parseSkillOutput([]byte{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !got.Safe {
		t.Error("empty input should produce Safe=true")
	}
}

func TestParseSkillOutput_SingleSafeObject(t *testing.T) {
	t.Parallel()

	raw := `{"is_safe":true,"findings":[]}`

	got, err := parseSkillOutput([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !got.Safe || len(got.Findings) != 0 {
		t.Errorf("safe object with no findings: want Safe=true, Findings=0; got %+v", got)
	}
}

func TestParseSkillOutput_SingleUnsafeObject(t *testing.T) {
	t.Parallel()

	raw := `{
		"is_safe": false,
		"findings": [
			{"rule_id":"RULE1","category":"injection","severity":"HIGH","description":"bad stuff"}
		]
	}`

	got, err := parseSkillOutput([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Safe {
		t.Error("is_safe=false should produce Safe=false")
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

func TestParseSkillOutput_FindingsCollectedEvenWhenIsSafeTrue(t *testing.T) {
	t.Parallel()

	// The scanner can report is_safe=true with sub-critical findings present.
	// We must surface the findings regardless and trust the scanner's safe verdict.
	raw := `{"is_safe":true,"findings":[
		{"rule_id":"R1","category":"policy","severity":"MEDIUM","description":"desc"}
	]}`

	got, err := parseSkillOutput([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got.Findings) != 1 {
		t.Errorf("findings must be collected even when is_safe=true; got %d", len(got.Findings))
	}

	// MEDIUM → warning, so Safe should remain true (scanner verdict).
	if !got.Safe {
		t.Error("MEDIUM finding with is_safe=true should keep Safe=true")
	}
}

func TestParseSkillOutput_TrustsScannerSafeVerdict(t *testing.T) {
	t.Parallel()

	raw := `{"is_safe":true,"findings":[
		{"rule_id":"R1","category":"injection","severity":"CRITICAL","description":"critical"}
	]}`

	got, err := parseSkillOutput([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !got.Safe {
		t.Error("is_safe=true should be trusted even when a CRITICAL finding is present")
	}

	if len(got.Findings) != 1 {
		t.Errorf("finding must still be surfaced; got %d", len(got.Findings))
	}
}

func TestParseSkillOutput_ArrayOfResults(t *testing.T) {
	t.Parallel()

	// scan-all produces a JSON array; each element is one skill result.
	raw := `[
		{"is_safe":true,"findings":[]},
		{"is_safe":false,"findings":[
			{"rule_id":"R2","category":"cat","severity":"MEDIUM","description":"d"}
		]}
	]`

	got, err := parseSkillOutput([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Safe {
		t.Error("any is_safe=false in array should produce Safe=false")
	}

	if len(got.Findings) != 1 {
		t.Errorf("want 1 finding from unsafe result, got %d", len(got.Findings))
	}
}

func TestParseSkillOutput_FindingMessageFormat(t *testing.T) {
	t.Parallel()

	raw := `{"is_safe":false,"findings":[
		{"rule_id":"RULE_ID","category":"the_cat","severity":"HIGH","description":"the desc"}
	]}`

	got, err := parseSkillOutput([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	msg := got.Findings[0].Message
	for _, part := range []string{"RULE_ID", "the_cat", "the desc"} {
		if !containsStr(msg, part) {
			t.Errorf("message %q should contain %q", msg, part)
		}
	}
}

func TestParseSkillOutput_InvalidJSON(t *testing.T) {
	t.Parallel()

	_, err := parseSkillOutput([]byte(`not json`))
	if err == nil {
		t.Error("invalid JSON should return an error")
	}
}

// --- trimToSkillJSON ---

func TestTrimToSkillJSON_StartsWithObject(t *testing.T) {
	t.Parallel()

	in := []byte(`{"a":1}`)
	if got := string(trimToSkillJSON(in)); got != string(in) {
		t.Errorf("object JSON should be returned unchanged, got %q", got)
	}
}

func TestTrimToSkillJSON_StartsWithArray(t *testing.T) {
	t.Parallel()

	in := []byte(`[{"a":1}]`)
	if got := string(trimToSkillJSON(in)); got != string(in) {
		t.Errorf("array JSON should be returned unchanged, got %q", got)
	}
}

func TestTrimToSkillJSON_LeadingTextBeforeObject(t *testing.T) {
	t.Parallel()

	in := []byte("progress output\n{\"a\":1}")
	want := `{"a":1}`

	if got := string(trimToSkillJSON(in)); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTrimToSkillJSON_LeadingTextBeforeArray(t *testing.T) {
	t.Parallel()

	in := []byte("progress output\n[1,2]")
	want := "[1,2]"

	if got := string(trimToSkillJSON(in)); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTrimToSkillJSON_NoJSON(t *testing.T) {
	t.Parallel()

	in := []byte("no json here")
	if got := string(trimToSkillJSON(in)); got != string(in) {
		t.Errorf("no JSON start → should return input unchanged, got %q", got)
	}
}

// --- mapSkillSeverity ---

func TestMapSkillSeverity(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  FindingSeverity
	}{
		{"CRITICAL", SeverityError},
		{"critical", SeverityError},
		{"HIGH", SeverityError},
		{"high", SeverityError},
		{"MEDIUM", SeverityWarning},
		{"medium", SeverityWarning},
		{"LOW", SeverityInfo},
		{"low", SeverityInfo},
		{"INFO", SeverityInfo},
		{"UNKNOWN", SeverityInfo},
		{"", SeverityInfo},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()

			if got := mapSkillSeverity(tc.input); got != tc.want {
				t.Errorf("mapSkillSeverity(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// --- skillArtifactBytes ---

func TestSkillArtifactBytes_NilData(t *testing.T) {
	t.Parallel()

	if got := skillArtifactBytes(&corev1.Record{}); got != nil {
		t.Error("nil Data should return nil")
	}
}

func TestSkillArtifactBytes_NoModules(t *testing.T) {
	t.Parallel()

	data, _ := structpb.NewStruct(map[string]any{})
	record := &corev1.Record{Data: data}

	if got := skillArtifactBytes(record); got != nil {
		t.Error("no modules field should return nil")
	}
}

func TestSkillArtifactBytes_WrongModuleName(t *testing.T) {
	t.Parallel()

	data, _ := structpb.NewStruct(map[string]any{
		"modules": []any{
			map[string]any{
				"name":     "some/other/module",
				"artifact": map[string]any{"data": base64.StdEncoding.EncodeToString([]byte("payload"))},
			},
		},
	})

	if got := skillArtifactBytes(&corev1.Record{Data: data}); got != nil {
		t.Error("wrong module name should return nil")
	}
}

func TestSkillArtifactBytes_MissingArtifactData(t *testing.T) {
	t.Parallel()

	data, _ := structpb.NewStruct(map[string]any{
		"modules": []any{
			map[string]any{"name": "core/language_model/agentskills"},
		},
	})

	if got := skillArtifactBytes(&corev1.Record{Data: data}); got != nil {
		t.Error("missing artifact.data should return nil")
	}
}

func TestSkillArtifactBytes_InvalidBase64(t *testing.T) {
	t.Parallel()

	data, _ := structpb.NewStruct(map[string]any{
		"modules": []any{
			map[string]any{
				"name":     "core/language_model/agentskills",
				"artifact": map[string]any{"data": "!!!not-base64!!!"},
			},
		},
	})

	if got := skillArtifactBytes(&corev1.Record{Data: data}); got != nil {
		t.Error("invalid base64 should return nil")
	}
}

func TestSkillArtifactBytes_ValidPayload(t *testing.T) {
	t.Parallel()

	payload := []byte("skill content")
	encoded := base64.StdEncoding.EncodeToString(payload)

	data, _ := structpb.NewStruct(map[string]any{
		"modules": []any{
			map[string]any{
				"name":     "core/language_model/agentskills",
				"artifact": map[string]any{"data": encoded},
			},
		},
	})

	got := skillArtifactBytes(&corev1.Record{Data: data})
	if string(got) != string(payload) {
		t.Errorf("want %q, got %q", payload, got)
	}
}

func TestSkillArtifactBytes_SkipsOtherModules(t *testing.T) {
	t.Parallel()

	payload := []byte("correct")
	encoded := base64.StdEncoding.EncodeToString(payload)

	data, _ := structpb.NewStruct(map[string]any{
		"modules": []any{
			map[string]any{
				"name":     "some/other/module",
				"artifact": map[string]any{"data": base64.StdEncoding.EncodeToString([]byte("wrong"))},
			},
			map[string]any{
				"name":     "core/language_model/agentskills",
				"artifact": map[string]any{"data": encoded},
			},
		},
	})

	got := skillArtifactBytes(&corev1.Record{Data: data})
	if string(got) != string(payload) {
		t.Errorf("should skip non-agentskills modules; want %q, got %q", payload, got)
	}
}

// --- NewSkillRunner / Name ---

func TestNewSkillRunner_DefaultCLIPath(t *testing.T) {
	t.Parallel()

	r := NewSkillRunner(SkillConfig{})
	if r.cfg.CLIPath != DefaultSkillCLIPath {
		t.Errorf("empty CLIPath should default to %q, got %q", DefaultSkillCLIPath, r.cfg.CLIPath)
	}
}

func TestNewSkillRunner_CustomCLIPath(t *testing.T) {
	t.Parallel()

	r := NewSkillRunner(SkillConfig{CLIPath: "/opt/skill-scanner"})
	if r.cfg.CLIPath != "/opt/skill-scanner" {
		t.Errorf("custom CLIPath should be preserved, got %q", r.cfg.CLIPath)
	}
}

func TestSkillRunner_Name(t *testing.T) {
	t.Parallel()

	r := NewSkillRunner(SkillConfig{})
	if got := r.Name(); got != "skill" {
		t.Errorf("Name() = %q, want %q", got, "skill")
	}
}

// --- buildSkillScannerEnv ---

func TestBuildSkillScannerEnv_ContainsParentEnv(t *testing.T) {
	const marker = "TEST_BUILD_SKILL_SCANNER_ENV_MARKER"
	t.Setenv(marker, "present")

	env := buildSkillScannerEnv()

	if !slices.Contains(env, marker+"=present") {
		t.Error("buildSkillScannerEnv should inherit the parent process environment")
	}
}

func TestBuildSkillScannerEnv_MapsAzureVars(t *testing.T) {
	t.Setenv("AZURE_OPENAI_API_KEY", "test-key")
	t.Setenv("AZURE_OPENAI_BASE_URL", "https://openai.example.com")
	t.Setenv("AZURE_OPENAI_DEPLOYMENT", "gpt-4o")
	t.Setenv("AZURE_OPENAI_API_VERSION", "2024-08-01")

	// Ensure SKILL vars are absent so they get derived.
	t.Setenv("SKILL_SCANNER_LLM_API_KEY", "")
	t.Setenv("SKILL_SCANNER_LLM_BASE_URL", "")
	t.Setenv("SKILL_SCANNER_LLM_MODEL", "")
	t.Setenv("SKILL_SCANNER_LLM_API_VERSION", "")
	t.Setenv("SKILL_SCANNER_LLM_PROVIDER", "")

	env := buildSkillScannerEnv()
	envMap := make(map[string]string)

	for _, e := range env {
		if k, v, ok := splitEnvEntry(e); ok {
			envMap[k] = v
		}
	}

	cases := map[string]string{
		"SKILL_SCANNER_LLM_API_KEY":     "test-key",
		"SKILL_SCANNER_LLM_BASE_URL":    "https://openai.example.com",
		"SKILL_SCANNER_LLM_MODEL":       "azure/gpt-4o",
		"SKILL_SCANNER_LLM_API_VERSION": "2024-08-01",
		"SKILL_SCANNER_LLM_PROVIDER":    "openai-compatible",
	}

	for k, want := range cases {
		if got := envMap[k]; got != want {
			t.Errorf("env[%s] = %q, want %q", k, got, want)
		}
	}
}

func TestBuildSkillScannerEnv_ExistingSkillVarNotOverridden(t *testing.T) {
	t.Setenv("AZURE_OPENAI_API_KEY", "azure-key")
	t.Setenv("SKILL_SCANNER_LLM_API_KEY", "already-set")

	env := buildSkillScannerEnv()
	envMap := make(map[string]string)

	for _, e := range env {
		if k, v, ok := splitEnvEntry(e); ok {
			envMap[k] = v
		}
	}

	// appendEnvIfMissing does not override an already-set key.
	if got := envMap["SKILL_SCANNER_LLM_API_KEY"]; got != "already-set" {
		t.Errorf("pre-existing SKILL_SCANNER_LLM_API_KEY should not be overridden; got %q", got)
	}
}
