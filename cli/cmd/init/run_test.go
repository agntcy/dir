// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package init

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	extractor "github.com/agntcy/dir/cli/internal/extractor"
	clientconfig "github.com/agntcy/dir/client/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPromptOASFURLKeepsDefaultOnEmpty(t *testing.T) {
	cmd, _ := newTestCmd("\n")
	got, err := promptOASFURL(cmd, "https://schema.oasf.outshift.com")
	require.NoError(t, err)
	assert.Equal(t, "https://schema.oasf.outshift.com", got)
}

func TestPromptOASFURLUsesInput(t *testing.T) {
	cmd, _ := newTestCmd("https://local.example\n")
	got, err := promptOASFURL(cmd, "https://schema.oasf.outshift.com")
	require.NoError(t, err)
	assert.Equal(t, "https://local.example", got)
}

func TestRunRemoveTearsDownAndClearsConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "xdg"))

	// Seed provisioned assets + a saved config section.
	assetDir := filepath.Join(t.TempDir(), "assets")
	require.NoError(t, os.MkdirAll(assetDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(assetDir, "manifest.json"), []byte("{}"), 0o600))
	require.NoError(t, clientconfig.SaveExtractor("", &clientconfig.Extractor{
		OASFURL: "https://schema.oasf.outshift.com", AssetDir: assetDir,
	}))

	cmd, out := newTestCmd("")
	err := runRemove(cmd, &options{assetDir: assetDir, yes: true})
	require.NoError(t, err)

	_, statErr := os.Stat(assetDir)
	assert.True(t, os.IsNotExist(statErr))

	got, err := clientconfig.LoadExtractor("")
	require.NoError(t, err)
	assert.Nil(t, got)
	assert.Contains(t, out.String(), "Removed")
}

func TestRunProvisionUsesSavedAssetDir(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "xdg"))

	// Provisioned once to a custom dir; the choice is saved in config.
	assetDir := filepath.Join(t.TempDir(), "custom")
	require.NoError(t, os.MkdirAll(assetDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(assetDir, "manifest.json"), []byte("{}"), 0o600))
	require.NoError(t, clientconfig.SaveExtractor("", &clientconfig.Extractor{
		OASFURL: extractor.DefaultOASFURL, AssetDir: assetDir,
	}))

	// A bare re-run (no --asset-dir) with EOF stdin: it must detect the saved
	// install rather than checking the default dir, then no-op on the prompt.
	cmd, out := newTestCmd("")
	err := runProvision(cmd, &options{oasfURL: extractor.DefaultOASFURL})
	require.NoError(t, err)

	assert.Contains(t, out.String(), "already provisioned at "+assetDir)
}

func TestRunProvisionNonTTYWithoutYesIsNoOp(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "xdg"))

	cmd, out := newTestCmd("") // EOF stdin => confirm returns false
	err := runProvision(cmd, &options{})
	require.NoError(t, err)

	// Nothing persisted.
	got, err := clientconfig.LoadExtractor("")
	require.NoError(t, err)
	assert.Nil(t, got)
	assert.True(t,
		strings.Contains(out.String(), "Skipped") || strings.Contains(out.String(), "--yes"),
		"expected a skip/--yes hint, got: %s", out.String())
}
