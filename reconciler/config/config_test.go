// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"testing"
	"time"

	dbconfig "github.com/agntcy/dir/server/database/config"
	ociconfig "github.com/agntcy/dir/server/store/oci/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigConstants(t *testing.T) {
	assert.Equal(t, "RECONCILER", DefaultEnvPrefix)
	assert.Equal(t, "reconciler.config", DefaultConfigName)
	assert.Equal(t, "yml", DefaultConfigType)
	assert.Equal(t, "/etc/agntcy/reconciler", DefaultConfigPath)
}

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

func TestConfig_StructDefaults(t *testing.T) {
	cfg, err := LoadConfig()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Local registry config exists (zero value or from env)
	assert.Equal(t, ociconfig.Config{}, cfg.LocalRegistry)
}
