// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agentinstall

import (
	"time"

	"github.com/agntcy/dir/cli/internal/agentcfg"
	"github.com/agntcy/dir/cli/internal/pkgstate"
)

// Identity is the record identity a manifest row needs and Artifacts does not
// carry: which package this is, where it came from, and whether it is held.
type Identity struct {
	// Name is the record name, which is what a row is keyed on.
	Name string

	// Version is the record version, spelled as the record spells it.
	Version string

	// CID is the content address of the exact record installed. Empty for a
	// built-in package: it is rebuilt on demand with the current timestamp, so
	// its content address differs every time and recording one would be noise
	// that no version check can use.
	CID string

	// Scope is where the artifacts landed.
	Scope pkgstate.Scope

	// Origin is where to look for a newer version.
	Origin pkgstate.Origin

	// Directory is the server address the package came from. Empty for a
	// built-in package, whose upstream is this binary rather than any
	// Directory.
	Directory string

	// Pinned holds the package at Version, so a bare upgrade skips it.
	Pinned bool
}

// Record upserts one row per agent that actually received an artifact, and
// reports whether it changed anything.
//
// Agents whose outcomes were all skipped or failed are left out, so a later
// listing never claims an install that did not happen. It is shared by every
// command that writes artifacts — `install`, its piped form, `init`, and
// `upgrade` — because a row written by one of them is read by all the others,
// and four copies of this would drift.
func Record(
	m *pkgstate.Manifest,
	arts Artifacts,
	agents []agentcfg.Agent,
	outcomes []agentcfg.Outcome,
	id Identity,
	now time.Time,
) bool {
	if id.Name == "" {
		// Nothing to key a row on. A nameless record can still be installed by
		// CID; it just cannot be tracked.
		return false
	}

	changed := false

	for _, agent := range agents {
		entry, ok := BuildEntry(arts, agent, outcomes)
		if !ok {
			continue
		}

		entry.Name = id.Name
		entry.Version = id.Version
		entry.CID = id.CID
		entry.Scope = id.Scope
		entry.Origin = id.Origin
		entry.Directory = id.Directory
		entry.Pinned = id.Pinned
		entry.InstalledAt = now

		// Carry forward artifacts the previous row named and this install did
		// not write, so a partly successful reinstall never drops the only
		// record of something still on disk.
		if prior, found := m.Find(entry.Key()); found {
			entry = entry.WithCarriedArtifacts(prior)
		}

		m.Upsert(entry)

		changed = true
	}

	return changed
}

// Reconcile rewrites the rows for a package whose version moved: it drops what
// the orphan prune took away, then records what the install wrote.
//
// It is Record plus one subtraction, and the subtraction is the whole reason it
// exists. Record carries forward artifacts the previous row named and the new
// install did not write, because on a plain reinstall those are still on disk.
// After an upgrade that only holds for the ones the prune left alone: v1's
// renamed MCP key has just been taken out of the agent's config, so carrying it
// onto the new row would leave the row claiming something gone, and send a
// later uninstall hunting for it.
func Reconcile(
	m *pkgstate.Manifest,
	arts Artifacts,
	agents []agentcfg.Agent,
	removed, installed []agentcfg.Outcome,
	id Identity,
	now time.Time,
) bool {
	changed := false

	for _, agent := range agents {
		skill, servers := Pruned(agent, removed)
		if !skill && len(servers) == 0 {
			continue
		}

		prior, ok := m.Find(pkgstate.Key{Name: id.Name, Agent: agent.ID, Scope: id.Scope})
		if !ok {
			continue
		}

		stripped := prior.WithoutArtifacts(skill, servers)
		if stripped.SkillPath == "" && len(stripped.MCPServers) == 0 {
			// Nothing of ours is left, so the row would stand for nothing. An
			// install that then succeeds writes a fresh one below.
			m.Remove(stripped.Key())
		} else {
			m.Upsert(stripped)
		}

		changed = true
	}

	// Ordered so the subtraction lands first: Record reads the row back to
	// decide what to carry forward.
	return Record(m, arts, agents, installed, id, now) || changed
}

// Forget drops the row for every agent left with nothing of ours, and reports
// whether it changed anything.
//
// An agent whose removal failed keeps its row, because its artifacts are still
// on disk and the row is the only note of what they are.
func Forget(
	m *pkgstate.Manifest,
	name string,
	scope pkgstate.Scope,
	agents []agentcfg.Agent,
	outcomes []agentcfg.Outcome,
) bool {
	if name == "" {
		return false
	}

	changed := false

	for _, agent := range agents {
		if !Cleared(agent, outcomes) {
			continue
		}

		if m.Remove(pkgstate.Key{Name: name, Agent: agent.ID, Scope: scope}) {
			changed = true
		}
	}

	return changed
}
