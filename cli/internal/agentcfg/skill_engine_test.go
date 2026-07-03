// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agentcfg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstallSkillFolderWritesVerbatim(t *testing.T) {
	root := t.TempDir()
	skillPath := filepath.Join(root, "skills", "test-skill", "SKILL.md")

	target := &SkillTarget{
		Strategy: SkillFolder,
		Path:     func(Env, string) (string, error) { return skillPath, nil },
	}

	outcome, err := InstallSkill(target, Env{}, "test-skill", sampleDoc, false)
	require.NoError(t, err)
	assert.Equal(t, ActionAdded, outcome.Action)

	got, err := os.ReadFile(skillPath)
	require.NoError(t, err)
	assert.Equal(t, sampleDoc, string(got))

	// Idempotent.
	again, err := InstallSkill(target, Env{}, "test-skill", sampleDoc, false)
	require.NoError(t, err)
	assert.Equal(t, ActionUnchanged, again.Action)

	// Remove deletes the test-skill folder.
	rm, err := RemoveSkill(target, Env{}, "test-skill", false)
	require.NoError(t, err)
	assert.Equal(t, ActionRemoved, rm.Action)

	_, statErr := os.Stat(filepath.Dir(skillPath))
	assert.True(t, os.IsNotExist(statErr))
}

func TestInstallSkillDedicatedFileRendersAndRemoves(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "rules", "test-skill.md")

	target := &SkillTarget{
		Strategy: DedicatedFile,
		Path:     func(Env, string) (string, error) { return path, nil },
		Render:   renderContinue,
	}

	outcome, err := InstallSkill(target, Env{}, "test-skill", sampleDoc, false)
	require.NoError(t, err)
	assert.Equal(t, ActionAdded, outcome.Action)

	got, _ := os.ReadFile(path)
	assert.Contains(t, string(got), "alwaysApply: true")

	rm, err := RemoveSkill(target, Env{}, "test-skill", false)
	require.NoError(t, err)
	assert.Equal(t, ActionRemoved, rm.Action)

	_, statErr := os.Stat(path)
	assert.True(t, os.IsNotExist(statErr))
}

func TestInstallSkillManagedBlockPreservesUserContent(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "GEMINI.md")
	require.NoError(t, os.WriteFile(path, []byte("# My notes\n\nKeep me.\n"), 0o600))

	target := &SkillTarget{
		Strategy: ManagedBlock,
		Path:     func(Env, string) (string, error) { return path, nil },
		Render:   renderManagedInner,
	}

	outcome, err := InstallSkill(target, Env{}, "test-skill", sampleDoc, false)
	require.NoError(t, err)
	assert.Equal(t, ActionAdded, outcome.Action)

	got, _ := os.ReadFile(path)
	assert.Contains(t, string(got), "Keep me.")
	assert.Contains(t, string(got), blockBegin("test-skill"))

	// Idempotent.
	again, err := InstallSkill(target, Env{}, "test-skill", sampleDoc, false)
	require.NoError(t, err)
	assert.Equal(t, ActionUnchanged, again.Action)

	// Remove strips only our block.
	rm, err := RemoveSkill(target, Env{}, "test-skill", false)
	require.NoError(t, err)
	assert.Equal(t, ActionRemoved, rm.Action)

	after, _ := os.ReadFile(path)
	assert.Contains(t, string(after), "Keep me.")
	assert.NotContains(t, string(after), blockBegin("test-skill"))
}

func TestInstallSkillProjectFallback(t *testing.T) {
	root := t.TempDir()
	projectPath := filepath.Join(root, "repo", ".cursor", "rules", "test-skill.mdc")

	target := &SkillTarget{
		Strategy:    DedicatedFile,
		Path:        func(Env, string) (string, error) { return "", ErrNoGlobalPath },
		ProjectPath: func(Env, string) (string, error) { return projectPath, nil },
		Render:      renderCursor,
	}

	outcome, err := InstallSkill(target, Env{}, "test-skill", sampleDoc, false)
	require.NoError(t, err)
	assert.Equal(t, ActionAdded, outcome.Action)
	assert.Equal(t, projectPath, outcome.Path)
	assert.NotEmpty(t, outcome.Reason, "project fallback should be explained")

	_, statErr := os.Stat(projectPath)
	assert.NoError(t, statErr)
}

