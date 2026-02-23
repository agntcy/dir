// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	authn "github.com/agntcy/dir/server/authn/config"
	authz "github.com/agntcy/dir/server/authz/config"
	dbconfig "github.com/agntcy/dir/server/database/config"
	events "github.com/agntcy/dir/server/events/config"
	ratelimitconfig "github.com/agntcy/dir/server/middleware/ratelimit/config"
	naming "github.com/agntcy/dir/server/naming/config"
	publication "github.com/agntcy/dir/server/publication/config"
	routing "github.com/agntcy/dir/server/routing/config"
	store "github.com/agntcy/dir/server/store/config"
	oci "github.com/agntcy/dir/server/store/oci/config"
	"github.com/agntcy/dir/utils/logging"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

const (
	// Config params.

	DefaultEnvPrefix  = "DIRECTORY_SERVER"
	DefaultConfigName = "server.config"
	DefaultConfigType = "yml"
	DefaultConfigPath = "/etc/agntcy/dir"

	// API configuration.

	DefaultListenAddress = "0.0.0.0:8888"

	// OASF Validation configuration.

	// Connection management configuration.
	// These defaults are based on production gRPC best practices and provide
	// a balance between resource efficiency and connection stability.

	// DefaultMaxConcurrentStreams limits concurrent RPC streams per connection.
	// This prevents a single connection from monopolizing server resources.
	// Value: 1000 is industry standard, sufficient for most clients.
	DefaultMaxConcurrentStreams = 1000

	// DefaultMaxRecvMsgSize limits maximum received message size (4 MB).
	// Protects against memory exhaustion from large messages.
	// Value: 4 MB covers 99% of OCI artifacts and metadata.
	DefaultMaxRecvMsgSize = 4 * 1024 * 1024

	// DefaultMaxSendMsgSize limits maximum sent message size (4 MB).
	// Value: 4 MB matches receive limit for consistency.
	DefaultMaxSendMsgSize = 4 * 1024 * 1024

	// DefaultConnectionTimeout limits time for connection establishment (120 seconds).
	// Prevents hanging connection attempts from slow clients.
	// Value: 2 minutes allows for slow networks without wasting resources.
	DefaultConnectionTimeout = 120 * time.Second

	// DefaultMaxConnectionIdle closes idle connections after this duration (15 minutes).
	// An idle connection has no active RPC streams.
	// Value: 15 minutes balances resource cleanup vs connection churn.
	DefaultMaxConnectionIdle = 15 * time.Minute

	// DefaultMaxConnectionAge forces connection rotation after this duration (30 minutes).
	// Prevents long-lived connections from accumulating issues.
	// Value: 30 minutes ensures regular TLS session rotation for security.
	DefaultMaxConnectionAge = 30 * time.Minute

	// DefaultMaxConnectionAgeGrace is grace period after MaxConnectionAge (5 minutes).
	// Allows inflight RPCs to complete before force-closing connection.
	// Value: 5 minutes provides sufficient time for most operations.
	DefaultMaxConnectionAgeGrace = 5 * time.Minute

	// DefaultKeepaliveTime is interval for sending keepalive pings (5 minutes).
	// Detects dead connections when client crashes or network partitions.
	// Value: 5 minutes detects issues fast without excessive traffic.
	DefaultKeepaliveTime = 5 * time.Minute

	// DefaultKeepaliveTimeout is wait time for keepalive ping response (1 minute).
	// Connection is closed if no pong received within this timeout.
	// Value: 1 minute allows for network delays without long waits.
	DefaultKeepaliveTimeout = 1 * time.Minute

	// DefaultMinTime is minimum time between client keepalive pings (1 minute).
	// Prevents clients from abusing keepalive by sending excessive pings.
	// Value: 1 minute prevents abuse while allowing reasonable client detection.
	DefaultMinTime = 1 * time.Minute

	// DefaultPermitWithoutStream allows keepalive pings without active streams.
	// Enables clients to detect dead connections even when idle.
	// Value: true provides better connection health detection.
	DefaultPermitWithoutStream = true

	// Metrics configuration.

	// DefaultMetricsEnabled enables Prometheus metrics collection.
	DefaultMetricsEnabled = true

	// DefaultMetricsAddress is the default listen address for the metrics HTTP server.
	DefaultMetricsAddress = ":9090"
)

