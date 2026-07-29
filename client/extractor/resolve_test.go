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

// provisionMarker writes the on-disk manifest that marks an asset dir as
// provisioned, so chooseBackend's local branch sees a real provisioned dir.
func provisionMarker(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, manifestName), []byte("{}"), 0o600))

	return dir
}

func TestChooseBackendPrefersRemoteWhenConfigured(t *testing.T) {
	// A configured remote address wins even when local assets are also present:
	// the resolver must not pay a local model load when a server is available.
	kind, err := chooseBackend(Config{RemoteAddr: "oasf-sdk:31234", AssetDir: provisionMarker(t)})

	require.NoError(t, err)
	assert.Equal(t, backendRemote, kind)
}

func TestChooseBackendFallsBackToLocalAssets(t *testing.T) {
	// No remote address, but provisioned assets on disk: use the local backend.
	kind, err := chooseBackend(Config{AssetDir: provisionMarker(t)})

	require.NoError(t, err)
	assert.Equal(t, backendLocal, kind)
}

func TestChooseBackendErrorsWhenNeitherAvailable(t *testing.T) {
	// No remote address and an unprovisioned dir: the caller gets one actionable
	// error naming both ways to fix it, not a silent empty extractor.
	_, err := chooseBackend(Config{AssetDir: filepath.Join(t.TempDir(), "absent")})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "dirctl init")
	assert.Contains(t, err.Error(), "OASF-SDK server")
}