func TestInstallSkillDryRunWritesNothing(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "test-skill.md")

	target := &SkillTarget{
		Strategy: DedicatedFile,
		Path:     func(Env, string) (string, error) { return path, nil },
		Render:   renderRoo,
	}

	outcome, err := InstallSkill(target, Env{}, "test-skill", sampleDoc, true)
	require.NoError(t, err)
	assert.Equal(t, ActionAdded, outcome.Action)

	_, statErr := os.Stat(path)
	assert.True(t, os.IsNotExist(statErr))
}

func TestRemoveSkillAbsentIsUnchanged(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "test-skill.md")

	target := &SkillTarget{
		Strategy: DedicatedFile,
		Path:     func(Env, string) (string, error) { return path, nil },
		Render:   renderRoo,
	}

	outcome, err := RemoveSkill(target, Env{}, "test-skill", false)
	require.NoError(t, err)
	assert.Equal(t, ActionUnchanged, outcome.Action)
}

func TestInstallSkillVersionReplace(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "rules", "rec.md")

	target := &SkillTarget{
		Strategy: DedicatedFile,
		Path:     func(Env, string) (string, error) { return path, nil },
		Render:   renderContinue,
	}

	canonicalA := "# Version A\n\nThis is version A.\n"
	canonicalB := "# Version B\n\nThis is version B.\n"

	// Install canonical A.
	first, err := InstallSkill(target, Env{}, "rec", canonicalA, false)
	require.NoError(t, err)
	assert.Equal(t, ActionAdded, first.Action)

	// Install canonical B — same slug, different content.
	second, err := InstallSkill(target, Env{}, "rec", canonicalB, false)
	require.NoError(t, err)
	assert.Equal(t, ActionUpdated, second.Action, "second install of same slug should report ActionUpdated")

	got, err := os.ReadFile(path)
	require.NoError(t, err)

	renderedB, err := renderContinue(canonicalB)
	require.NoError(t, err)
	assert.Equal(t, string(renderedB), string(got), "file content should equal the render of canonical B")
}

func TestInstallSkillManagedBlockVersionReplace(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "AGENTS.md")

	target := &SkillTarget{
		Strategy: ManagedBlock,
		Path:     func(Env, string) (string, error) { return path, nil },
		Render:   renderManagedInner,
	}

	canonicalA := "---\nname: rec\n---\n\n# Version A\n\nThis is version A.\n"
	canonicalB := "---\nname: rec\n---\n\n# Version B\n\nThis is version B.\n"

	// Install canonical A into a fresh file.
	first, err := InstallSkill(target, Env{}, "rec", canonicalA, false)
	require.NoError(t, err)
	assert.Equal(t, ActionAdded, first.Action)

	// Install the SAME slug with a DIFFERENT canonical: slug-scoped block
	// detection must recognize our existing block and replace it in place.
	second, err := InstallSkill(target, Env{}, "rec", canonicalB, false)
	require.NoError(t, err)
	assert.Equal(t, ActionUpdated, second.Action, "second install of same slug should report ActionUpdated")

	got, err := os.ReadFile(path)
	require.NoError(t, err)

	s := string(got)

	// Exactly one managed block for slug "rec" (replaced, not appended).
	assert.Equal(t, 1, strings.Count(s, blockBegin("rec")), "should have exactly one block marker")
	// The updated body from B is present; A's body is gone.
	assert.Contains(t, s, "# Version B")
	assert.NotContains(t, s, "# Version A")
}
