// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sigstoreTestConfig = `
current_context: corp
contexts:
  public:
    server_address: public.example.com:443
  corp:
    server_address: corp.example.com:443
    sigstore:
      fulcio_url: https://fulcio.corp.example
      rekor_url: https://rekor.corp.example
      timestamp_url: https://tsa.corp.example/api/v1/timestamp
      skip_tlog: true
      oidc_provider_url: https://idp.corp.example
      oidc_client_id: corp-sigstore
      tuf_mirror_url: https://tuf.corp.example
      trusted_root_path: /etc/dirctl/trusted_root.json
      ignore_tlog: true
      ignore_tsa: true
      ignore_sct: true
`

func TestResolveSigstore(t *testing.T) {
	t.Run("resolves the selected context's sigstore section", func(t *testing.T) {
		resetClientEnv(t)
		path := writeConfig(t, sigstoreTestConfig)

		cfg, resolved, err := ResolveSigstore(ResolveOptions{Path: path})

		require.NoError(t, err)
		assert.Equal(t, "corp", resolved.Name)
		assert.Equal(t, Sigstore{
			FulcioURL:       "https://fulcio.corp.example",
			RekorURL:        "https://rekor.corp.example",
			TimestampURL:    "https://tsa.corp.example/api/v1/timestamp",
			SkipTlog:        true,
			OIDCProviderURL: "https://idp.corp.example",
			OIDCClientID:    "corp-sigstore",
			TufMirrorURL:    "https://tuf.corp.example",
			TrustedRootPath: "/etc/dirctl/trusted_root.json",
			IgnoreTlog:      true,
			IgnoreTsa:       true,
			IgnoreSct:       true,
		}, cfg.Sigstore)
		assert.Equal(t, `context "corp"`, cfg.Sources["fulcio_url"])
		assert.Equal(t, `context "corp"`, cfg.Sources["ignore_tlog"])
	})

	t.Run("returns empty settings for a context without a sigstore section", func(t *testing.T) {
		resetClientEnv(t)
		path := writeConfig(t, sigstoreTestConfig)

		cfg, _, err := ResolveSigstore(ResolveOptions{Path: path, Context: "public"})

		require.NoError(t, err)
		assert.Equal(t, Sigstore{}, cfg.Sigstore)
		assert.Empty(t, cfg.Sources)
	})

	t.Run("returns empty settings without a selected context", func(t *testing.T) {
		resetClientEnv(t)
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())

		cfg, resolved, err := ResolveSigstore(ResolveOptions{})

		require.NoError(t, err)
		assert.Equal(t, "none", resolved.Source)
		assert.Equal(t, Sigstore{}, cfg.Sigstore)
		assert.Empty(t, cfg.Sources)
	})

	t.Run("environment overrides the context section", func(t *testing.T) {
		resetClientEnv(t)
		path := writeConfig(t, sigstoreTestConfig)
		t.Setenv("DIRECTORY_CLIENT_SIGSTORE_FULCIO_URL", "https://fulcio.env.example")
		t.Setenv("DIRECTORY_CLIENT_SIGSTORE_TRUSTED_ROOT_PATH", "/tmp/env_root.json")
		t.Setenv("DIRECTORY_CLIENT_SIGSTORE_SKIP_TLOG", "false")
		t.Setenv("DIRECTORY_CLIENT_SIGSTORE_IGNORE_SCT", "0")

		cfg, _, err := ResolveSigstore(ResolveOptions{Path: path})

		require.NoError(t, err)
		assert.Equal(t, "https://fulcio.env.example", cfg.FulcioURL)
		assert.Equal(t, "/tmp/env_root.json", cfg.TrustedRootPath)
		assert.False(t, cfg.SkipTlog)
		assert.False(t, cfg.IgnoreSct)
		assert.Equal(t, "https://rekor.corp.example", cfg.RekorURL)
		assert.True(t, cfg.IgnoreTlog)

		assert.Equal(t, "DIRECTORY_CLIENT_SIGSTORE_FULCIO_URL", cfg.Sources["fulcio_url"])
		assert.Equal(t, `context "corp"`, cfg.Sources["rekor_url"])
		assert.NotContains(t, cfg.Sources, "skip_tlog", "an env false unsets the setting")
	})

	t.Run("empty environment variables count as unset", func(t *testing.T) {
		resetClientEnv(t)
		path := writeConfig(t, sigstoreTestConfig)
		t.Setenv("DIRECTORY_CLIENT_SIGSTORE_FULCIO_URL", "")
		t.Setenv("DIRECTORY_CLIENT_SIGSTORE_IGNORE_TLOG", "")

		cfg, _, err := ResolveSigstore(ResolveOptions{Path: path})

		require.NoError(t, err)
		assert.Equal(t, "https://fulcio.corp.example", cfg.FulcioURL)
		assert.True(t, cfg.IgnoreTlog)
		assert.Equal(t, `context "corp"`, cfg.Sources["fulcio_url"])
	})

	t.Run("reports the first invalid boolean deterministically", func(t *testing.T) {
		resetClientEnv(t)
		path := writeConfig(t, sigstoreTestConfig)
		t.Setenv("DIRECTORY_CLIENT_SIGSTORE_IGNORE_SCT", "maybe")
		t.Setenv("DIRECTORY_CLIENT_SIGSTORE_SKIP_TLOG", "maybe")

		for range 20 {
			_, _, err := ResolveSigstore(ResolveOptions{Path: path})
			require.ErrorContains(t, err, "DIRECTORY_CLIENT_SIGSTORE_SKIP_TLOG")
		}
	})

	t.Run("rejects an invalid boolean in the environment", func(t *testing.T) {
		resetClientEnv(t)
		path := writeConfig(t, sigstoreTestConfig)
		t.Setenv("DIRECTORY_CLIENT_SIGSTORE_IGNORE_TLOG", "maybe")

		_, _, err := ResolveSigstore(ResolveOptions{Path: path})

		require.ErrorContains(t, err, "DIRECTORY_CLIENT_SIGSTORE_IGNORE_TLOG")
	})

	t.Run("rejects an unknown context", func(t *testing.T) {
		resetClientEnv(t)
		path := writeConfig(t, sigstoreTestConfig)

		_, _, err := ResolveSigstore(ResolveOptions{Path: path, Context: "missing"})

		require.ErrorContains(t, err, `unknown client context "missing"`)
	})
}

func TestSaveContextPreservesSigstore(t *testing.T) {
	resetClientEnv(t)
	path := writeConfig(t, sigstoreTestConfig)

	require.NoError(t, SaveContext(path, "dev", Context{ServerAddress: "dev.example.com:443"}, false))

	cfg, _, err := ResolveSigstore(ResolveOptions{Path: path})
	require.NoError(t, err)
	assert.Equal(t, "https://fulcio.corp.example", cfg.FulcioURL)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	contexts, ok := readRawConfig(t, path)["contexts"].(map[string]any)
	require.True(t, ok, "contexts section missing in:\n%s", data)

	dev, ok := contexts["dev"].(map[string]any)
	require.True(t, ok)
	assert.NotContains(t, dev, "sigstore", "an unset sigstore section must not be written")
}
