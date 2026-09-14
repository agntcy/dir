// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"testing"
	"time"

	dbconfig "github.com/agntcy/dir/server/database/config"
	ansconfig "github.com/agntcy/dir/server/identity/ans/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig_NoFile_ReturnsDefaults(t *testing.T) {
	cfg, err := LoadConfig()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Database defaults
	assert.Equal(t, dbconfig.DefaultType, cfg.Database.Type)
	assert.Equal(t, dbconfig.DefaultPostgresHost, cfg.Database.Postgres.Host)
	assert.Equal(t, dbconfig.DefaultPostgresPort, cfg.Database.Postgres.Port)
	assert.Equal(t, dbconfig.DefaultPostgresDatabase, cfg.Database.Postgres.Database)
	assert.Equal(t, dbconfig.DefaultPostgresSSLMode, cfg.Database.Postgres.SSLMode)

	// Task defaults
	assert.True(t, cfg.Regsync.Enabled)
	assert.True(t, cfg.Indexer.Enabled)
}

func TestLoadConfig_EnvOverrides(t *testing.T) {
	t.Setenv("RECONCILER_REGSYNC_ENABLED", "false")
	t.Setenv("RECONCILER_INDEXER_ENABLED", "false")
	t.Setenv("RECONCILER_INDEXER_INTERVAL", "2h")

	cfg, err := LoadConfig()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.False(t, cfg.Regsync.Enabled)
	assert.False(t, cfg.Indexer.Enabled)
	assert.Equal(t, 2*time.Hour, cfg.Indexer.Interval)
}

func TestLoadConfig_IdentityDefaultsOff(t *testing.T) {
	cfg, err := LoadConfig()
	require.NoError(t, err)

	assert.False(t, cfg.Identity.Enabled)
	assert.False(t, cfg.Identity.Ans.Enabled)
}

// The identity task and its ans block are absent from most configuration
// files, so every key must be bound for environment overrides to resolve.
func TestLoadConfig_IdentityEnvOverrides(t *testing.T) {
	t.Setenv("RECONCILER_IDENTITY_ENABLED", "true")
	t.Setenv("RECONCILER_IDENTITY_INTERVAL", "2m")
	t.Setenv("RECONCILER_IDENTITY_ANS_ENABLED", "true")
	t.Setenv("RECONCILER_IDENTITY_ANS_TRUSTED_LOG_HOSTS", "log-a.example.com:8443,log-b.example.com")
	t.Setenv("RECONCILER_IDENTITY_ANS_ROOT_KEYS", "log-a.example.com+abcd1234+AAAA,log-b.example.com+ef567890+BBBB")
	t.Setenv("RECONCILER_IDENTITY_ANS_ALLOW_UNPINNED_ROOT_KEYS", "true")
	t.Setenv("RECONCILER_IDENTITY_ANS_TIMEOUT", "3s")
	t.Setenv("RECONCILER_IDENTITY_ANS_DNS_SERVER", "127.0.0.1:5353")
	t.Setenv("RECONCILER_IDENTITY_ANS_CA_FILE", "/etc/agntcy/ans-log-ca.pem")

	cfg, err := LoadConfig()
	require.NoError(t, err)

	assert.True(t, cfg.Identity.Enabled)
	assert.Equal(t, 2*time.Minute, cfg.Identity.Interval)
	assert.Equal(t, ansconfig.Config{
		Enabled:               true,
		TrustedLogHosts:       []string{"log-a.example.com:8443", "log-b.example.com"},
		RootKeys:              []string{"log-a.example.com+abcd1234+AAAA", "log-b.example.com+ef567890+BBBB"},
		AllowUnpinnedRootKeys: true,
		Timeout:               3 * time.Second,
		DNSServer:             "127.0.0.1:5353",
		CAFile:                "/etc/agntcy/ans-log-ca.pem",
	}, cfg.Identity.Ans)
}
