// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/cli/internal/agentcfg"
	"github.com/agntcy/dir/cli/internal/agentinstall"
	"github.com/agntcy/dir/cli/internal/pkgstate"
	"github.com/agntcy/dir/cli/presenter"
	"github.com/spf13/cobra"
)

// applied is one record that an install or uninstall acted on, paired with the
// outcomes it produced. The record identity travels alongside the artifacts
// because a manifest row needs the name, version, and CID that Artifacts does
// not itself carry.
type applied struct {
	record   *corev1.Record
	arts     agentinstall.Artifacts
	pinned   bool
	outcomes []agentcfg.Outcome
}

// manifestRecordFn updates the install manifest after an apply. Install and
// uninstall pass their own; a dry run passes neither, because it writes nothing
// to record.
type manifestRecordFn func(cmd *cobra.Command, items []applied, agents []agentcfg.Agent, scope agentcfg.Scope)

// recordInstalls adds or replaces one manifest row per agent that actually
// received an artifact. Agents whose outcomes were all skipped or failed are
// left out, so a later listing never claims an install that did not happen.
func recordInstalls(cmd *cobra.Command, items []applied, agents []agentcfg.Agent, scope agentcfg.Scope) {
	if !recordable(scope) {
		return
	}

	now := time.Now().UTC()

	withManifest(cmd, func(m *pkgstate.Manifest) bool {
		changed := false

		for _, item := range items {
			name := item.record.GetName()
			if name == "" {
				// Nothing to key a row on. A nameless record can still be
				// installed by CID; it just cannot be tracked.
				continue
			}

			for _, agent := range agents {
				entry, ok := agentinstall.BuildEntry(item.arts, agent, item.outcomes)
				if !ok {
					continue
				}

				entry.Name = name
				entry.Version = item.record.GetVersion()
				entry.CID = item.record.GetCid()
				entry.Scope = pkgstate.ScopeGlobal
				entry.Origin = pkgstate.OriginDirectory
				entry.Pinned = item.pinned
				entry.InstalledAt = now

				m.Upsert(entry)

				changed = true
			}
		}

		return changed
	})
}

// recordUninstalls drops the manifest row for every agent the record's
// artifacts were removed from. An agent whose removal failed keeps its row,
// because its artifacts are still on disk.
func recordUninstalls(cmd *cobra.Command, items []applied, agents []agentcfg.Agent, scope agentcfg.Scope) {
	if !recordable(scope) {
		return
	}

	withManifest(cmd, func(m *pkgstate.Manifest) bool {
		changed := false

		for _, item := range items {
			name := item.record.GetName()
			if name == "" {
				continue
			}

			for _, agent := range agents {
				if !agentinstall.Cleared(agent, item.outcomes) {
					continue
				}

				if m.Remove(pkgstate.Key{Name: name, Agent: agent.ID, Scope: pkgstate.ScopeGlobal}) {
					changed = true
				}
			}
		}

		return changed
	})
}

// recordable reports whether artifacts written at this scope go into the
// manifest. Only global installs are tracked for now; the row shape already
// carries `scope`, so a committed per-project manifest is a later addition
// rather than a rewrite.
func recordable(scope agentcfg.Scope) bool {
	return scope == agentcfg.Global
}

// withManifest loads the manifest, applies mutate, and saves it when mutate
// reports something changed.
//
// Every failure here is a warning on stderr, never a returned error: the
// manifest is bookkeeping, not the product, so losing it must not fail an
// install the user just watched succeed.
func withManifest(cmd *cobra.Command, mutate func(*pkgstate.Manifest) bool) {
	path, err := pkgstate.DefaultPath()
	if err != nil {
		warnManifest(cmd, err)

		return
	}

	m, err := pkgstate.Load(path)
	if err != nil {
		warnManifest(cmd, err)
	}

	// A frozen manifest belongs to a newer dirctl, whose rows this binary
	// cannot represent. Save would refuse anyway; stopping here keeps the user
	// from reading two warnings about one file.
	if m.Frozen() {
		return
	}

	if !mutate(m) {
		return
	}

	if err := m.Save(path); err != nil {
		warnManifest(cmd, err)
	}
}

func warnManifest(cmd *cobra.Command, err error) {
	presenter.Errorf(cmd, "Warning: install manifest: %s\n", err)
}
