// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package pkgstate_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/agntcy/dir/cli/internal/pkgstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func manifestPath(t *testing.T) string {
	t.Helper()

	return filepath.Join(t.TempDir(), pkgstate.ManifestFileName)
}

func sampleEntry() pkgstate.Entry {
	return pkgstate.Entry{
		Name:        "cisco.com/agent",
		Version:     "1.0.0",
		CID:         "bafyreibsample",
		Agent:       "claude-code",
		Scope:       pkgstate.ScopeGlobal,
		Origin:      pkgstate.OriginDirectory,
		InstalledAt: time.Date(2026, time.August, 14, 9, 30, 0, 0, time.UTC),
		SkillPath:   "/home/dev/.claude/skills/cisco.com-agent",
		SkillFiles:  []string{"SKILL.md", "references/api.md"},
		MCPServers:  []string{"agntcy-dir"},
	}
}

func TestLoadMissingFileIsEmpty(t *testing.T) {
	m, err := pkgstate.Load(manifestPath(t))
	require.NoError(t, err)
	assert.Empty(t, m.Entries)
	assert.False(t, m.Degraded())
	assert.False(t, m.Frozen())
}

func TestSaveThenLoadRoundTrip(t *testing.T) {
	path := manifestPath(t)
	entry := sampleEntry()

	m := &pkgstate.Manifest{}
	m.Upsert(entry)
	require.NoError(t, m.Save(path))

	loaded, err := pkgstate.Load(path)
	require.NoError(t, err)
	require.Len(t, loaded.Entries, 1)
	assert.Equal(t, entry, loaded.Entries[0])
}

func TestSaveWritesOwnerOnlyFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX file modes are not meaningful on Windows")
	}

	path := manifestPath(t)
	require.NoError(t, (&pkgstate.Manifest{}).Save(path))

	info, err := os.Stat(path)
	require.NoError(t, err)
	// The manifest lists every path dirctl wrote into this user's configs.
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestSaveCreatesMissingParentDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "dirctl", pkgstate.ManifestFileName)

	m := &pkgstate.Manifest{}
	m.Upsert(sampleEntry())
	require.NoError(t, m.Save(path))

	loaded, err := pkgstate.Load(path)
	require.NoError(t, err)
	assert.Len(t, loaded.Entries, 1)
}

func TestSaveLeavesNoTempFileBehind(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, pkgstate.ManifestFileName)

	m := &pkgstate.Manifest{}
	m.Upsert(sampleEntry())
	require.NoError(t, m.Save(path))
	require.NoError(t, m.Save(path))

	names, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, names, 1)
	assert.Equal(t, pkgstate.ManifestFileName, names[0].Name())
}

func TestSaveEmptyManifestWritesAnEmptyList(t *testing.T) {
	path := manifestPath(t)
	require.NoError(t, (&pkgstate.Manifest{}).Save(path))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	// Not `null`: a reader in any language sees a list it can iterate.
	assert.Contains(t, string(data), `"entries": []`)
	assert.Contains(t, string(data), `"schemaVersion": 1`)
}

func TestLoadCorruptFileDegradesAndStaysWritable(t *testing.T) {
	path := manifestPath(t)
	require.NoError(t, os.WriteFile(path, []byte("{not json"), 0o600))

	m, err := pkgstate.Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse install manifest")
	assert.Empty(t, m.Entries)
	assert.True(t, m.Degraded())
	// Its rows are already unreadable, so a fresh manifest may replace it.
	assert.False(t, m.Frozen())

	m.Upsert(sampleEntry())
	require.NoError(t, m.Save(path))

	loaded, err := pkgstate.Load(path)
	require.NoError(t, err)
	assert.Len(t, loaded.Entries, 1)
}

func TestLoadForwardSchemaVersionIsFrozen(t *testing.T) {
	path := manifestPath(t)
	future := []byte(`{"schemaVersion": 99, "entries": [{"name": "x", "agent": "claude-code"}]}`)
	require.NoError(t, os.WriteFile(path, future, 0o600))

	m, err := pkgstate.Load(path)
	require.Error(t, err)
	require.ErrorIs(t, err, pkgstate.ErrFrozen)
	assert.Contains(t, err.Error(), "upgrade dirctl")
	assert.Empty(t, m.Entries)
	assert.True(t, m.Degraded())
	assert.True(t, m.Frozen())

	// Saving would destroy rows this binary cannot represent, so it refuses and
	// the newer dirctl's file survives untouched.
	m.Upsert(sampleEntry())
	require.ErrorIs(t, m.Save(path), pkgstate.ErrFrozen)

	onDisk, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, future, onDisk)
}

