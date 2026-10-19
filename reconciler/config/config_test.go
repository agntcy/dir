// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"testing"
	"time"

	ansresolver "github.com/agntcy/dir/client/utils/identity/resolvers/ans"
	"github.com/agntcy/dir/reconciler/recordevents"
	"github.com/agntcy/dir/reconciler/tasks/identity"
	policytask "github.com/agntcy/dir/reconciler/tasks/policy"
	"github.com/agntcy/dir/reconciler/tasks/publish"
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
	assert.Nil(t, cfg.Prune.Criteria.Trusted, "trusted has no default: it excludes signature_statuses")
	assert.False(t, cfg.Prune.GetTrusted())
	assert.Empty(t, cfg.Prune.Criteria.SignatureStatuses)
	assert.Equal(t, "MEDIUM", cfg.Prune.Criteria.MinSeverity)
	assert.Equal(t, 168*time.Hour, cfg.Prune.Criteria.OlderThan)
	assert.Equal(t, 100, cfg.Prune.Limit)
	assert.True(t, cfg.Prune.DryRun)

	// Sync is off unless asked for; dry-run defaults on.
	assert.False(t, cfg.Sync.Enabled)
	assert.Equal(t, synctask.DefaultInterval, cfg.Sync.Interval)
	assert.Equal(t, []string{synctask.DefaultDomain}, cfg.Sync.Criteria.Domains)
	assert.Empty(t, cfg.Sync.Criteria.Skills)
	assert.Equal(t, synctask.DefaultLimit, cfg.Sync.Limit)
	assert.True(t, cfg.Sync.DryRun)

	// Publish is off unless asked for; dry-run defaults on. Empty domains
	// means every indexed record.
	assert.False(t, cfg.Publish.Enabled)
	assert.Equal(t, publish.DefaultInterval, cfg.Publish.Interval)
	assert.Empty(t, cfg.Publish.Criteria.Domains)
	assert.Empty(t, cfg.Publish.Criteria.Skills)
	assert.Equal(t, publish.DefaultLimit, cfg.Publish.Limit)
	assert.True(t, cfg.Publish.DryRun)

	// The identity task is off unless asked for.
	assert.False(t, cfg.Identity.Enabled)
	assert.Equal(t, identity.DefaultInterval, cfg.Identity.Interval)
	assert.Equal(t, identity.DefaultRecordTimeout, cfg.Identity.RecordTimeout)
	assert.Empty(t, cfg.Identity.SPIFFETrustBundles)

	// The ans scheme is off, with the resolver's own defaults.
	assert.False(t, cfg.Identity.ANS.Enabled)
	assert.Empty(t, cfg.Identity.ANS.TrustedLogHosts)
	assert.Empty(t, cfg.Identity.ANS.RootKeys)
	assert.False(t, cfg.Identity.ANS.AllowUnpinnedRootKeys)
	assert.Equal(t, ansresolver.DefaultRootKeysTTL, cfg.Identity.ANS.RootKeysTTL)
	assert.Equal(t, ansresolver.DefaultStatusCacheTTL, cfg.Identity.ANS.StatusCacheTTL)
	assert.Equal(t, identity.DefaultANSStaleGrace, cfg.Identity.ANS.StaleGrace)
	assert.Equal(t, ansresolver.DefaultTimeout, cfg.Identity.ANS.Timeout)
	assert.Equal(t, ansresolver.DefaultClockSkew, cfg.Identity.ANS.ClockSkew)

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
	require.NotNil(t, cfg.Prune.Criteria.Trusted)
	assert.True(t, *cfg.Prune.Criteria.Trusted)
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

