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
