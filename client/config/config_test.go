// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"path/filepath"
	"testing"

	dirclient "github.com/agntcy/dir/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultPath(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "xdg"))

	path, err := DefaultPath()

	require.NoError(t, err)
	assert.Equal(t, filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "dirctl", "config.yaml"), path)
}

func TestLoadFile(t *testing.T) {
	t.Run("loads schema", func(t *testing.T) {
		path := writeConfig(t, `
current_context: dev
contexts:
  dev:
    server_address: dev.gateway.example.com:443
    auth_mode: oidc
    oidc_issuer: https://dev.idp.example.com
    oidc_client_id: dirctl
    doctor:
      bootstrap_peers:
        - /dns4/dev.routing.example.com/tcp/5555/p2p/12D3KooWExample
`)

		file, err := LoadFile(path)

		require.NoError(t, err)
		assert.Equal(t, "dev", file.CurrentContext)
		require.Contains(t, file.Contexts, "dev")
		assert.Equal(t, "dev.gateway.example.com:443", file.Contexts["dev"].ServerAddress)
		assert.Equal(t, []string{"/dns4/dev.routing.example.com/tcp/5555/p2p/12D3KooWExample"}, file.Contexts["dev"].Doctor.BootstrapPeers)
	})

	t.Run("rejects unknown fields", func(t *testing.T) {
		path := writeConfig(t, `
current_context: dev
contexts:
  dev:
    server_address: dev.gateway.example.com:443
    unexpected_field: value
`)

		file, err := LoadFile(path)

		require.Error(t, err)
		assert.Nil(t, file)
		assert.Contains(t, err.Error(), "unexpected_field")
	})

	t.Run("rejects current context that is not configured", func(t *testing.T) {
		path := writeConfig(t, `
current_context: prod
contexts:
  dev:
    server_address: dev.gateway.example.com:443
`)

		file, err := LoadFile(path)

		require.Error(t, err)
		assert.Nil(t, file)
		assert.Contains(t, err.Error(), `current_context "prod"`)
	})

	t.Run("rejects empty context name", func(t *testing.T) {
		path := writeConfig(t, `
contexts:
  "":
    server_address: dev.gateway.example.com:443
`)

		file, err := LoadFile(path)

		require.Error(t, err)
		assert.Nil(t, file)
		assert.Contains(t, err.Error(), "context name must not be empty")
	})

	t.Run("rejects path separators in context name", func(t *testing.T) {
		path := writeConfig(t, `
contexts:
  dev/prod:
    server_address: dev.gateway.example.com:443
`)

		file, err := LoadFile(path)

		require.Error(t, err)
		assert.Nil(t, file)
		assert.Contains(t, err.Error(), "must not contain path separators")
	})
}

func TestResolveDoctor(t *testing.T) {
	t.Run("resolves doctor settings from selected context", func(t *testing.T) {
		resetClientEnv(t)
		path := writeConfig(t, `
current_context: prod
contexts:
  dev:
    server_address: dev.gateway.example.com:443
    auth_mode: insecure
    doctor:
      bootstrap_peers:
        - /dns4/dev.routing.example.com/tcp/5555/p2p/12D3KooWDev
  prod:
    server_address: prod.gateway.example.com:443
    auth_mode: insecure
    doctor:
      bootstrap_peers:
        - /dns4/prod.routing.example.com/tcp/5555/p2p/12D3KooWProd
`)

		cfg, resolved, err := ResolveDoctor(ResolveOptions{
			Path:    path,
			Context: "dev",
		})

		require.NoError(t, err)
		assert.Equal(t, "dev", resolved.Name)
		assert.Equal(t, "option", resolved.Source)
		assert.Equal(t, []string{"/dns4/dev.routing.example.com/tcp/5555/p2p/12D3KooWDev"}, cfg.BootstrapPeers)
	})

	t.Run("returns empty config without a selected context", func(t *testing.T) {
		resetClientEnv(t)
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())

		cfg, resolved, err := ResolveDoctor(ResolveOptions{})

		require.NoError(t, err)
		assert.Empty(t, resolved.Name)
		assert.Equal(t, "none", resolved.Source)
		assert.Empty(t, cfg.BootstrapPeers)
	})
}

