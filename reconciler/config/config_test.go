// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"testing"
	"time"

	"github.com/agntcy/dir/reconciler/recordevents"
	"github.com/agntcy/dir/reconciler/tasks/identity"
	policytask "github.com/agntcy/dir/reconciler/tasks/policy"
	synctask "github.com/agntcy/dir/reconciler/tasks/sync"
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
	assert.False(t, cfg.Prune.Enabled)
	assert.Equal(t, 30*time.Minute, cfg.Prune.Interval)
	assert.False(t, cfg.Prune.Criteria.Trusted)
	assert.Equal(t, "MEDIUM", cfg.Prune.Criteria.MinSeverity)
	assert.Equal(t, 168*time.Hour, cfg.Prune.Criteria.OlderThan)
	assert.Equal(t, 100, cfg.Prune.Limit)
	assert.True(t, cfg.Prune.DryRun)

	// Sync is off unless asked for; dry-run defaults on.
	assert.False(t, cfg.Sync.Enabled)
	assert.Equal(t, synctask.DefaultInterval, cfg.Sync.Interval)
	assert.Equal(t, synctask.DefaultDomain, cfg.Sync.Criteria.Domain)
	assert.Equal(t, synctask.DefaultLimit, cfg.Sync.Limit)
	assert.True(t, cfg.Sync.DryRun)

	// The identity task is off unless asked for.
	assert.False(t, cfg.Identity.Enabled)
	assert.Equal(t, identity.DefaultInterval, cfg.Identity.Interval)
	assert.Equal(t, identity.DefaultRecordTimeout, cfg.Identity.RecordTimeout)
	assert.Empty(t, cfg.Identity.SPIFFETrustBundles)

	// Policy parameters come from the task's own defaults, not a second copy.
	assert.False(t, cfg.PolicyEvaluation.Enabled)
	assert.Equal(t, policytask.DefaultInterval, cfg.PolicyEvaluation.Interval)
	assert.Equal(t, policytask.DefaultRecordTimeout, cfg.PolicyEvaluation.RecordTimeout)
	assert.Equal(t, policytask.DefaultBatchSize, cfg.PolicyEvaluation.BatchSize)

	// Hearing of a push is on by default, with the package's own parameters.
	assert.True(t, cfg.RecordEvents.Enabled)
	assert.Equal(t, recordevents.DefaultWindow, cfg.RecordEvents.Window)
	assert.Equal(t, recordevents.DefaultReconnectDelay, cfg.RecordEvents.ReconnectDelay)
}

func TestLoadConfig_RecordEventsEnvOverrides(t *testing.T) {
	t.Setenv("RECONCILER_RECORD_EVENTS_ENABLED", "false")
	t.Setenv("RECONCILER_RECORD_EVENTS_WINDOW", "7s")
	t.Setenv("RECONCILER_RECORD_EVENTS_RECONNECT_DELAY", "20s")

	cfg, err := LoadConfig()
	require.NoError(t, err)

	assert.False(t, cfg.RecordEvents.Enabled)
	assert.Equal(t, 7*time.Second, cfg.RecordEvents.Window)
	assert.Equal(t, 20*time.Second, cfg.RecordEvents.ReconnectDelay)
}

func TestLoadConfig_EnvOverrides(t *testing.T) {
	t.Setenv("RECONCILER_REGSYNC_ENABLED", "false")
	t.Setenv("RECONCILER_INDEXER_ENABLED", "false")
	t.Setenv("RECONCILER_INDEXER_INTERVAL", "2h")
	t.Setenv("RECONCILER_POLICY_EVALUATION_ENABLED", "true")
	t.Setenv("RECONCILER_POLICY_EVALUATION_INTERVAL", "5m")
	t.Setenv("RECONCILER_POLICY_EVALUATION_RECORD_TIMEOUT", "45s")
	t.Setenv("RECONCILER_POLICY_EVALUATION_BATCH_SIZE", "250")
	t.Setenv("RECONCILER_PRUNE_ENABLED", "true")
	t.Setenv("RECONCILER_PRUNE_INTERVAL", "15m")
	t.Setenv("RECONCILER_PRUNE_CRITERIA_TRUSTED", "true")
	t.Setenv("RECONCILER_PRUNE_CRITERIA_MIN_SEVERITY", "HIGH")
	t.Setenv("RECONCILER_PRUNE_CRITERIA_OLDER_THAN", "48h")
	t.Setenv("RECONCILER_PRUNE_LIMIT", "25")
	t.Setenv("RECONCILER_PRUNE_DRY_RUN", "false")

	cfg, err := LoadConfig()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.False(t, cfg.Regsync.Enabled)
	assert.False(t, cfg.Indexer.Enabled)
	assert.Equal(t, 2*time.Hour, cfg.Indexer.Interval)
	assert.True(t, cfg.PolicyEvaluation.Enabled)
	assert.Equal(t, 5*time.Minute, cfg.PolicyEvaluation.Interval)
	assert.Equal(t, 45*time.Second, cfg.PolicyEvaluation.RecordTimeout)
	assert.Equal(t, 250, cfg.PolicyEvaluation.BatchSize)
	assert.True(t, cfg.Prune.Enabled)
	assert.Equal(t, 15*time.Minute, cfg.Prune.Interval)
	assert.True(t, cfg.Prune.Criteria.Trusted)
	assert.Equal(t, "HIGH", cfg.Prune.Criteria.MinSeverity)
	assert.Equal(t, 48*time.Hour, cfg.Prune.Criteria.OlderThan)
	assert.Equal(t, 25, cfg.Prune.Limit)
	assert.False(t, cfg.Prune.DryRun)
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

func TestLoadConfig_SyncFromEnv(t *testing.T) {
	t.Setenv("RECONCILER_SYNC_ENABLED", "true")
	t.Setenv("RECONCILER_SYNC_INTERVAL", "6h")
	t.Setenv("RECONCILER_SYNC_CRITERIA_DOMAIN", "research")
	t.Setenv("RECONCILER_SYNC_LIMIT", "25")
	t.Setenv("RECONCILER_SYNC_DRY_RUN", "false")

	cfg, err := LoadConfig()
	require.NoError(t, err)

	assert.True(t, cfg.Sync.Enabled)
	assert.Equal(t, 6*time.Hour, cfg.Sync.Interval)
	assert.Equal(t, "research", cfg.Sync.Criteria.Domain)
	assert.Equal(t, 25, cfg.Sync.Limit)
	assert.False(t, cfg.Sync.DryRun)
}
