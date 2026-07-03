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

func TestConfigResolveFillsDefaults(t *testing.T) {
	got := Config{}.Resolve()
	assert.Equal(t, DefaultOASFURL, got.OASFURL)
	assert.Equal(t, DefaultAssetDir(), got.AssetDir)
}

func TestConfigResolveKeepsExplicit(t *testing.T) {
	got := Config{OASFURL: "https://local", AssetDir: "/tmp/x"}.Resolve()
	assert.Equal(t, "https://local", got.OASFURL)
	assert.Equal(t, "/tmp/x", got.AssetDir)
}

func TestDefaultAssetDir(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(home, ".agntcy", "oasf-sdk", "extractor"), DefaultAssetDir())
}

func TestConfigValidate(t *testing.T) {
	require.NoError(t, Config{OASFURL: "https://schema.oasf.outshift.com"}.Validate())
	require.NoError(t, Config{OASFURL: "http://localhost:8080"}.Validate())
	require.Error(t, Config{OASFURL: ""}.Validate())
	require.Error(t, Config{OASFURL: "not-a-url"}.Validate())
	require.Error(t, Config{OASFURL: "ftp://x"}.Validate())
}
