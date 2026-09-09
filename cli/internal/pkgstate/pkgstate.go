// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package pkgstate owns dirctl's install manifest: the record of what
// `dirctl install` wrote, where, and for which agent.
//
// Installed artifacts carry no provenance of their own — an Agent Skill is
// marked only with its slug, with no version and no CID — so installed state
// cannot be reconstructed from disk. This file is therefore the only place that
// can answer "what is installed, and is any of it stale?", and the only place
// that will be able to say what to remove once removal stops re-deriving
// artifacts from the record.
//
// Two rules shape the API:
//
//   - The manifest is bookkeeping, not the product. Losing it must never fail
//     an install, so every failure here is a value the caller can warn about
//     and carry on past.
//   - The path is injected. Load and Save take one and DefaultPath is the only
//     function that reads the environment, so tests never touch a real home
//     directory. This is the pattern agentcfg.Env already sets.
//
// Readers must also tolerate drift. A user can edit or delete an artifact
// behind our back, so anything reporting on a row has to stat what it reports
// rather than trust the row alone.
package pkgstate

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/agntcy/dir/cli/internal/fsutil"
)

const (
	// SchemaVersion is the manifest format this binary writes and understands.
	SchemaVersion = 1

	// ManifestFileName is the manifest's file name.
	ManifestFileName = "installed.json"

	// configDirName matches the directory the reusable client config already
	// uses, so dirctl keeps all of its state in one place.
	configDirName = "dirctl"

	// manifestFilePerm keeps the manifest readable only by its owner: it lists
	// every path dirctl has written into this user's agent configs.
	manifestFilePerm = 0o600
)

// ErrFrozen is returned by Save when the file on disk was written by a newer
// dirctl. Overwriting it would destroy rows this binary cannot represent, so
// Save refuses instead.
var ErrFrozen = errors.New("install manifest was written by a newer dirctl")

// Origin says where a row's upstream lives, which decides how it is
// version-checked.
type Origin string

const (
	// OriginDirectory is a record pulled from a Directory, so its upstream is
	// the Directory and a version check is a Resolve call.
	OriginDirectory Origin = "directory"

	// OriginBuiltin is a record dirctl builds from content compiled into the
	// binary, so its upstream is the binary itself.
	OriginBuiltin Origin = "builtin"
)

// Scope says which configuration location a row's artifacts were written to.
type Scope string

const (
	// ScopeGlobal is the user's global agent config.
	ScopeGlobal Scope = "global"

	// ScopeProject is the current repository.
	ScopeProject Scope = "project"
)

// Key identifies one row: one package, for one agent, at one scope.
type Key struct {
	Name  string
	Agent string
	Scope Scope
}

// Entry is one installed package, for one agent, at one scope.
//
// The artifact fields record what was actually written, not what the record's
// modules imply. Nothing reads them yet: today's uninstall re-derives artifacts
// from the record and then deletes rows. They exist so that manifest-driven
// uninstall and upgrade can remove exactly these files and server keys without
// re-fetching a record that may since have been garbage-collected upstream.
//
// There is deliberately no artifact "type" field: one record can yield both a
// skill and MCP servers, so the kind is derived from these fields for display.
type Entry struct {
	// Name is the record name, as pushed to the Directory.
	Name string `json:"name"`

	// Version is the record version that was installed, spelled as the record
	// spells it. Compare it through cli/util/pkgver, never as text.
	Version string `json:"version,omitempty"`

	// CID is the content address of the exact record that was installed.
	CID string `json:"cid,omitempty"`

	// Agent is the agentcfg agent ID, e.g. "claude-code".
	Agent string `json:"agent"`

	// Scope is where the artifacts landed.
	Scope Scope `json:"scope"`

	// Origin is where to look for a newer version.
	Origin Origin `json:"origin"`

	// Pinned holds the package at Version, so a bare upgrade skips it.
	Pinned bool `json:"pinned,omitempty"`

	// InstalledAt is when this row was last written, in UTC.
	InstalledAt time.Time `json:"installedAt"`

	// SkillPath is the file or directory the skill landed in. Empty when the
	// record installed no skill for this agent.
	SkillPath string `json:"skillPath,omitempty"`

	// SkillFiles are a skill bundle's files, relative to SkillPath. Empty for a
	// single-file skill, where SkillPath is itself the file.
	SkillFiles []string `json:"skillFiles,omitempty"`

	// MCPServers are the config keys the MCP server entries were stored under.
	MCPServers []string `json:"mcpServers,omitempty"`
}

// Key returns the row's primary key.
func (e Entry) Key() Key {
	return Key{Name: e.Name, Agent: e.Agent, Scope: e.Scope}
}

// WithCarriedArtifacts returns e with the artifacts prior still records but
// this install did not write, carried forward.
//
// A row names every artifact dirctl wrote and has not since removed, not only
// the artifacts the newest install produced. Two cases make the difference
// matter, and both leave files or config keys on disk that nothing else would
// ever clean up:
//
//   - A partly successful reinstall. The skill is written but the agent's MCP
//     config cannot be read, so the new row names only the skill while the
//     earlier install's server key is still in that config.
//   - A new version that renames its MCP server. Installing it adds the new
//     key and leaves the old one in place, because plain install removes
//     nothing first.
//
// Skill paths are carried only when the new row has none, since one row holds
// one skill path and the slug is derived from the record name, so it does not
// move between installs of the same package.
func (e Entry) WithCarriedArtifacts(prior Entry) Entry {
	if e.SkillPath == "" && prior.SkillPath != "" {
		e.SkillPath = prior.SkillPath
		e.SkillFiles = prior.SkillFiles
	}

	seen := make(map[string]bool, len(e.MCPServers))
	for _, name := range e.MCPServers {
		seen[name] = true
	}

	for _, name := range prior.MCPServers {
		if !seen[name] {
			e.MCPServers = append(e.MCPServers, name)
			seen[name] = true
		}
	}

	return e
}