// The ans block is switchable from the environment too; its lists are
// comma-separated.
func TestLoadConfig_ANSFromEnv(t *testing.T) {
	t.Setenv("RECONCILER_IDENTITY_ANS_ENABLED", "true")
	t.Setenv("RECONCILER_IDENTITY_ANS_TRUSTED_LOG_HOSTS", "log.example.com,log2.example.com:8443")
	t.Setenv("RECONCILER_IDENTITY_ANS_ROOT_KEYS", "example-log+1a2b3c4d+AjBZ,other-log+5e6f7a8b+AjBZ")
	t.Setenv("RECONCILER_IDENTITY_ANS_ALLOW_UNPINNED_ROOT_KEYS", "true")
	t.Setenv("RECONCILER_IDENTITY_ANS_ROOT_KEYS_TTL", "5m")
	t.Setenv("RECONCILER_IDENTITY_ANS_STATUS_CACHE_TTL", "20s")
	t.Setenv("RECONCILER_IDENTITY_ANS_STALE_GRACE", "2h")
	t.Setenv("RECONCILER_IDENTITY_ANS_TIMEOUT", "3s")
	t.Setenv("RECONCILER_IDENTITY_ANS_CLOCK_SKEW", "5s")

	cfg, err := LoadConfig()
	require.NoError(t, err)

	ans := cfg.Identity.ANS
	assert.True(t, ans.Enabled)
	assert.Equal(t, []string{"log.example.com", "log2.example.com:8443"}, ans.TrustedLogHosts)
	assert.Equal(t, []string{"example-log+1a2b3c4d+AjBZ", "other-log+5e6f7a8b+AjBZ"}, ans.RootKeys)
	assert.True(t, ans.AllowUnpinnedRootKeys)
	assert.Equal(t, 5*time.Minute, ans.RootKeysTTL)
	assert.Equal(t, 20*time.Second, ans.StatusCacheTTL)
	assert.Equal(t, 2*time.Hour, ans.StaleGrace)
	assert.Equal(t, 3*time.Second, ans.Timeout)
	assert.Equal(t, 5*time.Second, ans.ClockSkew)
}

func TestLoadConfig_SyncFromEnv(t *testing.T) {
	t.Setenv("RECONCILER_SYNC_ENABLED", "true")
	t.Setenv("RECONCILER_SYNC_INTERVAL", "6h")
	t.Setenv("RECONCILER_SYNC_CRITERIA_DOMAINS", "research,analytics")
	t.Setenv("RECONCILER_SYNC_CRITERIA_SKILLS", "threat_detection,nlp")
	t.Setenv("RECONCILER_SYNC_LIMIT", "25")
	t.Setenv("RECONCILER_SYNC_DRY_RUN", "false")

	cfg, err := LoadConfig()
	require.NoError(t, err)

	assert.True(t, cfg.Sync.Enabled)
	assert.Equal(t, 6*time.Hour, cfg.Sync.Interval)
	assert.Equal(t, []string{"research", "analytics"}, cfg.Sync.Criteria.Domains)
	assert.Equal(t, []string{"threat_detection", "nlp"}, cfg.Sync.Criteria.Skills)
	assert.Equal(t, 25, cfg.Sync.Limit)
	assert.False(t, cfg.Sync.DryRun)
}

// The env form of signature_statuses is comma-separated.
func TestLoadConfig_PruneSignatureStatusesFromEnv(t *testing.T) {
	t.Setenv("RECONCILER_PRUNE_CRITERIA_SIGNATURE_STATUSES", "failed,unsigned")

	cfg, err := LoadConfig()
	require.NoError(t, err)

	assert.Equal(t, []string{"failed", "unsigned"}, cfg.Prune.Criteria.SignatureStatuses)
	assert.Nil(t, cfg.Prune.Criteria.Trusted)
	assert.NoError(t, cfg.Prune.Validate())
}

func TestLoadConfig_PublishFromEnv(t *testing.T) {
	t.Setenv("RECONCILER_PUBLISH_ENABLED", "true")
	t.Setenv("RECONCILER_PUBLISH_INTERVAL", "12h")
	t.Setenv("RECONCILER_PUBLISH_CRITERIA_DOMAINS", "finance,operations")
	t.Setenv("RECONCILER_PUBLISH_CRITERIA_SKILLS", "forecasting,risk,compliance")
	t.Setenv("RECONCILER_PUBLISH_LIMIT", "10")
	t.Setenv("RECONCILER_PUBLISH_DRY_RUN", "false")

	cfg, err := LoadConfig()
	require.NoError(t, err)

	assert.True(t, cfg.Publish.Enabled)
	assert.Equal(t, 12*time.Hour, cfg.Publish.Interval)
	assert.Equal(t, []string{"finance", "operations"}, cfg.Publish.Criteria.Domains)
	assert.Equal(t, []string{"forecasting", "risk", "compliance"}, cfg.Publish.Criteria.Skills)
	assert.Equal(t, 10, cfg.Publish.Limit)
	assert.False(t, cfg.Publish.DryRun)
}