func TestListContexts(t *testing.T) {
	t.Run("lists configured contexts", func(t *testing.T) {
		path := writeConfig(t, `
current_context: prod
contexts:
  prod:
    server_address: prod.gateway.example.com:443
  dev:
    server_address: dev.gateway.example.com:443
`)

		contexts, err := ListContexts(path)

		require.NoError(t, err)
		assert.Equal(t, []ContextSummary{
			{Name: "dev"},
			{Name: "prod", Current: true},
		}, contexts)
	})

	t.Run("returns empty list when default config is missing", func(t *testing.T) {
		resetClientEnv(t)
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())

		contexts, err := ListContexts("")

		require.NoError(t, err)
		assert.Empty(t, contexts)
	})

	t.Run("allows unknown context fields", func(t *testing.T) {
		path := writeConfig(t, `
current_context: prod
contexts:
  prod:
    server_address: prod.gateway.example.com:443
    doctor:
      timeout: 10s
  dev:
    server_address: dev.gateway.example.com:443
`)

		contexts, err := ListContexts(path)

		require.NoError(t, err)
		assert.Equal(t, []ContextSummary{
			{Name: "dev"},
			{Name: "prod", Current: true},
		}, contexts)
	})
}

func TestSetCurrentContext(t *testing.T) {
	path := writeConfig(t, `
current_context: dev
contexts:
  dev:
    server_address: dev.gateway.example.com:443
  prod:
    server_address: prod.gateway.example.com:443
`)

	resolved, err := SetCurrentContext(path, "prod")

	require.NoError(t, err)
	assert.Equal(t, "prod", resolved.Name)
	assert.Equal(t, "current_context", resolved.Source)

	file, err := LoadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "prod", file.CurrentContext)
}

func TestCurrentContext(t *testing.T) {
	t.Run("returns persisted current context", func(t *testing.T) {
		resetClientEnv(t)
		path := writeConfig(t, `
current_context: dev
contexts:
  dev:
    server_address: dev.gateway.example.com:443
  prod:
    server_address: prod.gateway.example.com:443
`)

		t.Setenv(ClientContextEnv, "prod")

		current, err := CurrentContext(path)

		require.NoError(t, err)
		assert.Equal(t, "dev", current.Name)
		assert.Equal(t, "current_context", current.Source)
	})

	t.Run("allows unknown context fields", func(t *testing.T) {
		resetClientEnv(t)
		path := writeConfig(t, `
current_context: dev
contexts:
  dev:
    server_address: dev.gateway.example.com:443
    doctor:
      timeout: 10s
`)

		current, err := CurrentContext(path)

		require.NoError(t, err)
		assert.Equal(t, "dev", current.Name)
		assert.Equal(t, "current_context", current.Source)
	})
}

func TestValidateContexts(t *testing.T) {
	path := writeConfig(t, `
contexts:
  valid:
    server_address: dev.gateway.example.com:443
    auth_mode: insecure
  invalid:
    auth_mode: insecure
`)

	results, err := ValidateContexts(path, "")

	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, "invalid", results[0].Name)
	require.ErrorContains(t, results[0].Error, "server_address is required")
	assert.Equal(t, "valid", results[1].Name)
	assert.NoError(t, results[1].Error)
}

func TestValidateContexts_AllowsUnknownContextFields(t *testing.T) {
	path := writeConfig(t, `
contexts:
  valid:
    server_address: dev.gateway.example.com:443
    auth_mode: insecure
    doctor:
      timeout: 10s
`)

	results, err := ValidateContexts(path, "")

	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "valid", results[0].Name)
	assert.NoError(t, results[0].Error)
}

