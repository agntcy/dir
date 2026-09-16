// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package dirpkg_test

import (
	"testing"

	"github.com/agntcy/dir/cli/internal/dirpkg"
	"github.com/agntcy/dir/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMCPServerEnvProjectsTheResolvedContext is the point of taking a config
// rather than finding one: `--context prod` has to reach the installed entry,
// or `dirctl mcp serve` dials somewhere the command never touched.
func TestMCPServerEnvProjectsTheResolvedContext(t *testing.T) {
	t.Parallel()

	env := dirpkg.MCPServerEnv(&client.Config{
		ServerAddress: "prod.example:8888",
		AuthMode:      "oidc",
	})

	assert.Equal(t, "prod.example:8888", env[dirpkg.ServerAddressEnv])
	assert.Equal(t, "oidc", env[dirpkg.AuthModeEnv])
}

// TestMCPServerEnvFallsBackToTheLocalDefault: no resolvable context is the
// ordinary case when the user declines init's Step 1. Mirroring the default
// makes the server dial the daemon insecurely rather than fall into OIDC
// auto-detection with an empty auth mode.
func TestMCPServerEnvFallsBackToTheLocalDefault(t *testing.T) {
	t.Parallel()

	for name, cfg := range map[string]*client.Config{
		"nil config":     nil,
		"empty settings": {},
		"blank settings": {ServerAddress: "  ", AuthMode: "\t"},
	} {
		env := dirpkg.MCPServerEnv(cfg)

		assert.Equal(t, dirpkg.LocalServerAddress, env[dirpkg.ServerAddressEnv], name)
		assert.Equal(t, dirpkg.LocalAuthMode, env[dirpkg.AuthModeEnv], name)
	}
}

// TestMCPServerEnvNeverProjectsASecret: the entry lands in an agent's config
// file, which is not a place for a bearer token.
func TestMCPServerEnvNeverProjectsASecret(t *testing.T) {
	t.Parallel()

	//nolint:gosec // G101: a fake token, which is the whole point of the test.
	env := dirpkg.MCPServerEnv(&client.Config{
		ServerAddress: "prod.example:8888",
		AuthMode:      "token",
		AuthToken:     "super-secret-jwt",
		SpiffeToken:   "/tmp/svid.json",
	})

	for key, value := range env {
		assert.NotContains(t, value, "super-secret-jwt", "%s leaked the auth token", key)
		assert.NotContains(t, value, "svid.json", "%s leaked the SPIFFE token", key)
	}

	assert.NotContains(t, env, "DIRECTORY_CLIENT_AUTH_TOKEN")
	assert.NotContains(t, env, "DIRECTORY_CLIENT_SPIFFE_TOKEN")
}

// TestMCPServerEnvProjectsNonSecretFieldsOnlyWhenSet: an empty value would
// override whatever the spawned server would otherwise work out for itself.
func TestMCPServerEnvProjectsNonSecretFieldsOnlyWhenSet(t *testing.T) {
	t.Parallel()

	bare := dirpkg.MCPServerEnv(&client.Config{ServerAddress: "a:1", AuthMode: "insecure"})
	assert.Len(t, bare, 2, "only the address and auth mode are unconditional")

	full := dirpkg.MCPServerEnv(&client.Config{
		ServerAddress: "a:1",
		AuthMode:      "oidc",
		TlsCAFile:     "/ca.pem",
		OIDCIssuer:    "https://dex.example",
		OIDCScopes:    []string{"openid", "groups"},
		TlsSkipVerify: true,
	})

	assert.Equal(t, "/ca.pem", full["DIRECTORY_CLIENT_TLS_CA_FILE"])
	assert.Equal(t, "https://dex.example", full["DIRECTORY_CLIENT_OIDC_ISSUER"])
	assert.Equal(t, "openid,groups", full["DIRECTORY_CLIENT_OIDC_SCOPES"])
	assert.Equal(t, "true", full["DIRECTORY_CLIENT_TLS_SKIP_VERIFY"])
	assert.NotContains(t, full, "DIRECTORY_CLIENT_TLS_CERT_FILE")
}

// TestArtifactsCarriesTheEnvOntoTheMCPEntry: the built-in record's MCP module
// ships no env of its own, so the overlay is the only thing pointing the
// spawned server anywhere.
func TestArtifactsCarriesTheEnvOntoTheMCPEntry(t *testing.T) {
	t.Parallel()

	arts, err := dirpkg.Artifacts(&client.Config{ServerAddress: "prod.example:8888", AuthMode: "oidc"})
	require.NoError(t, err)

	assert.NotEmpty(t, dirpkg.Name())
	assert.NotEmpty(t, dirpkg.Version())
	assert.NotEmpty(t, arts.MCPServerNames(), "the built-in package installs an MCP server")
	assert.True(t, arts.HasSkill(), "and a skill")
}
