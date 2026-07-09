// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package init

import (
	"path/filepath"
	"testing"

	clientconfig "github.com/agntcy/dir/client/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunContextSetupSeedsLocalWithYes(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "xdg"))

	cmd, _ := newTestCmd("")
	require.NoError(t, runContextSetup(cmd, &options{yes: true}))

	summaries, err := clientconfig.ListContexts("")
	require.NoError(t, err)
	require.Len(t, summaries, 1)
	assert.Equal(t, localContextName, summaries[0].Name)
	assert.True(t, summaries[0].Current, "seeded context must be current")

	got, err := clientconfig.LoadFile(mustDefaultPath(t))
	require.NoError(t, err)
	assert.Equal(t, localServerAddress, got.Contexts[localContextName].ServerAddress)
	assert.Equal(t, localAuthMode, got.Contexts[localContextName].AuthMode)
}

func TestRunContextSetupSkipsWhenConfigured(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "xdg"))
	require.NoError(t, clientconfig.SaveContext("", "prod",
		clientconfig.Context{ServerAddress: "prod:443"}, true))

	cmd, out := newTestCmd("")
	require.NoError(t, runContextSetup(cmd, &options{yes: true}))

	// Existing config untouched: only prod, still current; no local added.
	summaries, err := clientconfig.ListContexts("")
	require.NoError(t, err)
	require.Len(t, summaries, 1)
	assert.Equal(t, "prod", summaries[0].Name)
	assert.Contains(t, out.String(), "already configured")
}

func TestRunContextSetupNonTTYWithoutYesSkips(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "xdg"))

	cmd, out := newTestCmd("") // non-TTY (strings.Reader), no --yes
	require.NoError(t, runContextSetup(cmd, &options{}))

	summaries, err := clientconfig.ListContexts("")
	require.NoError(t, err)
	assert.Empty(t, summaries, "must not seed a context non-interactively without --yes")
	assert.Contains(t, out.String(), "non-interactive")
}

func mustDefaultPath(t *testing.T) string {
	t.Helper()

	path, err := clientconfig.DefaultPath()
	require.NoError(t, err)

	return path
}
