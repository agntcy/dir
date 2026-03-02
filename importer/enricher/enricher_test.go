// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:dupl
package enricher

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	typesv1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1"
	typesv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
)

// mockHostRunner returns a fixed response or error for testing without a real LLM.
type mockHostRunner struct {
	response string
	err      error
}

func (m *mockHostRunner) Prompt(_ context.Context, _ string) (string, error) {
	if m.err != nil {
		return "", m.err
	}

	return m.response, nil
}

func (m *mockHostRunner) PromptWithCallbacks(ctx context.Context, prompt string, _ func(string, string), _ func(string, string, string, bool), _ func(string)) (string, error) {
	return m.Prompt(ctx, prompt)
}

func (m *mockHostRunner) ClearSession() {}

func TestLoadPromptTemplate_EmptyUsesDefault(t *testing.T) {
	defaultTemplate := "default content here"

	result, err := loadPromptTemplate("", defaultTemplate)
	if err != nil {
		t.Fatalf("loadPromptTemplate() error = %v", err)
	}

	if result != defaultTemplate {
		t.Errorf("loadPromptTemplate() = %q, want %q", result, defaultTemplate)
	}
}

func TestLoadPromptTemplate_InlineReturnsAsIs(t *testing.T) {
	inline := "inline prompt template"

	result, err := loadPromptTemplate(inline, "default")
	if err != nil {
		t.Fatalf("loadPromptTemplate() error = %v", err)
	}

	if result != inline {
		t.Errorf("loadPromptTemplate() = %q, want %q", result, inline)
	}
}