var logger = logging.Logger("config")

type Config struct {
	// API configuration
	ListenAddress string `json:"listen_address,omitempty" mapstructure:"listen_address"`

	// OASF Validation configuration
	OASFAPIValidation OASFAPIValidationConfig `json:"oasf_api_validation" mapstructure:"oasf_api_validation"`

	// Logging configuration
	Logging LoggingConfig `json:"logging" mapstructure:"logging"`

	// Connection management configuration
	Connection ConnectionConfig `json:"connection" mapstructure:"connection"`

	// Rate limiting configuration
	RateLimit ratelimitconfig.Config `json:"ratelimit" mapstructure:"ratelimit"`

	// Authn configuration (JWT or X.509 authentication)
	Authn authn.Config `json:"authn" mapstructure:"authn"`

	// Authz configuration
	Authz authz.Config `json:"authz" mapstructure:"authz"`

	// Store configuration
	Store store.Config `json:"store" mapstructure:"store"`

	// Routing configuration
	Routing routing.Config `json:"routing" mapstructure:"routing"`

	// Database configuration
	Database dbconfig.Config `json:"database" mapstructure:"database"`

	// Sync configuration
	Sync SyncConfig `json:"sync" mapstructure:"sync"`

	// Publication configuration
	Publication publication.Config `json:"publication" mapstructure:"publication"`

	// Events configuration
	Events events.Config `json:"events" mapstructure:"events"`

	// Metrics configuration
	Metrics MetricsConfig `json:"metrics" mapstructure:"metrics"`

	// Naming holds name verification cache config (TTL for naming API; reconciler name task performs re-verification).
	Naming naming.Config `json:"naming,omitzero" mapstructure:"naming"`
}

type SyncConfig struct {
	// AuthConfig holds authentication configuration for sync operations.
	AuthConfig oci.AuthConfig `json:"auth_config" mapstructure:"auth_config"`
}

// OASFAPIValidationConfig defines OASF API validation configuration.
type OASFAPIValidationConfig struct {
	// SchemaURL is the OASF schema URL for API-based validation.
	// This is required - records will be validated using the OASF API validator.
	// The default value is set in the Helm chart values.yaml (apiserver.config.oasf_api_validation.schema_url).
	SchemaURL string `json:"schema_url,omitempty" mapstructure:"schema_url"`
}

// LoggingConfig defines gRPC request/response logging configuration.
type LoggingConfig struct {
	// Verbose enables verbose logging mode (includes request/response payloads).
	// Default: false (production mode - logs only start/finish with metadata).
	Verbose bool `json:"verbose,omitempty" mapstructure:"verbose"`
}

// ConnectionConfig defines gRPC connection management configuration.
// These settings control connection lifecycle, resource limits, and keepalive behavior
// to prevent resource exhaustion and detect dead connections.
type ConnectionConfig struct {
	// MaxConcurrentStreams limits concurrent RPCs per connection.
	// Prevents a single connection from monopolizing server resources.
	// Default: 1000
	MaxConcurrentStreams uint32 `json:"max_concurrent_streams,omitempty" mapstructure:"max_concurrent_streams"`

	// MaxRecvMsgSize limits the maximum message size the server can receive (in bytes).
	// Protects against memory exhaustion from large messages.
	// Default: 4194304 (4 MB)
	MaxRecvMsgSize int `json:"max_recv_msg_size,omitempty" mapstructure:"max_recv_msg_size"`

	// MaxSendMsgSize limits the maximum message size the server can send (in bytes).
	// Default: 4194304 (4 MB)
	MaxSendMsgSize int `json:"max_send_msg_size,omitempty" mapstructure:"max_send_msg_size"`

	// ConnectionTimeout limits the time for connection establishment.
	// Prevents hanging connection attempts from slow clients.
	// Default: 120s (2 minutes)
	ConnectionTimeout time.Duration `json:"connection_timeout,omitempty" mapstructure:"connection_timeout"`

	// Keepalive configuration for connection health management.
	Keepalive KeepaliveConfig `json:"keepalive" mapstructure:"keepalive"`
}

