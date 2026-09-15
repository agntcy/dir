// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	oasfv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/cli/internal/agentcfg"
	"github.com/agntcy/dir/cli/internal/pkgstate"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// isolateManifest points DefaultPath at a temp directory, so these tests never
// read or write the developer's real manifest.
func isolateManifest(t *testing.T) string {
	t.Helper()

	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	return filepath.Join(configHome, "dirctl", pkgstate.ManifestFileName)
}

// testCmd is a bare command whose output and errors are captured.
func testCmd(t *testing.T) (*cobra.Command, *bytes.Buffer) {
	t.Helper()

	var errOut bytes.Buffer

	cmd := &cobra.Command{Use: "test"}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&errOut)

	return cmd, &errOut
}

var (
	claudeCode = agentcfg.Agent{ID: "claude-code", Name: "Claude Code"}
	cursor     = agentcfg.Agent{ID: "cursor", Name: "Cursor"}
)

func testRecord(name, version string) *corev1.Record {
	return corev1.New(&oasfv1alpha1.Record{Name: name, Version: version})
}

func TestRecordInstallsWritesOneRowPerAgentThatGotSomething(t *testing.T) {
	path := isolateManifest(t)
	cmd, errOut := testCmd(t)

	rec := testRecord("cisco.com/agent", "1.0.0")
	item := applied{
		record: rec,
		pinned: true,
		outcomes: []agentcfg.Outcome{
			{Agent: "Claude Code", Artifact: agentcfg.ArtifactSkill, Path: "/skills/agent", Action: agentcfg.ActionAdded},
			// Cursor got nothing, so it must not appear in the manifest.
			{Agent: "Cursor", Artifact: agentcfg.ArtifactSkill, Action: agentcfg.ActionSkipped, Reason: "no global location"},
		},
	}

	recordInstalls(cmd, []applied{item}, []agentcfg.Agent{claudeCode, cursor}, agentcfg.Global)

	m, err := pkgstate.Load(path)
	require.NoError(t, err)
	require.Len(t, m.Entries, 1)

	entry := m.Entries[0]
	assert.Equal(t, "cisco.com/agent", entry.Name)
	assert.Equal(t, "1.0.0", entry.Version)
	assert.Equal(t, rec.GetCid(), entry.CID)
	assert.Equal(t, "claude-code", entry.Agent)
	assert.Equal(t, pkgstate.ScopeGlobal, entry.Scope)
	assert.Equal(t, pkgstate.OriginDirectory, entry.Origin)
	assert.True(t, entry.Pinned)
	assert.Equal(t, "/skills/agent", entry.SkillPath)
	assert.False(t, entry.InstalledAt.IsZero())
	assert.Empty(t, errOut.String())
}

func TestRecordInstallsReplacesTheRowOnReinstall(t *testing.T) {
	path := isolateManifest(t)
	cmd, _ := testCmd(t)

	outcomes := []agentcfg.Outcome{
		{Agent: "Claude Code", Artifact: agentcfg.ArtifactSkill, Path: "/skills/agent", Action: agentcfg.ActionAdded},
	}
	agents := []agentcfg.Agent{claudeCode}

	recordInstalls(cmd, []applied{{record: testRecord("cisco.com/agent", "1.0.0"), outcomes: outcomes}}, agents, agentcfg.Global)
	recordInstalls(cmd, []applied{{record: testRecord("cisco.com/agent", "2.0.0"), outcomes: outcomes}}, agents, agentcfg.Global)

	m, err := pkgstate.Load(path)
	require.NoError(t, err)
	require.Len(t, m.Entries, 1)
	assert.Equal(t, "2.0.0", m.Entries[0].Version)
}

func TestRecordInstallsCarriesArtifactsThroughAPartialReinstall(t *testing.T) {
	path := isolateManifest(t)
	cmd, _ := testCmd(t)

	rec := testRecord("cisco.com/agent", "1.0.0")
	agents := []agentcfg.Agent{claudeCode}

	recordInstalls(cmd, []applied{{record: rec, outcomes: []agentcfg.Outcome{
		{Agent: "Claude Code", Artifact: agentcfg.ArtifactSkill, Path: "/skills/agent", Action: agentcfg.ActionAdded},
		{Agent: "Claude Code", Artifact: agentcfg.ArtifactMCP, Server: "agntcy-dir", Action: agentcfg.ActionAdded},
	}}}, agents, agentcfg.Global)

	// Reinstall where the MCP config could not be written. The key is still in
	// that config, so the row must keep naming it.
	recordInstalls(cmd, []applied{{record: testRecord("cisco.com/agent", "2.0.0"), outcomes: []agentcfg.Outcome{
		{Agent: "Claude Code", Artifact: agentcfg.ArtifactSkill, Path: "/skills/agent", Action: agentcfg.ActionUpdated},
		{Agent: "Claude Code", Artifact: agentcfg.ArtifactMCP, Server: "agntcy-dir", Action: agentcfg.ActionFailed},
	}}}, agents, agentcfg.Global)

	m, err := pkgstate.Load(path)
	require.NoError(t, err)
	require.Len(t, m.Entries, 1)
	assert.Equal(t, "2.0.0", m.Entries[0].Version)
	assert.Equal(t, "/skills/agent", m.Entries[0].SkillPath)
	assert.Equal(t, []string{"agntcy-dir"}, m.Entries[0].MCPServers)
}