func TestLoadPromptTemplate_FilePath(t *testing.T) {
	dir := t.TempDir()

	path := filepath.Join(dir, "template.md")
	if err := os.WriteFile(path, []byte("file content"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	result, err := loadPromptTemplate(path, "default")
	if err != nil {
		t.Fatalf("loadPromptTemplate() error = %v", err)
	}

	if result != "file content" {
		t.Errorf("loadPromptTemplate() = %q, want %q", result, "file content")
	}
}

func TestLoadPromptTemplate_FileNotFound(t *testing.T) {
	_, err := loadPromptTemplate("/nonexistent/path/template.md", "default")
	if err == nil {
		t.Fatal("loadPromptTemplate() expected error for missing file")
	}

	if !contains(err.Error(), "failed to read prompt template file") {
		t.Errorf("error = %v, want containing 'failed to read prompt template file'", err)
	}
}

func TestParseResponse_ValidSkills(t *testing.T) {
	client := &MCPHostClient{}
	response := `{"skills":[{"name":"parent/child","id":1,"confidence":0.9,"reasoning":"test"}]}`

	fields, err := client.parseResponse(response, fieldTypeSkills)
	if err != nil {
		t.Fatalf("parseResponse() error = %v", err)
	}

	if len(fields) != 1 {
		t.Fatalf("parseResponse() returned %d fields, want 1", len(fields))
	}

	if fields[0].Name != "parent/child" || fields[0].ID != 1 || fields[0].Confidence != 0.9 {
		t.Errorf("parseResponse() field = %+v", fields[0])
	}
}

func TestParseResponse_ValidDomains(t *testing.T) {
	client := &MCPHostClient{}
	response := `{"domains":[{"name":"domain/sub","id":10,"confidence":0.8,"reasoning":"ok"}]}`

	fields, err := client.parseResponse(response, fieldTypeDomains)
	if err != nil {
		t.Fatalf("parseResponse() error = %v", err)
	}

	if len(fields) != 1 {
		t.Fatalf("parseResponse() returned %d fields, want 1", len(fields))
	}

	if fields[0].Name != "domain/sub" || fields[0].ID != 10 {
		t.Errorf("parseResponse() field = %+v", fields[0])
	}
}

func TestParseResponse_InvalidJSON(t *testing.T) {
	client := &MCPHostClient{}

	_, err := client.parseResponse("not json", fieldTypeSkills)
	if err == nil {
		t.Fatal("parseResponse() expected error for invalid JSON")
	}
}

func TestParseResponse_NoValidSkills(t *testing.T) {
	client := &MCPHostClient{}
	// Valid JSON but skill name missing slash (invalid format)
	response := `{"skills":[{"name":"noparentslash","id":1,"confidence":0.9}]}`

	_, err := client.parseResponse(response, fieldTypeSkills)
	if err == nil {
		t.Fatal("parseResponse() expected error when no valid skills")
	}

	if !contains(err.Error(), "no valid skills") {
		t.Errorf("error = %v, want containing 'no valid skills'", err)
	}
}

func TestParseResponse_SkillWithZeroIDSkipped(t *testing.T) {
	client := &MCPHostClient{}
	// One valid (parent/child, id=1), one invalid (id=0)
	response := `{"skills":[{"name":"valid/skill","id":1,"confidence":0.9},{"name":"invalid/id","id":0,"confidence":0.9}]}`

	fields, err := client.parseResponse(response, fieldTypeSkills)
	if err != nil {
		t.Fatalf("parseResponse() error = %v", err)
	}

	if len(fields) != 1 {
		t.Errorf("parseResponse() returned %d fields (id=0 should be skipped), want 1", len(fields))
	}
}

func TestParseResponse_InvalidConfidenceSkipped(t *testing.T) {
	client := &MCPHostClient{}
	response := `{"skills":[{"name":"a/b","id":1,"confidence":1.5}]}`
	_, err := client.parseResponse(response, fieldTypeSkills)
	// When all fields are skipped (e.g. invalid confidence), parseResponse returns error
	if err == nil {
		t.Fatal("parseResponse() expected error when confidence is out of range")
	}
}

func TestParseResponse_NoValidDomains(t *testing.T) {
	client := &MCPHostClient{}
	response := `{"domains":[{"name":"noparentslash","id":1,"confidence":0.9}]}`

	_, err := client.parseResponse(response, fieldTypeDomains)
	if err == nil {
		t.Fatal("parseResponse() expected error when no valid domains")
	}

	if !contains(err.Error(), "no valid domains") {
		t.Errorf("error = %v, want containing 'no valid domains'", err)
	}
}

func TestEnrichWithSkills_Success(t *testing.T) {
	ctx := context.Background()
	mock := &mockHostRunner{
		response: `{"skills":[{"name":"parent/child","id":1,"confidence":0.9,"reasoning":"test"}]}`,
	}

	client, err := NewMCPHostWithRunner(mock, Config{})
	if err != nil {
		t.Fatalf("NewMCPHostWithRunner() error = %v", err)
	}

	record := &typesv1alpha1.Record{
		Name:    "test-server",
		Version: "1.0.0",
	}

	got, err := client.EnrichWithSkills(ctx, record)
	if err != nil {
		t.Fatalf("EnrichWithSkills() error = %v", err)
	}

	if got == nil || len(got.GetSkills()) != 1 {
		t.Errorf("EnrichWithSkills() record has %d skills, want 1", len(got.GetSkills()))
	}

	if len(got.GetSkills()) > 0 && (got.GetSkills()[0].GetName() != "parent/child" || got.GetSkills()[0].GetId() != 1) {
		t.Errorf("EnrichWithSkills() skill = %+v", got.GetSkills()[0])
	}
}

func TestEnrichWithDomains_Success(t *testing.T) {
	ctx := context.Background()
	mock := &mockHostRunner{
		response: `{"domains":[{"name":"domain/sub","id":10,"confidence":0.8,"reasoning":"ok"}]}`,
	}

	client, err := NewMCPHostWithRunner(mock, Config{})
	if err != nil {
		t.Fatalf("NewMCPHostWithRunner() error = %v", err)
	}

	record := &typesv1alpha1.Record{
		Name:    "test-server",
		Version: "1.0.0",
	}

	got, err := client.EnrichWithDomains(ctx, record)
	if err != nil {
		t.Fatalf("EnrichWithDomains() error = %v", err)
	}

	if got == nil || len(got.GetDomains()) != 1 {
		t.Errorf("EnrichWithDomains() record has %d domains, want 1", len(got.GetDomains()))
	}

	if len(got.GetDomains()) > 0 && (got.GetDomains()[0].GetName() != "domain/sub" || got.GetDomains()[0].GetId() != 10) {
		t.Errorf("EnrichWithDomains() domain = %+v", got.GetDomains()[0])
	}
}

func TestEnrichWithSkillsV1_Success(t *testing.T) {
	ctx := context.Background()
	mock := &mockHostRunner{
		response: `{"skills":[{"name":"a/b","id":2,"confidence":0.95,"reasoning":"v1"}]}`,
	}

	client, err := NewMCPHostWithRunner(mock, Config{})
	if err != nil {
		t.Fatalf("NewMCPHostWithRunner() error = %v", err)
	}

	record := &typesv1.Record{
		Name:    "test-server",
		Version: "1.0.0",
	}

	got, err := client.EnrichWithSkillsV1(ctx, record)
	if err != nil {
		t.Fatalf("EnrichWithSkillsV1() error = %v", err)
	}

	if got == nil || len(got.GetSkills()) != 1 {
		t.Errorf("EnrichWithSkillsV1() record has %d skills, want 1", len(got.GetSkills()))
	}

	if len(got.GetSkills()) > 0 && (got.GetSkills()[0].GetName() != "a/b" || got.GetSkills()[0].GetId() != 2) {
		t.Errorf("EnrichWithSkillsV1() skill = %+v", got.GetSkills()[0])
	}
}

func TestEnrichWithDomainsV1_Success(t *testing.T) {
	ctx := context.Background()
	mock := &mockHostRunner{
		response: `{"domains":[{"name":"x/y","id":20,"confidence":0.7}]}`,
	}

	client, err := NewMCPHostWithRunner(mock, Config{})
	if err != nil {
		t.Fatalf("NewMCPHostWithRunner() error = %v", err)
	}

	record := &typesv1.Record{
		Name:    "test-server",
		Version: "1.0.0",
	}

	got, err := client.EnrichWithDomainsV1(ctx, record)
	if err != nil {
		t.Fatalf("EnrichWithDomainsV1() error = %v", err)
	}

	if got == nil || len(got.GetDomains()) != 1 {
		t.Errorf("EnrichWithDomainsV1() record has %d domains, want 1", len(got.GetDomains()))
	}

	if len(got.GetDomains()) > 0 && (got.GetDomains()[0].GetName() != "x/y" || got.GetDomains()[0].GetId() != 20) {
		t.Errorf("EnrichWithDomainsV1() domain = %+v", got.GetDomains()[0])
	}
}

func TestEnrichWithSkills_PromptError(t *testing.T) {
	ctx := context.Background()
	mock := &mockHostRunner{err: errors.New("prompt failed")}

	client, err := NewMCPHostWithRunner(mock, Config{})
	if err != nil {
		t.Fatalf("NewMCPHostWithRunner() error = %v", err)
	}

	record := &typesv1alpha1.Record{Name: "test", Version: "1.0.0"}

	_, err = client.EnrichWithSkills(ctx, record)
	if err == nil {
		t.Fatal("EnrichWithSkills() expected error when prompt fails")
	}

	if !contains(err.Error(), "prompt") && !contains(err.Error(), "prompt failed") {
		t.Errorf("error = %v", err)
	}
}

func TestEnrichWithSkills_LowConfidenceFiltered(t *testing.T) {
	ctx := context.Background()
	// One high-confidence, one below DefaultConfidenceThreshold (0.5)
	mock := &mockHostRunner{
		response: `{"skills":[{"name":"high/conf","id":1,"confidence":0.9},{"name":"low/conf","id":2,"confidence":0.3}]}`,
	}

	client, err := NewMCPHostWithRunner(mock, Config{})
	if err != nil {
		t.Fatalf("NewMCPHostWithRunner() error = %v", err)
	}

	record := &typesv1alpha1.Record{Name: "test", Version: "1.0.0"}

	got, err := client.EnrichWithSkills(ctx, record)
	if err != nil {
		t.Fatalf("EnrichWithSkills() error = %v", err)
	}
	// Only high-confidence skill should be added
	if len(got.GetSkills()) != 1 || got.GetSkills()[0].GetName() != "high/conf" {
		t.Errorf("EnrichWithSkills() expected 1 skill (high/conf), got %d: %v", len(got.GetSkills()), got.GetSkills())
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || (len(s) > 0 && len(sub) > 0 && findSub(s, sub)))
}

func findSub(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}

	return false
}
