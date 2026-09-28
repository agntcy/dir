// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"testing"

	"github.com/agntcy/dir/cli/internal/pkgstate"
	"github.com/stretchr/testify/require"
)

// isolateHome points the agent-path resolver at a temp directory, so a
// developer's real skill folders and agent configs never decide a test.
func isolateHome(t *testing.T) string {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)

	return home
}

// seedManifest writes entries to the isolated manifest.
func seedManifest(t *testing.T, entries ...pkgstate.Entry) {
	t.Helper()

	path := isolateManifest(t)
	m := &pkgstate.Manifest{Entries: entries}
	require.NoError(t, m.Save(path))
}

// loadManifest reads back the isolated manifest, so a test can assert on what
// was written rather than on what was printed.
func loadManifest(t *testing.T) *pkgstate.Manifest {
	t.Helper()

	path, err := pkgstate.DefaultPath()
	require.NoError(t, err)

	m, err := pkgstate.Load(path)
	require.NoError(t, err)

	return m
}

// directoryEntry is a manifest row for a package pulled from a Directory,
// installed globally for one agent.
func directoryEntry(name, version, agent string) pkgstate.Entry {
	return pkgstate.Entry{
		Name:    name,
		Version: version,
		CID:     "bafy" + name,
		Agent:   agent,
		Scope:   pkgstate.ScopeGlobal,
		Origin:  pkgstate.OriginDirectory,
	}
}