// KeepaliveConfig defines keepalive parameters for connection health.
// Keepalive pings detect dead connections (client crash, network partition)
// and automatically close idle or aged connections to free resources.
type KeepaliveConfig struct {
	// MaxConnectionIdle is the duration after which idle connections are closed.
	// An idle connection has no active RPC streams.
	// Default: 15m (15 minutes)
	MaxConnectionIdle time.Duration `json:"max_connection_idle,omitempty" mapstructure:"max_connection_idle"`

	// MaxConnectionAge is the maximum duration a connection may exist.
	// Forces connection rotation to prevent resource leaks and ensure TLS session rotation.
	// Default: 30m (30 minutes)
	MaxConnectionAge time.Duration `json:"max_connection_age,omitempty" mapstructure:"max_connection_age"`

	// MaxConnectionAgeGrace is the grace period after MaxConnectionAge
	// to allow inflight RPCs to complete before force-closing the connection.
	// Default: 5m (5 minutes)
	MaxConnectionAgeGrace time.Duration `json:"max_connection_age_grace,omitempty" mapstructure:"max_connection_age_grace"`

	// Time is the duration after which a keepalive ping is sent
	// on idle connections to check if the connection is still alive.
	// Default: 5m (5 minutes)
	Time time.Duration `json:"time,omitempty" mapstructure:"time"`

	// Timeout is the duration the server waits for a keepalive ping response.
	// If no response is received, the connection is closed.
	// Default: 1m (1 minute)
	Timeout time.Duration `json:"timeout,omitempty" mapstructure:"timeout"`

	// MinTime is the minimum duration clients should wait between keepalive pings.
	// Prevents clients from abusing keepalive by sending excessive pings.
	// Default: 1m (1 minute)
	MinTime time.Duration `json:"min_time,omitempty" mapstructure:"min_time"`

	// PermitWithoutStream allows clients to send keepalive pings
	// even when there are no active RPC streams.
	// Enables clients to detect dead connections proactively.
	// Default: true
	PermitWithoutStream bool `json:"permit_without_stream,omitempty" mapstructure:"permit_without_stream"`
}

