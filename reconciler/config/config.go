// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package config handles configuration loading for the reconciler service.
package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/agntcy/dir/reconciler/recordevents"
	"github.com/agntcy/dir/reconciler/tasks/identity"
	"github.com/agntcy/dir/reconciler/tasks/indexer"
	"github.com/agntcy/dir/reconciler/tasks/metrics"
	policytask "github.com/agntcy/dir/reconciler/tasks/policy"
	"github.com/agntcy/dir/reconciler/tasks/prune"
	"github.com/agntcy/dir/reconciler/tasks/regsync"
	"github.com/agntcy/dir/reconciler/tasks/scan"
	"github.com/agntcy/dir/reconciler/tasks/signature"
	synctask "github.com/agntcy/dir/reconciler/tasks/sync"
	authnconfig "github.com/agntcy/dir/server/authn/config"
	dbconfig "github.com/agntcy/dir/server/database/config"
	policy "github.com/agntcy/dir/server/policy/config"
	ociconfig "github.com/agntcy/dir/server/store/oci/config"
	validators "github.com/agntcy/dir/server/validators/config"
	"github.com/agntcy/dir/utils/logging"
	"github.com/spf13/viper"
)

const (
	// DefaultEnvPrefix is the environment variable prefix.
	DefaultEnvPrefix = "RECONCILER"

	// DefaultConfigName is the default configuration file name.
	DefaultConfigName = "reconciler.config"

	// DefaultConfigType is the default configuration file type.
	DefaultConfigType = "yml"

	// DefaultConfigPath is the default configuration file path.
	DefaultConfigPath = "/etc/agntcy/reconciler"
)

var logger = logging.Logger("reconciler/config")

// Config holds the reconciler configuration.
type Config struct {
	// Database holds PostgreSQL connection configuration.
	Database dbconfig.Config `json:"database" mapstructure:"database"`

	// LocalRegistry holds configuration for the local OCI registry.
	LocalRegistry ociconfig.Config `json:"local_registry" mapstructure:"local_registry"`

	// ServerAddress is the gRPC address of the apiserver (e.g. "localhost:8888").
	// Required by the metrics and sync tasks when the reconciler runs as
	// a standalone process, because the routing layer (Badger datastore) is
	// embedded in the server and cannot be shared across process boundaries.
	// Those tasks call RoutingService over gRPC instead.
	// Leave empty in daemon mode — the in-process routing API is used directly.
	ServerAddress string `json:"server_address" mapstructure:"server_address"`

	// ServerAuthn holds authentication configuration for the gRPC connection to
	// the apiserver.
	ServerAuthn authnconfig.Config `json:"server_authn" mapstructure:"server_authn"`

	// Validators is the same list the server uses. The indexer consults
	// entries whose op includes "index". YAML only.
	Validators validators.Config `json:"validators,omitempty" mapstructure:"validators"`

	// Policy is the directory named OPA policy files are loaded from.
	// Same path the server uses (policy.dir).
	Policy policy.Config `json:"policy" mapstructure:"policy"`

	// Regsync holds the regsync task configuration.
	Regsync regsync.Config `json:"regsync" mapstructure:"regsync"`

	// Indexer holds the indexer task configuration.
	Indexer indexer.Config `json:"indexer" mapstructure:"indexer"`

	// Signature holds the signature verification task configuration.
	Signature signature.Config `json:"signature" mapstructure:"signature"`

	// Scan holds the security scan task configuration.
	Scan scan.Config `json:"scan" mapstructure:"scan"`

	// Prune holds the prune task configuration.
	// Deletes records that match Criteria. Disabled by default because it
	// deletes records.
	Prune prune.Config `json:"prune" mapstructure:"prune"`

	// Sync holds the sync task configuration.
	// Searches routing for records matching Criteria and creates one sync
	// per announcing peer. Disabled by default because it creates remote syncs.
	Sync synctask.Config `json:"sync" mapstructure:"sync"`

	// Identity holds the identity claim verification task configuration.
	Identity identity.Config `json:"identity" mapstructure:"identity"`

	// Metrics holds the usage-metrics refresh task configuration.
	Metrics metrics.Config `json:"metrics" mapstructure:"metrics"`

	// PolicyEvaluation holds the policy evaluation task configuration. It is
	// not under policy, which is the policy directory the server uses too.
	PolicyEvaluation policytask.Config `json:"policy_evaluation" mapstructure:"policy_evaluation"`

	// RecordEvents holds how the reconciler reacts to records arriving on the
	// server, so the indexer and the policy task run when there is something
	// for them instead of at their next interval.
	RecordEvents recordevents.Config `json:"record_events" mapstructure:"record_events"`
}