func TestLoadAcceptsMissingSchemaVersion(t *testing.T) {
	// A hand-written file with no schemaVersion reads as the current format
	// rather than being thrown away.
	path := manifestPath(t)
	require.NoError(t, os.WriteFile(path,
		[]byte(`{"entries": [{"name": "x", "agent": "claude-code", "scope": "global"}]}`), 0o600))

	m, err := pkgstate.Load(path)
	require.NoError(t, err)
	require.Len(t, m.Entries, 1)
	assert.Equal(t, "x", m.Entries[0].Name)
}

func TestLoadUnreadableFileIsFrozen(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX file modes are not meaningful on Windows")
	}

	if os.Geteuid() == 0 {
		t.Skip("root bypasses file permissions")
	}

	path := manifestPath(t)
	require.NoError(t, os.WriteFile(path, []byte(`{"schemaVersion":1,"entries":[]}`), 0o000))

	m, err := pkgstate.Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read install manifest")
	// The contents are unknown, so overwriting them is not safe either.
	assert.True(t, m.Frozen())
}

func TestUpsertReplacesInPlaceOnTheSameKey(t *testing.T) {
	m := &pkgstate.Manifest{}
	m.Upsert(sampleEntry())
	m.Upsert(pkgstate.Entry{Name: "other", Agent: "cursor", Scope: pkgstate.ScopeGlobal})

	updated := sampleEntry()
	updated.Version = "2.0.0"
	m.Upsert(updated)

	require.Len(t, m.Entries, 2)
	// Replaced where it stood, so the file the user reads keeps its order.
	assert.Equal(t, "2.0.0", m.Entries[0].Version)
	assert.Equal(t, "other", m.Entries[1].Name)
}

func TestUpsertKeysOnNameAgentAndScope(t *testing.T) {
	m := &pkgstate.Manifest{}

	base := sampleEntry()
	m.Upsert(base)

	otherAgent := base
	otherAgent.Agent = "cursor"
	m.Upsert(otherAgent)

	oneRepo := base
	oneRepo.Scope = pkgstate.ProjectScope("/src/alpha")
	m.Upsert(oneRepo)

	anotherRepo := base
	anotherRepo.Scope = pkgstate.ProjectScope("/src/beta")
	m.Upsert(anotherRepo)

	// One package, four rows: two agents globally, plus one repository each.
	// Naming the repository is what keeps the last two apart — a bare
	// "project" would collide on one key.
	assert.Len(t, m.Entries, 4)
}

func TestWithCarriedArtifactsKeepsAServerThisInstallDidNotWrite(t *testing.T) {
	// A partly successful reinstall: the skill was written, the agent's MCP
	// config could not be read, so the new row names no server. The earlier
	// key is still in that config and only the row knows about it.
	prior := sampleEntry()

	fresh := sampleEntry()
	fresh.MCPServers = nil

	merged := fresh.WithCarriedArtifacts(prior)
	assert.Equal(t, []string{"agntcy-dir"}, merged.MCPServers)
}

func TestWithCarriedArtifactsUnionsRenamedServers(t *testing.T) {
	// A new version renamed its server. Plain install removes nothing first,
	// so both keys are on disk and the row must name both.
	prior := sampleEntry()

	fresh := sampleEntry()
	fresh.MCPServers = []string{"agntcy-dir-v2"}

	merged := fresh.WithCarriedArtifacts(prior)
	assert.Equal(t, []string{"agntcy-dir-v2", "agntcy-dir"}, merged.MCPServers)
}

func TestWithCarriedArtifactsDoesNotDuplicateServers(t *testing.T) {
	prior := sampleEntry()
	merged := sampleEntry().WithCarriedArtifacts(prior)
	assert.Equal(t, []string{"agntcy-dir"}, merged.MCPServers)
}

func TestWithCarriedArtifactsKeepsASkillThisInstallDidNotWrite(t *testing.T) {
	prior := sampleEntry()

	fresh := sampleEntry()
	fresh.SkillPath = ""
	fresh.SkillFiles = nil

	merged := fresh.WithCarriedArtifacts(prior)
	assert.Equal(t, prior.SkillPath, merged.SkillPath)
	assert.Equal(t, prior.SkillFiles, merged.SkillFiles)
}