// MetricsConfig holds Prometheus metrics configuration.
type MetricsConfig struct {
	// Enabled enables Prometheus metrics collection.
	// Default: true
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`

	// Address is the HTTP listen address for the metrics endpoint.
	// The metrics server runs on a separate port from the gRPC server.
	// Default: ":9090"
	Address string `json:"address,omitempty" mapstructure:"address"`
}

// DefaultConnectionConfig returns connection configuration with production-safe defaults.
// These defaults are based on industry best practices for production gRPC deployments
// and provide a balance between resource efficiency, connection stability, and security.
func DefaultConnectionConfig() ConnectionConfig {
	return ConnectionConfig{
		MaxConcurrentStreams: DefaultMaxConcurrentStreams,
		MaxRecvMsgSize:       DefaultMaxRecvMsgSize,
		MaxSendMsgSize:       DefaultMaxSendMsgSize,
		ConnectionTimeout:    DefaultConnectionTimeout,
		Keepalive: KeepaliveConfig{
			MaxConnectionIdle:     DefaultMaxConnectionIdle,
			MaxConnectionAge:      DefaultMaxConnectionAge,
			MaxConnectionAgeGrace: DefaultMaxConnectionAgeGrace,
			Time:                  DefaultKeepaliveTime,
			Timeout:               DefaultKeepaliveTimeout,
			MinTime:               DefaultMinTime,
			PermitWithoutStream:   DefaultPermitWithoutStream,
		},
	}
}

// WithDefaults returns the connection configuration with defaults applied
// if the configuration is not set or has zero values.
// This method checks if MaxConcurrentStreams is 0 (indicating unset configuration)
// and returns the default configuration in that case.
func (c ConnectionConfig) WithDefaults() ConnectionConfig {
	// If MaxConcurrentStreams is 0, the config was not set - use defaults
	if c.MaxConcurrentStreams == 0 {
		return DefaultConnectionConfig()
	}

	return c
}

//nolint:maintidx
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
			logger.Info("Config file not found, use defaults.")
		} else {
			return nil, fmt.Errorf("failed to read configuration file: %w", err)
		}
	}

	//
	// API configuration
	//
	_ = v.BindEnv("listen_address")
	v.SetDefault("listen_address", DefaultListenAddress)

	//
	// OASF Validation configuration
	//
	_ = v.BindEnv("oasf_api_validation.schema_url")
	// Note: No default set here - default should come from Helm chart values.yaml
	// Schema URL is required for OASF API validation

	//
	// Logging configuration (gRPC request/response logging)
	//
	_ = v.BindEnv("logging.verbose")
	v.SetDefault("logging.verbose", false)

	//
	// Rate limiting configuration
	//
	_ = v.BindEnv("ratelimit.enabled")
	v.SetDefault("ratelimit.enabled", false)

	_ = v.BindEnv("ratelimit.global_rps")
	v.SetDefault("ratelimit.global_rps", 0.0)

	_ = v.BindEnv("ratelimit.global_burst")
	v.SetDefault("ratelimit.global_burst", 0)

	_ = v.BindEnv("ratelimit.per_client_rps")
	v.SetDefault("ratelimit.per_client_rps", 0.0)

	_ = v.BindEnv("ratelimit.per_client_burst")
	v.SetDefault("ratelimit.per_client_burst", 0)

	// Note: method_limits (per-method rate limit overrides) can only be configured
	// via YAML/JSON config file due to its complex nested map structure.
	// Environment variable configuration for method limits is not supported.
	// Example config:
	//   ratelimit:
	//     method_limits:
	//       "/agntcy.dir.store.v1.StoreService/CreateRecord":
	//         rps: 50
	//         burst: 100

	//
	// Authn configuration (authentication: JWT or X.509)
	//
	_ = v.BindEnv("authn.enabled")
	v.SetDefault("authn.enabled", "false")

	_ = v.BindEnv("authn.mode")
	v.SetDefault("authn.mode", "x509")

	_ = v.BindEnv("authn.socket_path")
	v.SetDefault("authn.socket_path", "")

	_ = v.BindEnv("authn.audiences")
	v.SetDefault("authn.audiences", "")

	//
	// Authz configuration (authorization policies)
	//
	_ = v.BindEnv("authz.enabled")
	v.SetDefault("authz.enabled", "false")

	_ = v.BindEnv("authz.enforcer_policy_file_path")
	v.SetDefault("authz.enforcer_policy_file_path", DefaultConfigPath+"/authz_policies.csv")

	//
	// Store configuration
	//
	_ = v.BindEnv("store.provider")
	v.SetDefault("store.provider", store.DefaultProvider)

	_ = v.BindEnv("store.oci.local_dir")
	v.SetDefault("store.oci.local_dir", "")

	_ = v.BindEnv("store.oci.cache_dir")
	v.SetDefault("store.oci.cache_dir", "")

	_ = v.BindEnv("store.oci.registry_address")
	v.SetDefault("store.oci.registry_address", oci.DefaultRegistryAddress)

	_ = v.BindEnv("store.oci.repository_name")
	v.SetDefault("store.oci.repository_name", oci.DefaultRepositoryName)

	_ = v.BindEnv("store.oci.auth_config.insecure")
	v.SetDefault("store.oci.auth_config.insecure", oci.DefaultAuthConfigInsecure)

	_ = v.BindEnv("store.oci.auth_config.username")
	_ = v.BindEnv("store.oci.auth_config.password")
	_ = v.BindEnv("store.oci.auth_config.access_token")
	_ = v.BindEnv("store.oci.auth_config.refresh_token")

	//
	// Store verification configuration
	//
	_ = v.BindEnv("store.verification.enabled")
	v.SetDefault("store.verification.enabled", store.DefaultVerificationEnabled)

	//
	// Routing configuration
	//
	_ = v.BindEnv("routing.listen_address")
	v.SetDefault("routing.listen_address", routing.DefaultListenAddress)

	_ = v.BindEnv("routing.directory_api_address")
	v.SetDefault("routing.directory_api_address", "")

	_ = v.BindEnv("routing.bootstrap_peers")
	v.SetDefault("routing.bootstrap_peers", strings.Join(routing.DefaultBootstrapPeers, ","))

	_ = v.BindEnv("routing.key_path")
	v.SetDefault("routing.key_path", "")

	_ = v.BindEnv("routing.datastore_dir")
	v.SetDefault("routing.datastore_dir", "")

	//
	// Routing GossipSub configuration
	// Note: Only enable/disable is configurable. Protocol parameters (topic, message size)
	// are hardcoded in server/routing/pubsub/constants.go for network compatibility.
	//
	_ = v.BindEnv("routing.gossipsub.enabled")
	v.SetDefault("routing.gossipsub.enabled", routing.DefaultGossipSubEnabled)

	//
	// Database configuration
	//
	_ = v.BindEnv("database.type")
	v.SetDefault("database.type", dbconfig.DefaultType)

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
	// Sync configuration
	//
	_ = v.BindEnv("sync.auth_config.username")
	_ = v.BindEnv("sync.auth_config.password")

	//
	// Publication configuration
	//

	_ = v.BindEnv("publication.scheduler_interval")
	v.SetDefault("publication.scheduler_interval", publication.DefaultPublicationSchedulerInterval)

	_ = v.BindEnv("publication.worker_count")
	v.SetDefault("publication.worker_count", publication.DefaultPublicationWorkerCount)

	_ = v.BindEnv("publication.worker_timeout")
	v.SetDefault("publication.worker_timeout", publication.DefaultPublicationWorkerTimeout)

	//
	// Events configuration
	//

	_ = v.BindEnv("events.subscriber_buffer_size")
	v.SetDefault("events.subscriber_buffer_size", events.DefaultSubscriberBufferSize)

	_ = v.BindEnv("events.log_slow_consumers")
	v.SetDefault("events.log_slow_consumers", events.DefaultLogSlowConsumers)

	_ = v.BindEnv("events.log_published_events")
	v.SetDefault("events.log_published_events", events.DefaultLogPublishedEvents)

	//
	// Metrics configuration
	//
	_ = v.BindEnv("metrics.enabled")
	v.SetDefault("metrics.enabled", DefaultMetricsEnabled)

	_ = v.BindEnv("metrics.address")
	v.SetDefault("metrics.address", DefaultMetricsAddress)

	//
	// Naming (name verification cache TTL for API responses; re-verification is done by the reconciler name task)
	//
	_ = v.BindEnv("naming.ttl")
	v.SetDefault("naming.ttl", naming.DefaultTTL)

	//
	// Connection management configuration
	//
	// Design Decision: No environment variables for connection management.
	// Rationale:
	//   - 11 env vars would be too many and too technical for most users
	//   - Production-safe defaults work for 90% of deployments
	//   - Advanced users can use YAML config file for fine-grained control
	//   - Follows industry best practices (Kubernetes, Prometheus, etc.)
	//
	// For advanced configuration, use YAML config file:
	//   connection:
	//     max_concurrent_streams: 2000
	//     max_recv_msg_size: 8388608  # 8 MB
	//     keepalive:
	//       max_connection_idle: 10m
	//       # ... other settings
	//
	// No viper defaults needed - defaults are applied via ConnectionConfig.WithDefaults()
	// after loading to ensure clean separation between loading and defaulting logic.

	// Load configuration into struct
	decodeHooks := mapstructure.ComposeDecodeHookFunc(
		mapstructure.TextUnmarshallerHookFunc(),
		mapstructure.StringToTimeDurationHookFunc(),
		mapstructure.StringToSliceHookFunc(","),
	)

	config := &Config{}
	if err := v.Unmarshal(config, viper.DecodeHook(decodeHooks)); err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Apply connection management defaults if not configured
	// This happens after unmarshal so YAML config takes precedence over defaults
	config.Connection = config.Connection.WithDefaults()

	return config, nil
}