func TestResolve(t *testing.T) {
	t.Run("resolves explicit context", func(t *testing.T) {
		resetClientEnv(t)
		path := writeConfig(t, `
current_context: prod
contexts:
  dev:
    server_address: dev.gateway.example.com:443
    auth_mode: oidc
    oidc_issuer: https://dev.idp.example.com
    oidc_client_id: dirctl
  prod:
    server_address: prod.gateway.example.com:443
    auth_mode: oidc
    oidc_issuer: https://prod.idp.example.com
    oidc_client_id: dirctl
`)

		cfg, resolved, err := Resolve(ResolveOptions{
			Path:    path,
			Context: "dev",
		})

		require.NoError(t, err)
		assert.Equal(t, "dev.gateway.example.com:443", cfg.ServerAddress)
		assert.Equal(t, "oidc", cfg.AuthMode)
		assert.Equal(t, "https://dev.idp.example.com", cfg.OIDCIssuer)
		assert.Equal(t, "dev", resolved.Name)
		assert.Equal(t, "option", resolved.Source)
		assert.Equal(t, path, resolved.Path)
	})

	t.Run("resolves current context", func(t *testing.T) {
		resetClientEnv(t)
		path := writeConfig(t, `
current_context: prod
contexts:
  prod:
    server_address: prod.gateway.example.com:443
    auth_mode: insecure
`)

		cfg, resolved, err := Resolve(ResolveOptions{Path: path})

		require.NoError(t, err)
		assert.Equal(t, "prod.gateway.example.com:443", cfg.ServerAddress)
		assert.Equal(t, "prod", resolved.Name)
		assert.Equal(t, "current_context", resolved.Source)
	})

	t.Run("environment overrides context", func(t *testing.T) {
		resetClientEnv(t)
		t.Setenv("DIRECTORY_CLIENT_SERVER_ADDRESS", "env.gateway.example.com:443")
		t.Setenv("DIRECTORY_CLIENT_AUTH_MODE", "insecure")

		path := writeConfig(t, `
current_context: dev
contexts:
  dev:
    server_address: dev.gateway.example.com:443
    auth_mode: oidc
    oidc_issuer: https://dev.idp.example.com
    oidc_client_id: dirctl
`)

		cfg, _, err := Resolve(ResolveOptions{Path: path})

		require.NoError(t, err)
		assert.Equal(t, "env.gateway.example.com:443", cfg.ServerAddress)
		assert.Equal(t, "insecure", cfg.AuthMode)
	})

	t.Run("context environment selects context", func(t *testing.T) {
		resetClientEnv(t)
		t.Setenv(ClientContextEnv, "dev")

		path := writeConfig(t, `
current_context: prod
contexts:
  dev:
    server_address: dev.gateway.example.com:443
    auth_mode: insecure
  prod:
    server_address: prod.gateway.example.com:443
    auth_mode: insecure
`)

		cfg, resolved, err := Resolve(ResolveOptions{Path: path})

		require.NoError(t, err)
		assert.Equal(t, "dev.gateway.example.com:443", cfg.ServerAddress)
		assert.Equal(t, "dev", resolved.Name)
		assert.Equal(t, "env", resolved.Source)
	})

	t.Run("allows unknown fields when requested", func(t *testing.T) {
		resetClientEnv(t)
		path := writeConfig(t, `
current_context: dev
contexts:
  dev:
    server_address: dev.gateway.example.com:443
    auth_mode: oidc
    oidc_issuer: https://dev.idp.example.com
    doctor:
      timeout: 10s
`)

		cfg, resolved, err := Resolve(ResolveOptions{
			Path:               path,
			AllowUnknownFields: true,
		})

		require.NoError(t, err)
		assert.Equal(t, "dev.gateway.example.com:443", cfg.ServerAddress)
		assert.Equal(t, "https://dev.idp.example.com", cfg.OIDCIssuer)
		assert.Equal(t, "dev", resolved.Name)
	})

	t.Run("works without config file when env provides required values", func(t *testing.T) {
		resetClientEnv(t)
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("DIRECTORY_CLIENT_SERVER_ADDRESS", "env.gateway.example.com:443")
		t.Setenv("DIRECTORY_CLIENT_AUTH_MODE", "insecure")

		cfg, resolved, err := Resolve(ResolveOptions{})

		require.NoError(t, err)
		assert.Equal(t, "env.gateway.example.com:443", cfg.ServerAddress)
		assert.Empty(t, resolved.Name)
		assert.Equal(t, "none", resolved.Source)
	})

	t.Run("explicit overrides win", func(t *testing.T) {
		resetClientEnv(t)
		t.Setenv("DIRECTORY_CLIENT_SERVER_ADDRESS", "env.gateway.example.com:443")

		path := writeConfig(t, `
current_context: dev
contexts:
  dev:
    server_address: dev.gateway.example.com:443
    auth_mode: oidc
    oidc_issuer: https://dev.idp.example.com
    oidc_client_id: dirctl
`)

		cfg, _, err := Resolve(ResolveOptions{
			Path: path,
			Overrides: &dirclient.Config{
				ServerAddress: "flag.gateway.example.com:443",
				AuthMode:      "insecure",
			},
			OverrideFields: []string{"server_address", "auth_mode"},
		})

		require.NoError(t, err)
		assert.Equal(t, "flag.gateway.example.com:443", cfg.ServerAddress)
		assert.Equal(t, "insecure", cfg.AuthMode)
	})

	t.Run("resolves tls and spiffe fields", func(t *testing.T) {
		resetClientEnv(t)
		path := writeConfig(t, `
contexts:
  secure:
    server_address: secure.gateway.example.com:443
    auth_mode: jwt
    jwt_audience: directory
    spiffe_socket_path: /tmp/spire-agent.sock
    spiffe_token: spiffe-jwt
    tls_skip_verify: true
    tls_ca_file: /tmp/ca.pem
    tls_cert_file: /tmp/client.pem
    tls_key_file: /tmp/client-key.pem
`)

		cfg, resolved, err := Resolve(ResolveOptions{
			Path:    path,
			Context: "secure",
		})

		require.NoError(t, err)
		assert.Equal(t, "secure", resolved.Name)
		assert.Equal(t, "secure.gateway.example.com:443", cfg.ServerAddress)
		assert.Equal(t, "jwt", cfg.AuthMode)
		assert.Equal(t, "directory", cfg.JWTAudience)
		assert.Equal(t, "/tmp/spire-agent.sock", cfg.SpiffeSocketPath)
		assert.Equal(t, "spiffe-jwt", cfg.SpiffeToken)
		assert.True(t, cfg.TlsSkipVerify)
		assert.Equal(t, "/tmp/ca.pem", cfg.TlsCAFile)
		assert.Equal(t, "/tmp/client.pem", cfg.TlsCertFile)
		assert.Equal(t, "/tmp/client-key.pem", cfg.TlsKeyFile)
	})
}

