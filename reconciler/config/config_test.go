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

func TestLoadConfig_Identity(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want identity.Config
	}{
		{
			name: "defaults leave the task off",
			want: identity.Config{Interval: identity.DefaultInterval},
		},
		{
			name: "environment enables the task and sets the interval",
			env: map[string]string{
				"RECONCILER_IDENTITY_ENABLED":  "true",
				"RECONCILER_IDENTITY_INTERVAL": "2m",
			},
			want: identity.Config{Enabled: true, Interval: 2 * time.Minute},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for key, value := range tt.env {
				t.Setenv(key, value)
			}

			cfg, err := LoadConfig()
			require.NoError(t, err)

			assert.Equal(t, tt.want, cfg.Identity)
		})
	}
}
