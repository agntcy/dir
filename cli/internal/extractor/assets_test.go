// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package extractor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsProvisioned(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{OASFURL: "https://x", AssetDir: dir}

	assert.False(t, IsProvisioned(cfg))

	require.NoError(t, os.WriteFile(filepath.Join(dir, "manifest.json"), []byte("{}"), 0o600))
	assert.True(t, IsProvisioned(cfg))
}

func TestTeardownRemovesAssetDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "extractor")
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "models"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "manifest.json"), []byte("{}"), 0o600))

	require.NoError(t, Teardown(Config{OASFURL: "https://x", AssetDir: dir}))

	_, err := os.Stat(dir)
	assert.True(t, os.IsNotExist(err))
}

func TestTeardownMissingDirIsNoError(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "does-not-exist")
	assert.NoError(t, Teardown(Config{OASFURL: "https://x", AssetDir: dir}))
}

func TestTeardownRefusesDangerousPaths(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	for _, bad := range []string{"", "/", home, "  ", "..", "../assets", "relative/dir", "."} {
		assert.Error(t, Teardown(Config{OASFURL: "https://x", AssetDir: bad}),
			"expected refusal for %q", bad)
	}
}