// manifestFile is the on-disk shape. It is separate from Manifest so the
// schema version is written from the constant on every save and never carried
// around as a mutable field.
type manifestFile struct {
	SchemaVersion int     `json:"schemaVersion"`
	Entries       []Entry `json:"entries"`
}

// Manifest is the in-memory manifest: Load builds one, Save writes it back.
type Manifest struct {
	// Entries are in stored order, which Upsert preserves so a file the user
	// reads does not reshuffle itself between installs.
	Entries []Entry

	degraded bool
	frozen   bool
}

// Load reads the manifest at path.
//
// A missing file is an empty manifest and no error — the ordinary
// first-install case. A file that cannot be understood is also an empty
// manifest, plus a non-nil error the caller should warn about rather than abort
// on, because losing the manifest must never fail an install.
//
// The two ways a file can be unusable differ in what the next Save does:
//
//	corrupt JSON            Degraded. Its rows are already unreadable, so Save
//	                        overwrites and the user starts a fresh manifest.
//	schemaVersion too new    Degraded and Frozen. A newer dirctl wrote it, so
//	                        Save refuses rather than destroy state this binary
//	                        cannot represent. An unreadable file is frozen too,
//	                        since its contents are unknown.
func Load(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return &Manifest{}, nil
		}

		return &Manifest{degraded: true, frozen: true},
			fmt.Errorf("read install manifest %s: %w", path, err)
	}

	var file manifestFile
	if err := json.Unmarshal(data, &file); err != nil {
		return &Manifest{degraded: true},
			fmt.Errorf("parse install manifest %s: %w", path, err)
	}

	if file.SchemaVersion > SchemaVersion {
		return &Manifest{degraded: true, frozen: true}, fmt.Errorf(
			"%w: %s has schema version %d, this dirctl understands %d — upgrade dirctl to manage installed packages",
			ErrFrozen, path, file.SchemaVersion, SchemaVersion)
	}

	return &Manifest{Entries: file.Entries}, nil
}

// Save writes the manifest to path through a temp file in the same directory,
// renamed over the target, so a reader never sees a half-written manifest and
// no partial file is left behind.
func (m *Manifest) Save(path string) error {
	if m.frozen {
		return fmt.Errorf("%w: refusing to overwrite %s", ErrFrozen, path)
	}

	// Marshal a non-nil slice so an emptied manifest is `"entries": []` rather
	// than `"entries": null`.
	entries := m.Entries
	if entries == nil {
		entries = []Entry{}
	}

	data, err := json.MarshalIndent(manifestFile{
		SchemaVersion: SchemaVersion,
		Entries:       entries,
	}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode install manifest: %w", err)
	}

	data = append(data, '\n')

	if err := fsutil.WriteAtomic(path, data, fsutil.WriteOptions{FileMode: manifestFilePerm}); err != nil {
		return fmt.Errorf("write install manifest %s: %w", path, err)
	}

	return nil
}

// Degraded reports that the file on disk could not be used, so Entries is
// empty even though the file may hold rows.
func (m *Manifest) Degraded() bool { return m.degraded }

// Frozen reports that Save will refuse, because the file belongs to a newer
// dirctl.
func (m *Manifest) Frozen() bool { return m.frozen }

// Upsert inserts e, or replaces the row with the same key in place so stored
// order stays stable.
func (m *Manifest) Upsert(e Entry) {
	key := e.Key()

	for i := range m.Entries {
		if m.Entries[i].Key() == key {
			m.Entries[i] = e

			return
		}
	}

	m.Entries = append(m.Entries, e)
}

// Remove drops the row with the given key and reports whether one was there.
func (m *Manifest) Remove(key Key) bool {
	for i := range m.Entries {
		if m.Entries[i].Key() == key {
			m.Entries = append(m.Entries[:i], m.Entries[i+1:]...)

			return true
		}
	}

	return false
}

// Find returns the row with the given key.
func (m *Manifest) Find(key Key) (Entry, bool) {
	for _, e := range m.Entries {
		if e.Key() == key {
			return e, true
		}
	}

	return Entry{}, false
}

// ByName returns every row for a package name — one per agent and scope it was
// installed into — in stored order.
func (m *Manifest) ByName(name string) []Entry {
	var found []Entry

	for _, e := range m.Entries {
		if e.Name == name {
			found = append(found, e)
		}
	}

	return found
}

// DefaultPath returns the global manifest path,
// $XDG_CONFIG_HOME/dirctl/installed.json, falling back to ~/.config when
// XDG_CONFIG_HOME is unset — the same convention the reusable client config
// already follows.
//
// This is the only function in the package that reads the environment.
// Everything else takes the path, which is what keeps the tests hermetic.
func DefaultPath() (string, error) {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("determine user home directory: %w", err)
		}

		configHome = filepath.Join(home, ".config")
	}

	return filepath.Join(configHome, configDirName, ManifestFileName), nil
}
