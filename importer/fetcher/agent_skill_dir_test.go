// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package fetcher

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/agntcy/dir/importer/types"
)

func TestAgentSkillDirFetcher_Fetch(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	skillDir := filepath.Join(dir, "code-review")
	if err := os.MkdirAll(skillDir, 0o700); err != nil {
		t.Fatal(err)
	}

	md := "---\nname: code-review\ndescription: Review code for bugs and style. Use when the user asks for a code review.\n---\n\nBody here.\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(md), 0o600); err != nil {
		t.Fatal(err)
	}

	f, err := NewAgentSkillDirFetcher(skillDir)
	if err != nil {
		t.Fatal(err)
	}

	outCh, errCh := f.Fetch(context.Background())

	var n int

	for item := range outCh {
		if item.Kind != types.SourceKindAgentSkill {
			t.Fatalf("Kind = %v, want AgentSkill", item.Kind)
		}

		if item.Skill == nil || item.Skill.GetFields()["name"].GetStringValue() != "code-review" {
			t.Fatal("unexpected skill payload")
		}

		if item.Skill.GetFields()["skillMarkdown"].GetStringValue() == "" {
			t.Fatal("expected wrapped skillMarkdown payload")
		}

		n++
	}

	for e := range errCh {
		if e != nil {
			t.Fatalf("unexpected err: %v", e)
		}
	}

	if n != 1 {
		t.Fatalf("got %d items, want 1", n)
	}
}
