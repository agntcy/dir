// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agentinstall

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"github.com/agntcy/dir/cli/internal/agentcfg"
	"github.com/agntcy/dir/cli/internal/pkgstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// bundleWith builds a gzip skill bundle carrying the given files.
func bundleWith(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buf bytes.Buffer

	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	// SKILL.md first, so the archive always has its manifest up front.
	names := []string{"SKILL.md"}

	for name := range files {
		if name != "SKILL.md" {
			names = append(names, name)
		}
	}

	for _, name := range names {
		content, ok := files[name]
		if !ok {
			continue
		}

		data := []byte(content)
		require.NoError(t, tw.WriteHeader(&tar.Header{
			Name: name, Mode: 0o600, Size: int64(len(data)), Typeflag: tar.TypeReg,
		}))

		_, err := tw.Write(data)
		require.NoError(t, err)
	}

	require.NoError(t, tw.Close())
	require.NoError(t, gzw.Close())

	return buf.Bytes()
}

// bundleArtifacts is a record shipping a skill bundle plus one MCP server.
func bundleArtifacts(t *testing.T, server string, files map[string]string) Artifacts {
	t.Helper()

	archive := bundleWith(t, files)

	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}

	arts := Artifacts{
		slug:        "cisco.com-agent",
		skill:       files["SKILL.md"],
		skillBundle: archive,
		skillFiles:  names,
	}

	if server != "" {
		arts.mcpServers = []mcpServer{{name: server, entry: map[string]any{"command": "dirctl"}}}
	}

	return arts
}

// upgradeIdentity is the row identity for a package pulled from a Directory.
func upgradeIdentity(version, cid string) Identity {
	return Identity{
		Name:      "cisco.com/agent",
		Version:   version,
		CID:       cid,
		Scope:     pkgstate.ScopeGlobal,
		Origin:    pkgstate.OriginDirectory,
		Directory: "dir.example:8888",
	}
}

// TestUpgradeLeavesNoOrphans drives the whole reconcile against the real
// engines, covering both orphan cases at once: v2 drops a bundle file and
// renames its MCP server.
func TestUpgradeLeavesNoOrphans(t *testing.T) {
	home := t.TempDir()
	env := agentcfg.Env{Home: home, GOOS: "linux", Cwd: home}

	agent, known := agentcfg.ByID(claudeCodeID)
	require.True(t, known)

	agents := []agentcfg.Agent{agent}
	m := &pkgstate.Manifest{}

	// v1: a bundle with a reference file, under the MCP key "agntcy-dir".
	v1 := bundleArtifacts(t, "agntcy-dir", map[string]string{
		"SKILL.md":            "# v1\n",
		"references/api.md":   "# api\n",
		"references/extra.md": "# extra\n",
	})

	installed := Install(env, v1, agents, agentcfg.Global, false)
	require.True(t, Record(m, v1, agents, installed, upgradeIdentity("1.0.0", "bafyone"), installedAt))

	skillDir := filepath.Join(home, ".claude", "skills", "cisco.com-agent")
	require.FileExists(t, filepath.Join(skillDir, "references", "api.md"))

	config := filepath.Join(home, ".claude.json")
	requireConfigHas(t, config, "agntcy-dir")

	// v2: one reference file dropped, and the server renamed.
	v2 := bundleArtifacts(t, "agntcy-directory", map[string]string{
		"SKILL.md":          "# v2\n",
		"references/api.md": "# api v2\n",
	})

	rows := m.ByName("cisco.com/agent")
	require.Len(t, rows, 1)

	removed := RemoveOrphans(env, rows, v2, false)
	written := Install(env, v2, agents, agentcfg.Global, false)
	require.True(t, Reconcile(m, v2, agents, removed, written, upgradeIdentity("2.0.0", "bafytwo"), installedAt))

	// The dropped reference file is gone, and the kept one moved on.
	assert.NoFileExists(t, filepath.Join(skillDir, "references", "extra.md"))

	api, err := os.ReadFile(filepath.Join(skillDir, "references", "api.md"))
	require.NoError(t, err)
	assert.Equal(t, "# api v2\n", string(api))

	// The renamed server replaced the old key rather than joining it.
	data, err := os.ReadFile(config)
	require.NoError(t, err)
	assert.NotContains(t, string(data), `"agntcy-dir"`)
	assert.Contains(t, string(data), "agntcy-directory")

	// And the row says exactly that.
	rows = m.ByName("cisco.com/agent")
	require.Len(t, rows, 1)
	assert.Equal(t, "2.0.0", rows[0].Version)
	assert.Equal(t, "bafytwo", rows[0].CID)
	assert.Equal(t, []string{"agntcy-directory"}, rows[0].MCPServers,
		"the old key is gone from the config, so the row must stop naming it")
	assert.ElementsMatch(t, []string{"SKILL.md", "references/api.md"}, rows[0].SkillFiles)
}