func TestResolveErrors(t *testing.T) {
	t.Run("explicit missing file errors", func(t *testing.T) {
		resetClientEnv(t)

		cfg, resolved, err := Resolve(ResolveOptions{Path: filepath.Join(t.TempDir(), "missing.yaml")})

		require.Error(t, err)
		assert.Nil(t, cfg)
		assert.Nil(t, resolved)
		assert.Contains(t, err.Error(), "failed to open client config file")
	})

	t.Run("unknown context errors", func(t *testing.T) {
		resetClientEnv(t)
		path := writeConfig(t, `
contexts:
  dev:
    server_address: dev.gateway.example.com:443
`)

		cfg, resolved, err := Resolve(ResolveOptions{Path: path, Context: "prod"})

		require.Error(t, err)
		assert.Nil(t, cfg)
		assert.Nil(t, resolved)
		assert.Contains(t, err.Error(), `unknown client context "prod"`)
	})

	t.Run("missing server address errors", func(t *testing.T) {
		resetClientEnv(t)
		path := writeConfig(t, `
contexts:
  dev:
    auth_mode: insecure
`)

		cfg, resolved, err := Resolve(ResolveOptions{Path: path, Context: "dev"})

		require.Error(t, err)
		assert.Nil(t, cfg)
		assert.Nil(t, resolved)
		assert.Contains(t, err.Error(), "server_address is required")
	})

	t.Run("invalid auth mode errors", func(t *testing.T) {
		resetClientEnv(t)
		path := writeConfig(t, `
contexts:
  dev:
    server_address: dev.gateway.example.com:443
    auth_mode: invalid
`)

		cfg, resolved, err := Resolve(ResolveOptions{Path: path, Context: "dev"})

		require.Error(t, err)
		assert.Nil(t, cfg)
		assert.Nil(t, resolved)
		assert.Contains(t, err.Error(), `unsupported auth_mode "invalid"`)
	})

	t.Run("oidc requires issuer without token", func(t *testing.T) {
		resetClientEnv(t)
		path := writeConfig(t, `
contexts:
  dev:
    server_address: dev.gateway.example.com:443
    auth_mode: oidc
`)

		cfg, resolved, err := Resolve(ResolveOptions{Path: path, Context: "dev"})

		require.Error(t, err)
		assert.Nil(t, cfg)
		assert.Nil(t, resolved)
		assert.Contains(t, err.Error(), "oidc_issuer is required")
	})

	t.Run("oidc allows issuer without client id for cached token usage", func(t *testing.T) {
		resetClientEnv(t)
		path := writeConfig(t, `
contexts:
  dev:
    server_address: dev.gateway.example.com:443
    auth_mode: oidc
    oidc_issuer: https://dev.idp.example.com
`)

		cfg, resolved, err := Resolve(ResolveOptions{Path: path, Context: "dev"})

		require.NoError(t, err)
		assert.Equal(t, "oidc", cfg.AuthMode)
		assert.Equal(t, "https://dev.idp.example.com", cfg.OIDCIssuer)
		assert.Empty(t, cfg.OIDCClientID)
		assert.Equal(t, "dev", resolved.Name)
	})
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	return path
}

func TestExtractorRoundTrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "xdg"))

	path, err := DefaultPath()
	require.NoError(t, err)

	// Save on a machine with no config file yet.
	require.NoError(t, SaveExtractor(path, &Extractor{
		OASFURL:  "https://schema.oasf.outshift.com",
		AssetDir: "/home/u/.agntcy/oasf-sdk/extractor",
	}))

	got, err := LoadExtractor(path)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "https://schema.oasf.outshift.com", got.OASFURL)
	assert.Equal(t, "/home/u/.agntcy/oasf-sdk/extractor", got.AssetDir)

	// Clear removes the section but keeps the file loadable.
	require.NoError(t, ClearExtractor(path))
	got, err = LoadExtractor(path)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestLoadExtractorAbsentFile(t *testing.T) {
	got, err := LoadExtractor(filepath.Join(t.TempDir(), "nope.yaml"))
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestSaveExtractorPreservesContexts(t *testing.T) {
	path := writeConfig(t, `
current_context: dev
contexts:
  dev:
    server_address: dev.example.com:443
`)
	require.NoError(t, SaveExtractor(path, &Extractor{
		OASFURL:  "https://schema.oasf.outshift.com",
		AssetDir: "/assets",
	}))

	file, err := LoadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "dev", file.CurrentContext)
	require.Contains(t, file.Contexts, "dev")
	require.NotNil(t, file.Extractor)
	assert.Equal(t, "https://schema.oasf.outshift.com", file.Extractor.OASFURL)
}

func resetClientEnv(t *testing.T) {
	t.Helper()

	keys := []string{
		ClientContextEnv,
		"DIRECTORY_CLIENT_SERVER_ADDRESS",
		"DIRECTORY_CLIENT_TLS_SKIP_VERIFY",
		"DIRECTORY_CLIENT_TLS_CERT_FILE",
		"DIRECTORY_CLIENT_TLS_KEY_FILE",
		"DIRECTORY_CLIENT_TLS_CA_FILE",
		"DIRECTORY_CLIENT_SPIFFE_SOCKET_PATH",
		"DIRECTORY_CLIENT_SPIFFE_TOKEN",
		"DIRECTORY_CLIENT_AUTH_MODE",
		"DIRECTORY_CLIENT_JWT_AUDIENCE",
		"DIRECTORY_CLIENT_OIDC_ISSUER",
		"DIRECTORY_CLIENT_OIDC_CLIENT_ID",
		"DIRECTORY_CLIENT_AUTH_TOKEN",
	}

	original := make(map[string]string, len(keys))

	present := make(map[string]bool, len(keys))
	for _, key := range keys {
		value, ok := os.LookupEnv(key)
		original[key] = value
		present[key] = ok
		require.NoError(t, os.Unsetenv(key)) //nolint:usetesting // Need truly unset variables.
	}

	t.Cleanup(func() {
		for _, key := range keys {
			if !present[key] {
				_ = os.Unsetenv(key) //nolint:usetesting // Restoring process env after manual unset.

				continue
			}

			_ = os.Setenv(key, original[key]) //nolint:usetesting // Restoring process env after manual unset.
		}
	})
}
