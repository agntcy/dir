// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package skill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSplitSkillFrontmatter(t *testing.T) {
	t.Parallel()

	yaml, body, err := splitSkillFrontmatter("---\nname: x\ndescription: y\n---\n\nHello body.\n")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(yaml, "name: x") {
		t.Fatalf("yaml = %q", yaml)
	}

	if strings.TrimSpace(body) != "Hello body." {
		t.Fatalf("body = %q", body)
	}
}

func TestSplitSkillFrontmatter_missingClose(t *testing.T) {
	t.Parallel()

	_, _, err := splitSkillFrontmatter("---\nname: x\n")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseSkillDirectory_success(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	skillDir := filepath.Join(dir, "pdf-processing")
	if err := os.MkdirAll(skillDir, 0o700); err != nil {
		t.Fatal(err)
	}

	content := "---\nname: pdf-processing\ndescription: Extract and merge PDFs. Use when the user works with PDF documents.\nlicense: Apache-2.0\nmetadata:\n  version: \"2.1.0\"\n  author: test\n---\n\n## Steps\nDo the thing.\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	st, err := ParseSkillDirectory(skillDir)
	if err != nil {
		t.Fatal(err)
	}

	if st.GetFields()["skillMarkdown"].GetStringValue() == "" {
		t.Fatal("skillMarkdown should be populated")
	}

	if st.GetFields()["name"].GetStringValue() != "pdf-processing" {
		t.Fatal("name mismatch")
	}

	meta := st.GetFields()["metadata"].GetStructValue()
	if meta == nil {
		t.Fatal("expected metadata")
	}

	if meta.GetFields()["version"].GetStringValue() != "2.1.0" {
		t.Fatalf("version in metadata: %v", meta.GetFields()["version"])
	}
}

func TestParseSkillDirectory_nameDirMismatchAllowedAtImporterLayer(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	skillDir := filepath.Join(dir, "wrong-dir")
	if err := os.MkdirAll(skillDir, 0o700); err != nil {
		t.Fatal(err)
	}

	content := "---\nname: right-name\ndescription: Some description.\n---\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	st, err := ParseSkillDirectory(skillDir)
	if err != nil {
		t.Fatalf("expected importer parsing to succeed: %v", err)
	}

	if got := st.GetFields()["name"].GetStringValue(); got != "right-name" {
		t.Fatalf("name = %q, want %q", got, "right-name")
	}
}

func TestParseSkillDirectory_notADirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	f := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(f, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := ParseSkillDirectory(f)
	if err == nil {
		t.Fatal("expected error")
	}
}