func TestWithCarriedArtifactsPrefersTheNewSkill(t *testing.T) {
	prior := sampleEntry()

	fresh := sampleEntry()
	fresh.SkillPath = "/home/dev/.claude/skills/renamed"
	fresh.SkillFiles = []string{"SKILL.md"}

	merged := fresh.WithCarriedArtifacts(prior)
	assert.Equal(t, "/home/dev/.claude/skills/renamed", merged.SkillPath)
	assert.Equal(t, []string{"SKILL.md"}, merged.SkillFiles)
}

func TestWithCarriedArtifactsOnAFirstInstall(t *testing.T) {
	// No prior row, so there is nothing to carry.
	entry := sampleEntry()
	assert.Equal(t, entry, entry.WithCarriedArtifacts(pkgstate.Entry{}))
}

func TestRemove(t *testing.T) {
	m := &pkgstate.Manifest{}
	entry := sampleEntry()
	m.Upsert(entry)

	assert.False(t, m.Remove(pkgstate.Key{Name: "cisco.com/agent", Agent: "cursor", Scope: pkgstate.ScopeGlobal}))
	assert.Len(t, m.Entries, 1)

	assert.True(t, m.Remove(entry.Key()))
	assert.Empty(t, m.Entries)

	assert.False(t, m.Remove(entry.Key()))
}

func TestFind(t *testing.T) {
	m := &pkgstate.Manifest{}
	entry := sampleEntry()
	m.Upsert(entry)

	got, ok := m.Find(entry.Key())
	require.True(t, ok)
	assert.Equal(t, entry, got)

	_, ok = m.Find(pkgstate.Key{Name: "missing", Agent: "cursor", Scope: pkgstate.ScopeGlobal})
	assert.False(t, ok)
}

func TestByName(t *testing.T) {
	m := &pkgstate.Manifest{}

	base := sampleEntry()
	m.Upsert(base)

	cursor := base
	cursor.Agent = "cursor"
	m.Upsert(cursor)

	m.Upsert(pkgstate.Entry{Name: "other", Agent: "claude-code", Scope: pkgstate.ScopeGlobal})

	rows := m.ByName("cisco.com/agent")
	require.Len(t, rows, 2)
	assert.Equal(t, "claude-code", rows[0].Agent)
	assert.Equal(t, "cursor", rows[1].Agent)

	assert.Empty(t, m.ByName("nothing-here"))
}

func TestDefaultPathUsesXDGConfigHome(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join("/tmp", "xdg"))

	path, err := pkgstate.DefaultPath()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/tmp", "xdg", "dirctl", "installed.json"), path)
}

func TestDefaultPathFallsBackToHomeConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")

	path, err := pkgstate.DefaultPath()
	require.NoError(t, err)

	home, err := os.UserHomeDir()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(home, ".config", "dirctl", "installed.json"), path)
}

func TestKindIsDerivedFromTheArtifactsThatLanded(t *testing.T) {
	both := pkgstate.Entry{SkillPath: "/skills/agent", MCPServers: []string{"agent"}}
	assert.Equal(t, "skill+mcp", both.Kind())

	assert.Equal(t, "skill", pkgstate.Entry{SkillPath: "/skills/agent"}.Kind())
	assert.Equal(t, "mcp", pkgstate.Entry{MCPServers: []string{"agent"}}.Kind())
	// Only a hand-edited manifest can hold a row that landed nothing.
	assert.Equal(t, "none", pkgstate.Entry{}.Kind())
}

func TestOnlyTwoNamedContextsThatDifferAreAMismatch(t *testing.T) {
	// Rows written before the field existed carry no context, and must stay
	// checkable rather than become permanently unassessable.
	assert.True(t, pkgstate.Entry{}.ContextMatches("local"))
	// A dirctl pointed at a server through DIRECTORY_CLIENT_SERVER_ADDRESS
	// alone has no context name, and must not skip every row it has.
	assert.True(t, pkgstate.Entry{Context: "staging"}.ContextMatches(""))
	assert.True(t, pkgstate.Entry{Context: "local"}.ContextMatches("local"))
	assert.False(t, pkgstate.Entry{Context: "staging"}.ContextMatches("local"))
}

func TestContextRoundTripsThroughTheFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "installed.json")

	m := &pkgstate.Manifest{Entries: []pkgstate.Entry{{
		Name: "cisco.com/agent", Agent: "claude-code", Scope: pkgstate.ScopeGlobal, Context: "staging",
	}}}
	require.NoError(t, m.Save(path))

	loaded, err := pkgstate.Load(path)
	require.NoError(t, err)
	require.Len(t, loaded.Entries, 1)
	assert.Equal(t, "staging", loaded.Entries[0].Context)
}
