// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agentcfg

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// skillFolderTarget is a skills-directory agent whose slug folder sits under
// root, plus that folder's path.
func skillFolderTarget(root string) (*SkillTarget, string) {
	dir := filepath.Join(root, "skills", "test-skill")

	return &SkillTarget{
		Strategy: SkillFolder,
		Path:     func(Env, string) (string, error) { return filepath.Join(dir, "SKILL.md"), nil },
	}, dir
}

// TestInstallSkillFolderReplacesABundleLeftByAnEarlierVersion is the orphan
// case a plain file write cannot fix: v1 shipped a bundle, v2 ships a single
// SKILL.md, and nothing afterwards would ever look at v1's scripts/ again.
func TestInstallSkillFolderReplacesABundleLeftByAnEarlierVersion(t *testing.T) {
	root := t.TempDir()
	target, dir := skillFolderTarget(root)

	// v1: a bundle.
	_, err := InstallSkillBundle(target, Env{}, "test-skill", skillBundleArchive(t), Global, false)
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(dir, "scripts", "run.sh"))

	// v2: a single-file skill.
	outcome, err := InstallSkill(target, Env{}, "test-skill", "# v2\n", Global, false)
	require.NoError(t, err)

	assert.Equal(t, ActionUpdated, outcome.Action)
	assert.NoDirExists(t, filepath.Join(dir, "scripts"))

	got, readErr := os.ReadFile(filepath.Join(dir, "SKILL.md"))
	require.NoError(t, readErr)
	assert.Equal(t, "# v2\n", string(got))
}

// TestInstallSkillFolderIsNotUnchangedWhileAStrayFileSurvives: the folder is
// the artifact, so an identical SKILL.md next to something else is not the
// desired state and must not report as already correct.
func TestInstallSkillFolderIsNotUnchangedWhileAStrayFileSurvives(t *testing.T) {
	root := t.TempDir()
	target, dir := skillFolderTarget(root)

	_, err := InstallSkill(target, Env{}, "test-skill", "# v1\n", Global, false)
	require.NoError(t, err)

	stray := filepath.Join(dir, "references", "api.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(stray), 0o750))
	require.NoError(t, os.WriteFile(stray, []byte("# old"), 0o600))

	// Same content, so only the stray file distinguishes this from a no-op.
	outcome, err := InstallSkill(target, Env{}, "test-skill", "# v1\n", Global, false)
	require.NoError(t, err)

	assert.Equal(t, ActionUpdated, outcome.Action)
	assert.NoDirExists(t, filepath.Join(dir, "references"))
	assert.FileExists(t, filepath.Join(dir, "SKILL.md"))
}

// TestInstallSkillFolderStaysIdempotentOnACleanFolder: the ordinary reinstall
// still writes nothing.
func TestInstallSkillFolderStaysIdempotentOnACleanFolder(t *testing.T) {
	root := t.TempDir()
	target, _ := skillFolderTarget(root)

	_, err := InstallSkill(target, Env{}, "test-skill", "# v1\n", Global, false)
	require.NoError(t, err)

	again, err := InstallSkill(target, Env{}, "test-skill", "# v1\n", Global, false)
	require.NoError(t, err)
	assert.Equal(t, ActionUnchanged, again.Action)
}

func TestInstallSkillFolderDryRunTouchesNothing(t *testing.T) {
	root := t.TempDir()
	target, dir := skillFolderTarget(root)

	_, err := InstallSkillBundle(target, Env{}, "test-skill", skillBundleArchive(t), Global, false)
	require.NoError(t, err)

	outcome, err := InstallSkill(target, Env{}, "test-skill", "# v2\n", Global, true)
	require.NoError(t, err)

	assert.Equal(t, ActionUpdated, outcome.Action)
	assert.FileExists(t, filepath.Join(dir, "scripts", "run.sh"), "a dry run must clear nothing")
}

// TestInstallSkillFolderAddsWhenOnlyStraysAreThere: a folder holding leftovers
// but no SKILL.md is not an install of ours to update.
func TestInstallSkillFolderAddsWhenOnlyStraysAreThere(t *testing.T) {
	root := t.TempDir()
	target, dir := skillFolderTarget(root)

	require.NoError(t, os.MkdirAll(filepath.Join(dir, "references"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "references", "api.md"), []byte("# old"), 0o600))

	outcome, err := InstallSkill(target, Env{}, "test-skill", "# v1\n", Global, false)
	require.NoError(t, err)

	assert.Equal(t, ActionAdded, outcome.Action)
	assert.NoDirExists(t, filepath.Join(dir, "references"))
	assert.FileExists(t, filepath.Join(dir, "SKILL.md"))
}