// LoadConfig loads the configuration from file and environment variables.
func LoadConfig() (*Config, error) {
	v := viper.NewWithOptions(
		viper.KeyDelimiter("."),
		viper.EnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_")),
	)

	v.SetConfigName(DefaultConfigName)
	v.SetConfigType(DefaultConfigType)
	v.AddConfigPath(DefaultConfigPath)

	v.SetEnvPrefix(DefaultEnvPrefix)
	v.AllowEmptyEnv(true)
	v.AutomaticEnv()

	// Read the config file
	if err := v.ReadInConfig(); err != nil {
		fileNotFoundError := viper.ConfigFileNotFoundError{}
		if errors.As(err, &fileNotFoundError) {
			logger.Info("Config file not found, using defaults")
		} else {
			return nil, fmt.Errorf("failed to read configuration file: %w", err)
		}
	}

	//
	// Database configuration
	//
	_ = v.BindEnv("database.type")
	v.SetDefault("database.type", dbconfig.DefaultType)

	// SQLite configuration
	_ = v.BindEnv("database.sqlite.path")
	v.SetDefault("database.sqlite.path", dbconfig.DefaultSQLitePath)

	// PostgreSQL configuration
	_ = v.BindEnv("database.postgres.host")
	v.SetDefault("database.postgres.host", dbconfig.DefaultPostgresHost)

	_ = v.BindEnv("database.postgres.port")
	v.SetDefault("database.postgres.port", dbconfig.DefaultPostgresPort)

	_ = v.BindEnv("database.postgres.database")
	v.SetDefault("database.postgres.database", dbconfig.DefaultPostgresDatabase)

	_ = v.BindEnv("database.postgres.username")
	_ = v.BindEnv("database.postgres.password")

	_ = v.BindEnv("database.postgres.ssl_mode")
	v.SetDefault("database.postgres.ssl_mode", dbconfig.DefaultPostgresSSLMode)

	//
	// Local registry configuration (shared by all tasks)
	//
	_ = v.BindEnv("local_registry.registry_address")
	_ = v.BindEnv("local_registry.repository_name")
	_ = v.BindEnv("local_registry.auth_config.username")
	_ = v.BindEnv("local_registry.auth_config.password")
	_ = v.BindEnv("local_registry.auth_config.insecure")

	//
	// Regsync task configuration
	//
	_ = v.BindEnv("regsync.enabled")
	v.SetDefault("regsync.enabled", true)

	_ = v.BindEnv("regsync.interval")
	v.SetDefault("regsync.interval", regsync.DefaultInterval)

	_ = v.BindEnv("regsync.timeout")
	v.SetDefault("regsync.timeout", regsync.DefaultTimeout)

	//
	// Authentication configuration for registry credentials provider
	//
	_ = v.BindEnv("regsync.authn.enabled")
	v.SetDefault("regsync.authn.enabled", false)

	_ = v.BindEnv("regsync.authn.mode")
	v.SetDefault("regsync.authn.mode", "x509")

	_ = v.BindEnv("regsync.authn.socket_path")
	_ = v.BindEnv("regsync.authn.audiences")

	//
	// Indexer task configuration
	//
	_ = v.BindEnv("indexer.enabled")
	v.SetDefault("indexer.enabled", true)

	_ = v.BindEnv("indexer.interval")
	v.SetDefault("indexer.interval", indexer.DefaultInterval)

	//
	// Signature task configuration (signature verification cache)
	//
	_ = v.BindEnv("signature.enabled")
	v.SetDefault("signature.enabled", true)

	_ = v.BindEnv("signature.interval")
	v.SetDefault("signature.interval", signature.DefaultInterval)

	_ = v.BindEnv("signature.ttl")
	v.SetDefault("signature.ttl", signature.DefaultTTL)

	_ = v.BindEnv("signature.record_timeout")
	v.SetDefault("signature.record_timeout", signature.DefaultRecordTimeout)

	//
	// Identity task configuration (identity and ownership claim verification).
	// The trust bundles for spiffe:// claims are a list, which has no
	// environment variable form: set identity.spiffe_trust_bundles in YAML.
	//
	_ = v.BindEnv("identity.enabled")
	v.SetDefault("identity.enabled", false)

	_ = v.BindEnv("identity.interval")
	v.SetDefault("identity.interval", identity.DefaultInterval)

	_ = v.BindEnv("identity.record_timeout")
	v.SetDefault("identity.record_timeout", identity.DefaultRecordTimeout)

	//
	// Scan task configuration (security scanning)
	//
	_ = v.BindEnv("scan.enabled")
	v.SetDefault("scan.enabled", false)

	_ = v.BindEnv("scan.interval")
	v.SetDefault("scan.interval", scan.DefaultInterval)

	_ = v.BindEnv("scan.ttl")
	v.SetDefault("scan.ttl", scan.DefaultTTL)

	_ = v.BindEnv("scan.record_timeout")
	v.SetDefault("scan.record_timeout", scan.DefaultRecordTimeout)

	_ = v.BindEnv("scan.mcp_cli_path")
	v.SetDefault("scan.mcp_cli_path", scan.DefaultMCPCLIPath)

	_ = v.BindEnv("scan.skill_cli_path")
	v.SetDefault("scan.skill_cli_path", scan.DefaultSkillCLIPath)

	_ = v.BindEnv("scan.a2a_cli_path")
	v.SetDefault("scan.a2a_cli_path", scan.DefaultA2ACLIPath)

	//
	// Prune task configuration
	//
	_ = v.BindEnv("prune.enabled")
	v.SetDefault("prune.enabled", false)

	_ = v.BindEnv("prune.interval")
	v.SetDefault("prune.interval", prune.DefaultInterval)

	_ = v.BindEnv("prune.record_timeout")
	v.SetDefault("prune.record_timeout", prune.DefaultRecordTimeout)

	_ = v.BindEnv("prune.criteria.trusted")
	v.SetDefault("prune.criteria.trusted", false)

	_ = v.BindEnv("prune.criteria.min_severity")
	v.SetDefault("prune.criteria.min_severity", prune.DefaultMinSeverity)

	_ = v.BindEnv("prune.criteria.older_than")
	v.SetDefault("prune.criteria.older_than", prune.DefaultOlderThan)

	_ = v.BindEnv("prune.limit")
	v.SetDefault("prune.limit", prune.DefaultLimit)

	_ = v.BindEnv("prune.dry_run")
	v.SetDefault("prune.dry_run", true)

	//
	// Sync task configuration
	//
	_ = v.BindEnv("sync.enabled")
	v.SetDefault("sync.enabled", false)

	_ = v.BindEnv("sync.interval")
	v.SetDefault("sync.interval", synctask.DefaultInterval)

	_ = v.BindEnv("sync.criteria.domain")
	v.SetDefault("sync.criteria.domain", synctask.DefaultDomain)

	_ = v.BindEnv("sync.limit")
	v.SetDefault("sync.limit", synctask.DefaultLimit)

	_ = v.BindEnv("sync.dry_run")
	v.SetDefault("sync.dry_run", true)

	//
	// Providers task configuration (provider-count gauge)
	//
	_ = v.BindEnv("metrics.enabled")
	v.SetDefault("metrics.enabled", true)

	_ = v.BindEnv("metrics.interval")
	v.SetDefault("metrics.interval", metrics.DefaultInterval)

	//
	// Policy evaluation task configuration (content-policy verdicts)
	//
	_ = v.BindEnv("policy_evaluation.enabled")
	v.SetDefault("policy_evaluation.enabled", false)

	_ = v.BindEnv("policy_evaluation.interval")
	v.SetDefault("policy_evaluation.interval", policytask.DefaultInterval)

	_ = v.BindEnv("policy_evaluation.record_timeout")
	v.SetDefault("policy_evaluation.record_timeout", policytask.DefaultRecordTimeout)

	_ = v.BindEnv("policy_evaluation.batch_size")
	v.SetDefault("policy_evaluation.batch_size", policytask.DefaultBatchSize)

	//
	// Record events (waking the indexer when a record is pushed)
	//
	_ = v.BindEnv("record_events.enabled")
	v.SetDefault("record_events.enabled", true)

	_ = v.BindEnv("record_events.window")
	v.SetDefault("record_events.window", recordevents.DefaultWindow)

	_ = v.BindEnv("record_events.reconnect_delay")
	v.SetDefault("record_events.reconnect_delay", recordevents.DefaultReconnectDelay)

	//
	// Server address (used by the metrics and sync tasks in standalone mode)
	//
	_ = v.BindEnv("server_address")

	//
	// Server authn configuration (used by the metrics task in standalone mode)
	//
	_ = v.BindEnv("server_authn.enabled")
	v.SetDefault("server_authn.enabled", false)

	_ = v.BindEnv("server_authn.mode")
	v.SetDefault("server_authn.mode", "x509")

	_ = v.BindEnv("server_authn.socket_path")
	_ = v.BindEnv("server_authn.audiences")

	//
	// Policy directory (named .rego files loaded by opa validators)
	//
	_ = v.BindEnv("policy.dir")
	v.SetDefault("policy.dir", "/etc/agntcy/dir/policies")

	// Unmarshal into config struct
	config := &Config{}
	if err := v.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	return config, nil
}