// TestUpgradeToASingleFileSkillClearsTheBundle: the folder is dirctl's, so
// installing a plain SKILL.md over a bundle replaces the whole folder.
func TestUpgradeToASingleFileSkillClearsTheBundle(t *testing.T) {
	home := t.TempDir()
	env := agentcfg.Env{Home: home, GOOS: "linux", Cwd: home}

	agent, known := agentcfg.ByID(claudeCodeID)
	require.True(t, known)

	agents := []agentcfg.Agent{agent}

	v1 := bundleArtifacts(t, "", map[string]string{
		"SKILL.md":          "# v1\n",
		"references/api.md": "# api\n",
	})
	Install(env, v1, agents, agentcfg.Global, false)

	skillDir := filepath.Join(home, ".claude", "skills", "cisco.com-agent")
	require.FileExists(t, filepath.Join(skillDir, "references", "api.md"))

	v2 := Artifacts{slug: "cisco.com-agent", skill: "# v2\n"}
	written := Install(env, v2, agents, agentcfg.Global, false)

	require.Len(t, written, 1)
	assert.Equal(t, agentcfg.ActionUpdated, written[0].Action)
	assert.NoDirExists(t, filepath.Join(skillDir, "references"))

	got, err := os.ReadFile(filepath.Join(skillDir, "SKILL.md"))
	require.NoError(t, err)
	assert.Equal(t, "# v2\n", string(got))
}

// TestUpgradeDropsARowWhoseArtifactsAreAllGone: v2 ships neither the skill nor
// the server v1 had, so nothing of the package is left to record.
func TestUpgradeDropsARowWhoseArtifactsAreAllGone(t *testing.T) {
	home := t.TempDir()
	env := agentcfg.Env{Home: home, GOOS: "linux", Cwd: home}

	agent, known := agentcfg.ByID(claudeCodeID)
	require.True(t, known)

	agents := []agentcfg.Agent{agent}
	m := &pkgstate.Manifest{}

	v1 := bundleArtifacts(t, "agntcy-dir", map[string]string{"SKILL.md": "# v1\n"})
	installed := Install(env, v1, agents, agentcfg.Global, false)
	Record(m, v1, agents, installed, upgradeIdentity("1.0.0", "bafyone"), installedAt)

	// v2 carries a different server and no skill at all.
	v2 := serverArtifacts("something-else")

	rows := m.ByName("cisco.com/agent")
	removed := RemoveOrphans(env, rows, v2, false)
	written := Install(env, v2, agents, agentcfg.Global, false)
	Reconcile(m, v2, agents, removed, written, upgradeIdentity("2.0.0", "bafytwo"), installedAt)

	assert.NoDirExists(t, filepath.Join(home, ".claude", "skills", "cisco.com-agent"))

	rows = m.ByName("cisco.com/agent")
	require.Len(t, rows, 1, "v2's own server landed, so the package is still installed")
	assert.Equal(t, "mcp", rows[0].Kind())
	assert.Equal(t, []string{"something-else"}, rows[0].MCPServers)
	assert.Empty(t, rows[0].SkillPath)
	assert.Empty(t, rows[0].SkillFiles)
}

// TestUpgradeDryRunTouchesNothing: the plan must be shown without moving a
// working install.
func TestUpgradeDryRunTouchesNothing(t *testing.T) {
	home := t.TempDir()
	env := agentcfg.Env{Home: home, GOOS: "linux", Cwd: home}

	agent, known := agentcfg.ByID(claudeCodeID)
	require.True(t, known)

	agents := []agentcfg.Agent{agent}

	v1 := bundleArtifacts(t, "agntcy-dir", map[string]string{
		"SKILL.md":          "# v1\n",
		"references/api.md": "# api\n",
	})
	Install(env, v1, agents, agentcfg.Global, false)

	v2 := bundleArtifacts(t, "agntcy-directory", map[string]string{"SKILL.md": "# v2\n"})

	row := recordedEntry("cisco.com/agent", claudeCodeID)
	row.SkillPath = filepath.Join(home, ".claude", "skills", "cisco.com-agent")
	row.MCPServers = []string{"agntcy-dir"}

	removed := RemoveOrphans(env, []pkgstate.Entry{row}, v2, true)
	written := Install(env, v2, agents, agentcfg.Global, true)

	require.NotEmpty(t, removed)
	require.NotEmpty(t, written)

	assert.FileExists(t, filepath.Join(row.SkillPath, "references", "api.md"))
	requireConfigHas(t, filepath.Join(home, ".claude.json"), "agntcy-dir")
}

func requireConfigHas(t *testing.T, path, key string) {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Contains(t, string(data), key)
}
