// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"testing"
	"time"

	"github.com/agntcy/dir/reconciler/tasks/identity"
	dbconfig "github.com/agntcy/dir/server/database/config"
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
	assert.False(t, cfg.Name.Enabled)

	// The identity task is off unless asked for.
	assert.False(t, cfg.Identity.Enabled)
	assert.Equal(t, identity.DefaultInterval, cfg.Identity.Interval)
	assert.Equal(t, identity.DefaultRecordTimeout, cfg.Identity.RecordTimeout)
	assert.Empty(t, cfg.Identity.SPIFFETrustBundles)
}

func TestLoadConfig_EnvOverrides(t *testing.T) {
	t.Setenv("RECONCILER_REGSYNC_ENABLED", "false")
	t.Setenv("RECONCILER_INDEXER_ENABLED", "false")
	t.Setenv("RECONCILER_NAME_ENABLED", "true")
	t.Setenv("RECONCILER_INDEXER_INTERVAL", "2h")
	t.Setenv("RECONCILER_NAME_INTERVAL", "30m")

	cfg, err := LoadConfig()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.False(t, cfg.Regsync.Enabled)
	assert.False(t, cfg.Indexer.Enabled)
	assert.True(t, cfg.Name.Enabled)
	assert.Equal(t, 2*time.Hour, cfg.Indexer.Interval)
	assert.Equal(t, 30*time.Minute, cfg.Name.Interval)
}

// The task must be switchable with environment variables alone, with no config
// file present.
func TestLoadConfig_IdentityFromEnv(t *testing.T) {
	t.Setenv("RECONCILER_IDENTITY_ENABLED", "true")
	t.Setenv("RECONCILER_IDENTITY_INTERVAL", "5m")
	t.Setenv("RECONCILER_IDENTITY_RECORD_TIMEOUT", "10s")

	cfg, err := LoadConfig()
	require.NoError(t, err)

	assert.True(t, cfg.Identity.Enabled)
	assert.Equal(t, 5*time.Minute, cfg.Identity.Interval)
	assert.Equal(t, 10*time.Second, cfg.Identity.RecordTimeout)
}
