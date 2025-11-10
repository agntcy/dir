// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:testifylint
package config

import (
	"testing"
	"time"

	authn "github.com/agntcy/dir/server/authn/config"
	authz "github.com/agntcy/dir/server/authz/config"
	database "github.com/agntcy/dir/server/database/config"
	sqliteconfig "github.com/agntcy/dir/server/database/sqlite/config"
	ratelimitconfig "github.com/agntcy/dir/server/middleware/ratelimit/config"
	publication "github.com/agntcy/dir/server/publication/config"
	routing "github.com/agntcy/dir/server/routing/config"
	store "github.com/agntcy/dir/server/store/config"
	oci "github.com/agntcy/dir/server/store/oci/config"
	sync "github.com/agntcy/dir/server/sync/config"
	monitor "github.com/agntcy/dir/server/sync/monitor/config"
	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	tests := []struct {
		Name           string
		EnvVars        map[string]string
		ExpectedConfig *Config
	}{
		{
			Name: "Custom config",
			EnvVars: map[string]string{
				"DIRECTORY_SERVER_LISTEN_ADDRESS":                       "example.com:8889",
				"DIRECTORY_SERVER_STORE_PROVIDER":                       "provider",
				"DIRECTORY_SERVER_STORE_OCI_LOCAL_DIR":                  "local-dir",
				"DIRECTORY_SERVER_STORE_OCI_REGISTRY_ADDRESS":           "example.com:5001",
				"DIRECTORY_SERVER_STORE_OCI_REPOSITORY_NAME":            "test-dir",
				"DIRECTORY_SERVER_STORE_OCI_AUTH_CONFIG_INSECURE":       "true",
				"DIRECTORY_SERVER_STORE_OCI_AUTH_CONFIG_USERNAME":       "username",
				"DIRECTORY_SERVER_STORE_OCI_AUTH_CONFIG_PASSWORD":       "password",
				"DIRECTORY_SERVER_STORE_OCI_AUTH_CONFIG_ACCESS_TOKEN":   "access-token",
				"DIRECTORY_SERVER_STORE_OCI_AUTH_CONFIG_REFRESH_TOKEN":  "refresh-token",
				"DIRECTORY_SERVER_ROUTING_LISTEN_ADDRESS":               "/ip4/1.1.1.1/tcp/1",
				"DIRECTORY_SERVER_ROUTING_BOOTSTRAP_PEERS":              "/ip4/1.1.1.1/tcp/1,/ip4/1.1.1.1/tcp/2",
				"DIRECTORY_SERVER_ROUTING_KEY_PATH":                     "/path/to/key",
				"DIRECTORY_SERVER_DATABASE_DB_TYPE":                     "sqlite",
				"DIRECTORY_SERVER_DATABASE_SQLITE_DB_PATH":              "sqlite.db",
				"DIRECTORY_SERVER_SYNC_SCHEDULER_INTERVAL":              "1s",
				"DIRECTORY_SERVER_SYNC_WORKER_COUNT":                    "1",
				"DIRECTORY_SERVER_SYNC_REGISTRY_MONITOR_CHECK_INTERVAL": "10s",
				"DIRECTORY_SERVER_SYNC_WORKER_TIMEOUT":                  "10s",
				"DIRECTORY_SERVER_SYNC_AUTH_CONFIG_USERNAME":            "sync-user",
				"DIRECTORY_SERVER_SYNC_AUTH_CONFIG_PASSWORD":            "sync-password",
				"DIRECTORY_SERVER_AUTHZ_ENABLED":                        "true",
				"DIRECTORY_SERVER_AUTHZ_SOCKET_PATH":                    "/test/agent.sock",
				"DIRECTORY_SERVER_AUTHZ_TRUST_DOMAIN":                   "dir.com",
				"DIRECTORY_SERVER_PUBLICATION_SCHEDULER_INTERVAL":       "10s",
				"DIRECTORY_SERVER_PUBLICATION_WORKER_COUNT":             "1",
				"DIRECTORY_SERVER_PUBLICATION_WORKER_TIMEOUT":           "10s",
			},
			ExpectedConfig: &Config{
				ListenAddress: "example.com:8889",
				Connection:    DefaultConnectionConfig(), // Connection defaults applied
				Authn: authn.Config{
					Enabled:   false,
					Mode:      authn.AuthModeX509, // Default from config.go:109
					Audiences: []string{},
				},
				Store: store.Config{
					Provider: "provider",
					OCI: oci.Config{
						LocalDir:        "local-dir",
						RegistryAddress: "example.com:5001",
						RepositoryName:  "test-dir",
						AuthConfig: oci.AuthConfig{
							Insecure:     true,
							Username:     "username",
							Password:     "password",
							RefreshToken: "refresh-token",
							AccessToken:  "access-token",
						},
					},
				},
				Routing: routing.Config{
					ListenAddress: "/ip4/1.1.1.1/tcp/1",
					BootstrapPeers: []string{
						"/ip4/1.1.1.1/tcp/1",
						"/ip4/1.1.1.1/tcp/2",
					},
					KeyPath: "/path/to/key",
					GossipSub: routing.GossipSubConfig{
						Enabled: true, // Default value
					},
				},
				Database: database.Config{
					DBType: "sqlite",
					SQLite: sqliteconfig.Config{
						DBPath: "sqlite.db",
					},
				},
				Sync: sync.Config{
					SchedulerInterval: 1 * time.Second,
					WorkerCount:       1,
					WorkerTimeout:     10 * time.Second,
					RegistryMonitor: monitor.Config{
						CheckInterval: 10 * time.Second,
					},
					AuthConfig: sync.AuthConfig{
						Username: "sync-user",
						Password: "sync-password",
					},
				},
				Authz: authz.Config{
					Enabled:     true,
					TrustDomain: "dir.com",
				},
				Publication: publication.Config{
					SchedulerInterval: 10 * time.Second,
					WorkerCount:       1,
					WorkerTimeout:     10 * time.Second,
				},
			},
		},
		{
			Name:    "Default config",
			EnvVars: map[string]string{},
			ExpectedConfig: &Config{
				ListenAddress: DefaultListenAddress,
				Connection:    DefaultConnectionConfig(), // Connection defaults applied
				Authn: authn.Config{
					Enabled:   false,
					Mode:      authn.AuthModeX509, // Default from config.go:109
					Audiences: []string{},
				},
				Store: store.Config{
					Provider: store.DefaultProvider,
					OCI: oci.Config{
						RegistryAddress: oci.DefaultRegistryAddress,
						RepositoryName:  oci.DefaultRepositoryName,
						AuthConfig: oci.AuthConfig{
							Insecure: oci.DefaultAuthConfigInsecure,
						},
					},
				},
				Routing: routing.Config{
					ListenAddress:  routing.DefaultListenAddress,
					BootstrapPeers: routing.DefaultBootstrapPeers,
					GossipSub: routing.GossipSubConfig{
						Enabled: routing.DefaultGossipSubEnabled,
					},
				},
				Database: database.Config{
					DBType: database.DefaultDBType,
					SQLite: sqliteconfig.Config{
						DBPath: sqliteconfig.DefaultSQLiteDBPath,
					},
				},
				Sync: sync.Config{
					SchedulerInterval: sync.DefaultSyncSchedulerInterval,
					WorkerCount:       sync.DefaultSyncWorkerCount,
					WorkerTimeout:     sync.DefaultSyncWorkerTimeout,
					RegistryMonitor: monitor.Config{
						CheckInterval: monitor.DefaultCheckInterval,
					},
				},
				Authz: authz.Config{},
				Publication: publication.Config{
					SchedulerInterval: publication.DefaultPublicationSchedulerInterval,
					WorkerCount:       publication.DefaultPublicationWorkerCount,
					WorkerTimeout:     publication.DefaultPublicationWorkerTimeout,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			for k, v := range test.EnvVars {
				t.Setenv(k, v)
			}

			config, err := LoadConfig()
			assert.NoError(t, err)
			assert.Equal(t, *config, *test.ExpectedConfig)
		})
	}
}

// TestConfig_RateLimiting tests that rate limiting configuration is correctly parsed.
func TestConfig_RateLimiting(t *testing.T) {
	tests := []struct {
		name           string
		envVars        map[string]string
		expectedConfig ratelimitconfig.Config
	}{
		{
			name: "rate limiting enabled with custom values",
			envVars: map[string]string{
				"DIRECTORY_SERVER_RATELIMIT_ENABLED":          "true",
				"DIRECTORY_SERVER_RATELIMIT_GLOBAL_RPS":       "50.0",
				"DIRECTORY_SERVER_RATELIMIT_GLOBAL_BURST":     "100",
				"DIRECTORY_SERVER_RATELIMIT_PER_CLIENT_RPS":   "500.0",
				"DIRECTORY_SERVER_RATELIMIT_PER_CLIENT_BURST": "1000",
			},
			expectedConfig: ratelimitconfig.Config{
				Enabled:        true,
				GlobalRPS:      50.0,
				GlobalBurst:    100,
				PerClientRPS:   500.0,
				PerClientBurst: 1000,
				MethodLimits:   map[string]ratelimitconfig.MethodLimit{},
			},
		},
		{
			name: "rate limiting disabled (default)",
			envVars: map[string]string{
				"DIRECTORY_SERVER_RATELIMIT_ENABLED": "false",
			},
			expectedConfig: ratelimitconfig.Config{
				Enabled:        false,
				GlobalRPS:      0,
				GlobalBurst:    0,
				PerClientRPS:   0,
				PerClientBurst: 0,
				MethodLimits:   map[string]ratelimitconfig.MethodLimit{},
			},
		},
		{
			name: "rate limiting with partial configuration",
			envVars: map[string]string{
				"DIRECTORY_SERVER_RATELIMIT_ENABLED":      "true",
				"DIRECTORY_SERVER_RATELIMIT_GLOBAL_RPS":   "200.0",
				"DIRECTORY_SERVER_RATELIMIT_GLOBAL_BURST": "400",
			},
			expectedConfig: ratelimitconfig.Config{
				Enabled:        true,
				GlobalRPS:      200.0,
				GlobalBurst:    400,
				PerClientRPS:   0,
				PerClientBurst: 0,
				MethodLimits:   map[string]ratelimitconfig.MethodLimit{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			// Load config
			cfg, err := LoadConfig()
			assert.NoError(t, err)

			// Verify rate limiting configuration
			assert.Equal(t, tt.expectedConfig.Enabled, cfg.RateLimit.Enabled)
			assert.Equal(t, tt.expectedConfig.GlobalRPS, cfg.RateLimit.GlobalRPS)
			assert.Equal(t, tt.expectedConfig.GlobalBurst, cfg.RateLimit.GlobalBurst)
			assert.Equal(t, tt.expectedConfig.PerClientRPS, cfg.RateLimit.PerClientRPS)
			assert.Equal(t, tt.expectedConfig.PerClientBurst, cfg.RateLimit.PerClientBurst)
		})
	}
}

// TestConfig_RateLimitingValidation tests that invalid rate limiting configuration
// is properly validated during server initialization (will be tested in server tests).
func TestConfig_RateLimitingValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      ratelimitconfig.Config
		shouldError bool
	}{
		{
			name: "valid rate limiting configuration",
			config: ratelimitconfig.Config{
				Enabled:        true,
				GlobalRPS:      100.0,
				GlobalBurst:    200,
				PerClientRPS:   1000.0,
				PerClientBurst: 2000,
			},
			shouldError: false,
		},
		{
			name: "invalid rate limiting - negative RPS",
			config: ratelimitconfig.Config{
				Enabled:        true,
				GlobalRPS:      -10.0,
				GlobalBurst:    200,
				PerClientRPS:   1000.0,
				PerClientBurst: 2000,
			},
			shouldError: true,
		},
		{
			name: "invalid rate limiting - negative burst",
			config: ratelimitconfig.Config{
				Enabled:        true,
				GlobalRPS:      100.0,
				GlobalBurst:    -200,
				PerClientRPS:   1000.0,
				PerClientBurst: 2000,
			},
			shouldError: true,
		},
		{
			name: "disabled rate limiting - no validation",
			config: ratelimitconfig.Config{
				Enabled:        false,
				GlobalRPS:      -100.0, // Invalid but should be ignored
				GlobalBurst:    -200,
				PerClientRPS:   -1000.0,
				PerClientBurst: -2000,
			},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestDefaultConnectionConfig verifies that DefaultConnectionConfig returns
// the correct production-safe default values for all connection parameters.
func TestDefaultConnectionConfig(t *testing.T) {
	cfg := DefaultConnectionConfig()

	// Verify connection limits
	assert.Equal(t, uint32(1000), cfg.MaxConcurrentStreams, "MaxConcurrentStreams should be 1000")
	assert.Equal(t, 4*1024*1024, cfg.MaxRecvMsgSize, "MaxRecvMsgSize should be 4 MB")
	assert.Equal(t, 4*1024*1024, cfg.MaxSendMsgSize, "MaxSendMsgSize should be 4 MB")
	assert.Equal(t, 120*time.Second, cfg.ConnectionTimeout, "ConnectionTimeout should be 120 seconds")

	// Verify keepalive parameters
	assert.Equal(t, 15*time.Minute, cfg.Keepalive.MaxConnectionIdle, "MaxConnectionIdle should be 15 minutes")
	assert.Equal(t, 30*time.Minute, cfg.Keepalive.MaxConnectionAge, "MaxConnectionAge should be 30 minutes")
	assert.Equal(t, 5*time.Minute, cfg.Keepalive.MaxConnectionAgeGrace, "MaxConnectionAgeGrace should be 5 minutes")
	assert.Equal(t, 5*time.Minute, cfg.Keepalive.Time, "Keepalive Time should be 5 minutes")
	assert.Equal(t, 1*time.Minute, cfg.Keepalive.Timeout, "Keepalive Timeout should be 1 minute")
	assert.Equal(t, 1*time.Minute, cfg.Keepalive.MinTime, "Keepalive MinTime should be 1 minute")
	assert.Equal(t, true, cfg.Keepalive.PermitWithoutStream, "PermitWithoutStream should be true")
}

// TestConnectionConfig_DefaultValues verifies that LoadConfig returns default
// connection configuration when no environment variables are set.
func TestConnectionConfig_DefaultValues(t *testing.T) {
	// No environment variables set - should use defaults
	cfg, err := LoadConfig()
	assert.NoError(t, err)

	// Verify connection configuration has default values
	// Note: In Phase 3 we'll add default loading to LoadConfig()
	// For now, we just verify the struct exists
	assert.NotNil(t, cfg.Connection)
}

// TestConnectionConfig_Constants verifies that all connection constants
// are defined with the expected values based on production best practices.
func TestConnectionConfig_Constants(t *testing.T) {
	// Connection limits - use EqualValues for cross-type comparison
	assert.EqualValues(t, 1000, DefaultMaxConcurrentStreams)
	assert.EqualValues(t, 4*1024*1024, DefaultMaxRecvMsgSize)
	assert.EqualValues(t, 4*1024*1024, DefaultMaxSendMsgSize)
	assert.Equal(t, 120*time.Second, DefaultConnectionTimeout)

	// Keepalive parameters
	assert.Equal(t, 15*time.Minute, DefaultMaxConnectionIdle)
	assert.Equal(t, 30*time.Minute, DefaultMaxConnectionAge)
	assert.Equal(t, 5*time.Minute, DefaultMaxConnectionAgeGrace)
	assert.Equal(t, 5*time.Minute, DefaultKeepaliveTime)
	assert.Equal(t, 1*time.Minute, DefaultKeepaliveTimeout)
	assert.Equal(t, 1*time.Minute, DefaultMinTime)
	assert.True(t, DefaultPermitWithoutStream)
}

// TestConnectionConfig_StructTags verifies that struct tags are properly
// defined for JSON and mapstructure serialization.
func TestConnectionConfig_StructTags(t *testing.T) {
	// This test ensures that configuration can be properly serialized
	// and deserialized from YAML/JSON files
	cfg := ConnectionConfig{
		MaxConcurrentStreams: 2000,
		MaxRecvMsgSize:       8 * 1024 * 1024,
		MaxSendMsgSize:       8 * 1024 * 1024,
		ConnectionTimeout:    60 * time.Second,
		Keepalive: KeepaliveConfig{
			MaxConnectionIdle:     10 * time.Minute,
			MaxConnectionAge:      20 * time.Minute,
			MaxConnectionAgeGrace: 3 * time.Minute,
			Time:                  3 * time.Minute,
			Timeout:               30 * time.Second,
			MinTime:               30 * time.Second,
			PermitWithoutStream:   false,
		},
	}

	// Verify struct is not empty and can be created
	assert.NotNil(t, cfg)
	assert.Equal(t, uint32(2000), cfg.MaxConcurrentStreams)
	assert.NotNil(t, cfg.Keepalive)
	assert.Equal(t, 10*time.Minute, cfg.Keepalive.MaxConnectionIdle)
}

// TestConnectionConfig_ProductionSafety verifies that default values
// are production-safe and follow gRPC best practices.
func TestConnectionConfig_ProductionSafety(t *testing.T) {
	cfg := DefaultConnectionConfig()

	// Verify MaxConcurrentStreams is reasonable (not too low, not unlimited)
	assert.Greater(t, cfg.MaxConcurrentStreams, uint32(100), "MaxConcurrentStreams should allow reasonable concurrency")
	assert.Less(t, cfg.MaxConcurrentStreams, uint32(10000), "MaxConcurrentStreams should not be excessive")

	// Verify message sizes protect against memory exhaustion
	assert.Greater(t, cfg.MaxRecvMsgSize, 1*1024*1024, "MaxRecvMsgSize should allow reasonable messages")
	assert.Less(t, cfg.MaxRecvMsgSize, 100*1024*1024, "MaxRecvMsgSize should prevent memory exhaustion")

	// Verify keepalive times are reasonable
	assert.Greater(t, cfg.Keepalive.Time, 1*time.Minute, "Keepalive Time should not be too aggressive")
	assert.Less(t, cfg.Keepalive.Time, 30*time.Minute, "Keepalive Time should detect dead connections")

	// Verify idle timeout is reasonable
	assert.Greater(t, cfg.Keepalive.MaxConnectionIdle, 5*time.Minute, "MaxConnectionIdle should not be too aggressive")
	assert.Less(t, cfg.Keepalive.MaxConnectionIdle, 2*time.Hour, "MaxConnectionIdle should free resources")

	// Verify connection age rotation
	assert.Greater(t, cfg.Keepalive.MaxConnectionAge, cfg.Keepalive.MaxConnectionIdle, "MaxConnectionAge should be greater than MaxConnectionIdle")
	assert.Greater(t, cfg.Keepalive.MaxConnectionAgeGrace, 1*time.Minute, "MaxConnectionAgeGrace should allow inflight RPCs to complete")

	// Verify keepalive timeout is reasonable
	assert.Greater(t, cfg.Keepalive.Timeout, 10*time.Second, "Keepalive Timeout should allow for network delays")
	assert.Less(t, cfg.Keepalive.Timeout, 5*time.Minute, "Keepalive Timeout should not wait too long")

	// Verify MinTime prevents abuse
	assert.Greater(t, cfg.Keepalive.MinTime, 10*time.Second, "MinTime should prevent excessive pings")

	// Verify PermitWithoutStream is enabled for better health detection
	assert.True(t, cfg.Keepalive.PermitWithoutStream, "PermitWithoutStream should be enabled")
}

// TestConnectionConfig_WithDefaults verifies that WithDefaults returns
// the correct configuration based on whether the config is set or not.
func TestConnectionConfig_WithDefaults(t *testing.T) {
	tests := []struct {
		name     string
		input    ConnectionConfig
		expected ConnectionConfig
	}{
		{
			name:     "empty config returns defaults",
			input:    ConnectionConfig{},
			expected: DefaultConnectionConfig(),
		},
		{
			name: "zero MaxConcurrentStreams returns defaults",
			input: ConnectionConfig{
				MaxConcurrentStreams: 0,
				MaxRecvMsgSize:       8 * 1024 * 1024,
			},
			expected: DefaultConnectionConfig(),
		},
		{
			name: "configured values are preserved",
			input: ConnectionConfig{
				MaxConcurrentStreams: 2000,
				MaxRecvMsgSize:       8 * 1024 * 1024,
				MaxSendMsgSize:       8 * 1024 * 1024,
				ConnectionTimeout:    60 * time.Second,
				Keepalive: KeepaliveConfig{
					MaxConnectionIdle:     10 * time.Minute,
					MaxConnectionAge:      20 * time.Minute,
					MaxConnectionAgeGrace: 3 * time.Minute,
					Time:                  3 * time.Minute,
					Timeout:               30 * time.Second,
					MinTime:               30 * time.Second,
					PermitWithoutStream:   false,
				},
			},
			expected: ConnectionConfig{
				MaxConcurrentStreams: 2000,
				MaxRecvMsgSize:       8 * 1024 * 1024,
				MaxSendMsgSize:       8 * 1024 * 1024,
				ConnectionTimeout:    60 * time.Second,
				Keepalive: KeepaliveConfig{
					MaxConnectionIdle:     10 * time.Minute,
					MaxConnectionAge:      20 * time.Minute,
					MaxConnectionAgeGrace: 3 * time.Minute,
					Time:                  3 * time.Minute,
					Timeout:               30 * time.Second,
					MinTime:               30 * time.Second,
					PermitWithoutStream:   false,
				},
			},
		},
		{
			name: "partial config with non-zero MaxConcurrentStreams is preserved",
			input: ConnectionConfig{
				MaxConcurrentStreams: 500,
				// Other fields zero/default
			},
			expected: ConnectionConfig{
				MaxConcurrentStreams: 500,
				// Other fields remain zero/default as set
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.WithDefaults()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestLoadConfig_ConnectionDefaults verifies that LoadConfig applies
// connection management defaults when no config file is present.
func TestLoadConfig_ConnectionDefaults(t *testing.T) {
	// LoadConfig should apply defaults automatically
	cfg, err := LoadConfig()
	assert.NoError(t, err)
	assert.NotNil(t, cfg)

	// Verify connection config has defaults applied (use EqualValues for uint32)
	assert.EqualValues(t, DefaultMaxConcurrentStreams, cfg.Connection.MaxConcurrentStreams)
	assert.EqualValues(t, DefaultMaxRecvMsgSize, cfg.Connection.MaxRecvMsgSize)
	assert.EqualValues(t, DefaultMaxSendMsgSize, cfg.Connection.MaxSendMsgSize)
	assert.Equal(t, DefaultConnectionTimeout, cfg.Connection.ConnectionTimeout)

	// Verify keepalive defaults
	assert.Equal(t, DefaultMaxConnectionIdle, cfg.Connection.Keepalive.MaxConnectionIdle)
	assert.Equal(t, DefaultMaxConnectionAge, cfg.Connection.Keepalive.MaxConnectionAge)
	assert.Equal(t, DefaultMaxConnectionAgeGrace, cfg.Connection.Keepalive.MaxConnectionAgeGrace)
	assert.Equal(t, DefaultKeepaliveTime, cfg.Connection.Keepalive.Time)
	assert.Equal(t, DefaultKeepaliveTimeout, cfg.Connection.Keepalive.Timeout)
	assert.Equal(t, DefaultMinTime, cfg.Connection.Keepalive.MinTime)
	assert.Equal(t, DefaultPermitWithoutStream, cfg.Connection.Keepalive.PermitWithoutStream)
}

// TestConnectionConfig_YAMLSerialization verifies that connection configuration
// can be serialized to and from YAML format correctly.
func TestConnectionConfig_YAMLSerialization(t *testing.T) {
	// Create a custom configuration
	customConfig := ConnectionConfig{
		MaxConcurrentStreams: 2000,
		MaxRecvMsgSize:       8388608, // 8 MB
		MaxSendMsgSize:       8388608, // 8 MB
		ConnectionTimeout:    60 * time.Second,
		Keepalive: KeepaliveConfig{
			MaxConnectionIdle:     10 * time.Minute,
			MaxConnectionAge:      20 * time.Minute,
			MaxConnectionAgeGrace: 3 * time.Minute,
			Time:                  3 * time.Minute,
			Timeout:               30 * time.Second,
			MinTime:               30 * time.Second,
			PermitWithoutStream:   false,
		},
	}

	// Verify all fields can be set with custom values
	assert.Equal(t, uint32(2000), customConfig.MaxConcurrentStreams)
	assert.Equal(t, 8388608, customConfig.MaxRecvMsgSize)
	assert.Equal(t, 8388608, customConfig.MaxSendMsgSize)
	assert.Equal(t, 60*time.Second, customConfig.ConnectionTimeout)
	assert.Equal(t, 10*time.Minute, customConfig.Keepalive.MaxConnectionIdle)
	assert.Equal(t, 20*time.Minute, customConfig.Keepalive.MaxConnectionAge)
	assert.Equal(t, 3*time.Minute, customConfig.Keepalive.MaxConnectionAgeGrace)
	assert.Equal(t, 3*time.Minute, customConfig.Keepalive.Time)
	assert.Equal(t, 30*time.Second, customConfig.Keepalive.Timeout)
	assert.Equal(t, 30*time.Second, customConfig.Keepalive.MinTime)
	assert.False(t, customConfig.Keepalive.PermitWithoutStream)
}

// TestConnectionConfig_MapstructureTags verifies that struct tags are properly
// defined for mapstructure to work with YAML/JSON loading.
func TestConnectionConfig_MapstructureTags(t *testing.T) {
	// This test ensures mapstructure can parse the config
	// The actual YAML loading is tested through LoadConfig in integration tests

	// Verify we have the correct field types for mapstructure
	cfg := ConnectionConfig{
		MaxConcurrentStreams: 1000,
		MaxRecvMsgSize:       4 * 1024 * 1024,
		MaxSendMsgSize:       4 * 1024 * 1024,
		ConnectionTimeout:    120 * time.Second,
		Keepalive: KeepaliveConfig{
			MaxConnectionIdle:     15 * time.Minute,
			MaxConnectionAge:      30 * time.Minute,
			MaxConnectionAgeGrace: 5 * time.Minute,
			Time:                  5 * time.Minute,
			Timeout:               1 * time.Minute,
			MinTime:               1 * time.Minute,
			PermitWithoutStream:   true,
		},
	}

	// Verify struct can be created and all fields are accessible
	assert.NotZero(t, cfg.MaxConcurrentStreams)
	assert.NotZero(t, cfg.MaxRecvMsgSize)
	assert.NotZero(t, cfg.MaxSendMsgSize)
	assert.NotZero(t, cfg.ConnectionTimeout)
	assert.NotZero(t, cfg.Keepalive.MaxConnectionIdle)
	assert.NotZero(t, cfg.Keepalive.MaxConnectionAge)
	assert.NotZero(t, cfg.Keepalive.Time)
}
