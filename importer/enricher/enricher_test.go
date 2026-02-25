// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package enricher

import (
	"os"
	"path/filepath"
	"testing"
)

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
	if err := os.WriteFile(path, []byte("file content"), 0600); err != nil {
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

func TestLoadPromptTemplate_FilePathWithSlash(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "template.txt")
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte("nested content"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	result, err := loadPromptTemplate(path, "default")
	if err != nil {
		t.Fatalf("loadPromptTemplate() error = %v", err)
	}
	if result != "nested content" {
		t.Errorf("loadPromptTemplate() = %q, want %q", result, "nested content")
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
	fields, err := client.parseResponse(response, fieldTypeSkills)
	// When all fields are skipped (e.g. invalid confidence), parseResponse returns error
	if err == nil {
		if len(fields) != 0 {
			t.Errorf("parseResponse() returned %d fields (invalid confidence should be skipped), want 0", len(fields))
		}
	}
}

func TestDefaultConfidenceThreshold(t *testing.T) {
	if DefaultConfidenceThreshold <= 0 || DefaultConfidenceThreshold > 1 {
		t.Errorf("DefaultConfidenceThreshold = %f, should be in (0, 1]", DefaultConfidenceThreshold)
	}
}

func TestDefaultRequestsPerMinute(t *testing.T) {
	if DefaultRequestsPerMinute <= 0 {
		t.Errorf("DefaultRequestsPerMinute = %d, should be positive", DefaultRequestsPerMinute)
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
