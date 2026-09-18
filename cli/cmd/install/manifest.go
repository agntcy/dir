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
// received an artifact.
func recordInstalls(cmd *cobra.Command, items []applied, agents []agentcfg.Agent, scope agentcfg.Scope) {
	now := time.Now().UTC()
	directory := ActiveDirectory()
	rowScope := manifestScope(scope)

	withManifest(cmd, func(m *pkgstate.Manifest) bool {
		changed := false

		for _, item := range items {
			if agentinstall.Record(m, item.arts, agents, item.outcomes, agentinstall.Identity{
				Name:      item.record.GetName(),
				Version:   item.record.GetVersion(),
				CID:       item.record.GetCid(),
				Scope:     rowScope,
				Origin:    pkgstate.OriginDirectory,
				Directory: directory,
				Pinned:    item.pinned,
			}, now) {
				changed = true
			}
		}

		return changed
	})
}

// recordUninstalls drops the manifest row for every agent the record's
// artifacts were removed from.
func recordUninstalls(cmd *cobra.Command, items []applied, agents []agentcfg.Agent, scope agentcfg.Scope) {
	rowScope := manifestScope(scope)

	withManifest(cmd, func(m *pkgstate.Manifest) bool {
		changed := false

		for _, item := range items {
			if agentinstall.Forget(m, item.record.GetName(), rowScope, agents, item.outcomes) {
				changed = true
			}
		}

		return changed
	})
}

// withManifest applies mutate to the manifest, warning on stderr if anything
// goes wrong.
//
// Every failure here is a warning, never a returned error: the manifest is
// bookkeeping, not the product, so losing it must not fail an install the user
// just watched succeed.
func withManifest(cmd *cobra.Command, mutate func(*pkgstate.Manifest) bool) {
	if err := pkgstate.Update(mutate); err != nil {
		warnManifest(cmd, err)
	}
}

// readManifest loads the manifest for a command whose whole job is to report
// on it, or to edit it.
//
// Unlike withManifest, a failure here is an error rather than a warning. An
// install that loses its bookkeeping still installed something; a command
// whose only product is the manifest must not report a write that did not
// happen.
//
//nolint:wrapcheck // pkgstate errors already name the manifest path and the operation.
func readManifest() (*pkgstate.Manifest, error) {
	return pkgstate.Read()
}

// editManifest applies mutate to the manifest and saves when mutate reports a
// change. See readManifest for why errors surface here rather than warn.
//
//nolint:wrapcheck // pkgstate errors already name the manifest path and the operation.
func editManifest(mutate func(*pkgstate.Manifest) bool) error {
	return pkgstate.Update(mutate)
}

func warnManifest(cmd *cobra.Command, err error) {
	presenter.Errorf(cmd, "Warning: install manifest: %s\n", err)
}