func TestRecordInstallsSkipsANamelessRecord(t *testing.T) {
	path := isolateManifest(t)
	cmd, _ := testCmd(t)

	// Installable by CID, but there is nothing to key a row on.
	recordInstalls(cmd, []applied{{
		record: testRecord("", "1.0.0"),
		outcomes: []agentcfg.Outcome{
			{Agent: "Claude Code", Artifact: agentcfg.ArtifactSkill, Path: "/skills/x", Action: agentcfg.ActionAdded},
		},
	}}, []agentcfg.Agent{claudeCode}, agentcfg.Global)

	_, err := os.Stat(path)
	assert.True(t, os.IsNotExist(err), "nothing changed, so no manifest is written")
}

func TestRecordInstallsNamesTheRepositoryForAProjectInstall(t *testing.T) {
	path := isolateManifest(t)
	cmd, _ := testCmd(t)

	recordInstalls(cmd, []applied{{
		record: testRecord("cisco.com/agent", "1.0.0"),
		outcomes: []agentcfg.Outcome{
			{Agent: "Claude Code", Artifact: agentcfg.ArtifactSkill, Path: "/skills/x", Action: agentcfg.ActionAdded},
		},
	}}, []agentcfg.Agent{claudeCode}, agentcfg.Project)

	m, err := pkgstate.Load(path)
	require.NoError(t, err)
	require.Len(t, m.Entries, 1)

	cwd, err := os.Getwd()
	require.NoError(t, err)
	// The repository, not the word "project": two repositories must not
	// collide on one (name, agent, scope) key.
	assert.Equal(t, pkgstate.ProjectScope(cwd), m.Entries[0].Scope)
}

func TestRecordUninstallsDropsTheRow(t *testing.T) {
	path := isolateManifest(t)
	cmd, _ := testCmd(t)

	rec := testRecord("cisco.com/agent", "1.0.0")
	agents := []agentcfg.Agent{claudeCode, cursor}

	recordInstalls(cmd, []applied{{record: rec, outcomes: []agentcfg.Outcome{
		{Agent: "Claude Code", Artifact: agentcfg.ArtifactSkill, Path: "/claude/skills/x", Action: agentcfg.ActionAdded},
		{Agent: "Cursor", Artifact: agentcfg.ArtifactSkill, Path: "/cursor/skills/x", Action: agentcfg.ActionAdded},
	}}}, agents, agentcfg.Global)

	recordUninstalls(cmd, []applied{{record: rec, outcomes: []agentcfg.Outcome{
		{Agent: "Claude Code", Artifact: agentcfg.ArtifactSkill, Path: "/claude/skills/x", Action: agentcfg.ActionRemoved},
		// Cursor's removal failed, so its artifacts are still on disk and its
		// row has to stay.
		{Agent: "Cursor", Artifact: agentcfg.ArtifactSkill, Path: "/cursor/skills/x", Action: agentcfg.ActionFailed},
	}}}, agents, agentcfg.Global)

	m, err := pkgstate.Load(path)
	require.NoError(t, err)
	require.Len(t, m.Entries, 1)
	assert.Equal(t, "cursor", m.Entries[0].Agent)
}

func TestWithManifestWarnsOnCorruptFileAndCarriesOn(t *testing.T) {
	path := isolateManifest(t)
	cmd, errOut := testCmd(t)

	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte("{not json"), 0o600))

	recordInstalls(cmd, []applied{{
		record: testRecord("cisco.com/agent", "1.0.0"),
		outcomes: []agentcfg.Outcome{
			{Agent: "Claude Code", Artifact: agentcfg.ArtifactSkill, Path: "/skills/x", Action: agentcfg.ActionAdded},
		},
	}}, []agentcfg.Agent{claudeCode}, agentcfg.Global)

	// The user is told, and the install still gets recorded: bookkeeping never
	// fails an install.
	assert.Contains(t, errOut.String(), "Warning: install manifest")

	m, err := pkgstate.Load(path)
	require.NoError(t, err)
	assert.Len(t, m.Entries, 1)
}

func TestWithManifestLeavesANewerDirctlsFileAlone(t *testing.T) {
	path := isolateManifest(t)
	cmd, errOut := testCmd(t)

	future := []byte(`{"schemaVersion": 99, "entries": []}`)

	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, future, 0o600))

	recordInstalls(cmd, []applied{{
		record: testRecord("cisco.com/agent", "1.0.0"),
		outcomes: []agentcfg.Outcome{
			{Agent: "Claude Code", Artifact: agentcfg.ArtifactSkill, Path: "/skills/x", Action: agentcfg.ActionAdded},
		},
	}}, []agentcfg.Agent{claudeCode}, agentcfg.Global)

	assert.Contains(t, errOut.String(), "upgrade dirctl")
	// Exactly one warning, not one from Load and another from Save.
	assert.Equal(t, 1, bytes.Count(errOut.Bytes(), []byte("Warning:")))

	onDisk, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, future, onDisk)
}

func TestManifestScopeNamesTheRepositoryForAProjectInstall(t *testing.T) {
	assert.Equal(t, pkgstate.ScopeGlobal, manifestScope(agentcfg.Global))

	cwd, err := os.Getwd()
	require.NoError(t, err)
	assert.Equal(t, pkgstate.ProjectScope(cwd), manifestScope(agentcfg.Project))
}
