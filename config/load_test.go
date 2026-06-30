// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/agntcy/dir/config"
	"github.com/agntcy/dir/config/auth"
	"github.com/agntcy/dir/config/naming"
	reconcilercfg "github.com/agntcy/dir/config/reconciler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadReconcilerConfig_Defaults(t *testing.T) {
	t.Parallel()

	cfg, err := config.LoadReconcilerConfig(
		config.WithReader(strings.NewReader(""), "yaml"),
	)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, config.DefaultDatabaseType, cfg.Database.Type)
	assert.Equal(t, config.DefaultPostgresHost, cfg.Database.Postgres.Host)
	assert.Equal(t, config.DefaultPostgresPort, cfg.Database.Postgres.Port)
	assert.Equal(t, config.DefaultPostgresDatabase, cfg.Database.Postgres.Database)
	assert.Equal(t, config.DefaultPostgresSSLMode, cfg.Database.Postgres.SSLMode)

	assert.True(t, cfg.Regsync.Enabled)
	assert.True(t, cfg.Indexer.Enabled)
	assert.False(t, cfg.Name.Enabled)
	assert.True(t, cfg.Signature.Enabled)
	assert.True(t, cfg.Metrics.Enabled)
}

func TestLoadReconcilerConfig_EnvOverrides(t *testing.T) {
	t.Setenv("RECONCILER_REGSYNC_ENABLED", "false")
	t.Setenv("RECONCILER_INDEXER_ENABLED", "false")
	t.Setenv("RECONCILER_NAME_ENABLED", "true")
	t.Setenv("RECONCILER_INDEXER_INTERVAL", "2h")
	t.Setenv("RECONCILER_NAME_INTERVAL", "30m")

	cfg, err := config.LoadReconcilerConfig(
		config.WithReader(strings.NewReader(""), "yaml"),
	)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.False(t, cfg.Regsync.Enabled)
	assert.False(t, cfg.Indexer.Enabled)
	assert.True(t, cfg.Name.Enabled)
	assert.Equal(t, 2*time.Hour, cfg.Indexer.Interval)
	assert.Equal(t, 30*time.Minute, cfg.Name.Interval)
}

func TestLoadConfig_Defaults(t *testing.T) {
	t.Parallel()

	cfg, err := config.LoadConfig(
		config.WithReader(strings.NewReader(""), "yaml"),
	)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.Equal(t, config.DefaultRegistryAddress, cfg.Store.RegistryAddress)
	require.Equal(t, config.DefaultRepositoryName, cfg.Store.RepositoryName)

	require.Equal(t, config.DefaultDatabaseType, cfg.Database.Type)

	require.Equal(t, config.DefaultListenAddress, cfg.APIServer.ListenAddress)
	require.Equal(t, config.DefaultMetricsAddress, cfg.APIServer.Metrics.Address)

	require.Equal(t, naming.DefaultTTL, cfg.APIServer.Naming.GetTTL())

	require.True(t, cfg.Reconciler.Regsync.Enabled)
	require.True(t, cfg.Reconciler.Indexer.Enabled)
	require.True(t, cfg.Reconciler.Signature.Enabled)
}

func TestLoadConfig_FromYAML(t *testing.T) {
	t.Parallel()

	yaml := `
store:
  registry_address: "registry.example.com"
  repository_name: "custom/repo"
database:
  type: postgres
  postgres:
    host: db.example.com
    port: 6432
server:
  listen_address: "0.0.0.0:9999"
  authn:
    enabled: true
    mode: jwt
reconciler:
  regsync:
    interval: 5m
`

	cfg, err := config.LoadConfig(config.WithReader(strings.NewReader(yaml), "yaml"))
	require.NoError(t, err)

	require.Equal(t, "registry.example.com", cfg.Store.RegistryAddress)
	require.Equal(t, "custom/repo", cfg.Store.RepositoryName)
	require.Equal(t, "postgres", cfg.Database.Type)
	require.Equal(t, "db.example.com", cfg.Database.Postgres.Host)
	require.Equal(t, 6432, cfg.Database.Postgres.Port)
	require.Equal(t, "0.0.0.0:9999", cfg.APIServer.ListenAddress)
	require.True(t, cfg.APIServer.Authn.Enabled)
	require.Equal(t, "5m0s", cfg.Reconciler.Regsync.Interval.String())
}

func TestLoadConfig_EnvOverride(t *testing.T) {
	t.Setenv("DIRECTORY_STORE_REGISTRY_ADDRESS", "env.registry:5000")
	t.Setenv("DIRECTORY_DATABASE_TYPE", "postgres")
	t.Setenv("DIRECTORY_SERVER_LISTEN_ADDRESS", "0.0.0.0:7777")

	cfg, err := config.LoadConfig(config.WithReader(strings.NewReader(""), "yaml"))
	require.NoError(t, err)

	require.Equal(t, "env.registry:5000", cfg.Store.RegistryAddress)
	require.Equal(t, "postgres", cfg.Database.Type)
	require.Equal(t, "0.0.0.0:7777", cfg.APIServer.ListenAddress)
}

// serverLoadConfig is a helper that mimics the old server/config.LoadConfig behaviour:
// it loads with the "server.config" file name and applies connection defaults.
func serverLoadConfig(t *testing.T) (*config.Config, error) {
	t.Helper()

	cfg, err := config.LoadConfig(
		config.WithConfigName("server.config"),
		config.WithConfigPath(config.DefaultConfigPath),
	)
	if err != nil {
		return nil, fmt.Errorf("serverLoadConfig: %w", err)
	}

	cfg.APIServer.Connection = cfg.APIServer.Connection.WithDefaults()

	return cfg, nil
}

//nolint:testifylint,maintidx
func TestLoadConfig_ServerDefaults(t *testing.T) {
	tests := []struct {
		Name           string
		EnvVars        map[string]string
		ExpectedConfig *config.Config
	}{
		{
			Name: "Custom config",
			EnvVars: map[string]string{
				"DIRECTORY_SERVER_LISTEN_ADDRESS":                  "example.com:8889",
				"DIRECTORY_SERVER_ROUTING_LISTEN_ADDRESS":          "/ip4/1.1.1.1/tcp/1",
				"DIRECTORY_SERVER_ROUTING_BOOTSTRAP_PEERS":         "/ip4/1.1.1.1/tcp/1,/ip4/1.1.1.1/tcp/2",
				"DIRECTORY_SERVER_ROUTING_KEY_PATH":                "/path/to/key",
				"DIRECTORY_SERVER_SYNC_AUTH_CONFIG_USERNAME":       "sync-user",
				"DIRECTORY_SERVER_SYNC_AUTH_CONFIG_PASSWORD":       "sync-password",
				"DIRECTORY_SERVER_AUTHZ_ENABLED":                   "true",
				"DIRECTORY_SERVER_AUTHZ_ENFORCER_POLICY_FILE_PATH": "/tmp/authz_policies.csv",
				"DIRECTORY_SERVER_PUBLICATION_SCHEDULER_INTERVAL":  "10s",
				"DIRECTORY_SERVER_PUBLICATION_WORKER_COUNT":        "1",
				"DIRECTORY_SERVER_PUBLICATION_WORKER_TIMEOUT":      "10s",
				"DIRECTORY_SERVER_HTTP_GATEWAY_ENABLED":            "true",
				"DIRECTORY_SERVER_HTTP_GATEWAY_LISTEN_ADDRESS":     "address:123",
				"DIRECTORY_SERVER_HTTP_GATEWAY_PUBLIC_URL":         "https://example.com",
				"DIRECTORY_SERVER_HTTP_GATEWAY_CATALOG_TITLE":      "Cisco AI Catalog",
				"DIRECTORY_OASF_API_VALIDATION_SCHEMA_URL":         "https://custom.schema.url",
				"DIRECTORY_STORE_LOCAL_DIR":                        "local-dir",
				"DIRECTORY_STORE_REGISTRY_ADDRESS":                 "example.com:5001",
				"DIRECTORY_STORE_REPOSITORY_NAME":                  "test-dir",
				"DIRECTORY_STORE_AUTH_CONFIG_INSECURE":             "true",
				"DIRECTORY_STORE_AUTH_CONFIG_USERNAME":             "username",
				"DIRECTORY_STORE_AUTH_CONFIG_PASSWORD":             "password",
				"DIRECTORY_STORE_AUTH_CONFIG_ACCESS_TOKEN":         "access-token",
				"DIRECTORY_STORE_AUTH_CONFIG_REFRESH_TOKEN":        "refresh-token",
				"DIRECTORY_DATABASE_TYPE":                          "postgres",
				"DIRECTORY_DATABASE_POSTGRES_HOST":                 "localhost",
				"DIRECTORY_DATABASE_POSTGRES_PORT":                 "5432",
				"DIRECTORY_DATABASE_POSTGRES_DATABASE":             "dir",
				"DIRECTORY_DATABASE_POSTGRES_SSL_MODE":             "auto",
			},
			ExpectedConfig: &config.Config{
				OASFAPIValidation: config.OASFAPIValidation{
					SchemaURL: "https://custom.schema.url",
				},
				Store: config.Registry{
					LocalDir:        "local-dir",
					RegistryAddress: "example.com:5001",
					RepositoryName:  "test-dir",
					RegistryAuth: config.RegistryAuth{
						Insecure:     true,
						Username:     "username",
						Password:     "password",
						RefreshToken: "refresh-token",
						AccessToken:  "access-token",
					},
				},
				Database: config.Database{
					Type: "postgres",
					SQLite: config.SQLite{
						Path: config.DefaultSQLitePath(),
					},
					Postgres: config.Postgres{
						Host:     "localhost",
						Port:     5432,
						Database: "dir",
						SSLMode:  "auto",
					},
				},
				APIServer: config.APIServer{
					ListenAddress: "example.com:8889",
					Connection:    config.DefaultConnection(),
					Authn: auth.Authn{
						Enabled:   false,
						Mode:      auth.ModeX509,
						Audiences: []string{},
					},
					Routing: config.Routing{
						ListenAddress: "/ip4/1.1.1.1/tcp/1",
						BootstrapPeers: []string{
							"/ip4/1.1.1.1/tcp/1",
							"/ip4/1.1.1.1/tcp/2",
						},
						KeyPath: "/path/to/key",
						GossipSub: config.GossipSub{
							Enabled: true,
						},
					},
					Sync: config.Sync{
						RegistryAuth: config.RegistryAuth{
							Username: "sync-user",
							Password: "sync-password",
						},
					},
					Authz: auth.Authz{
						Enabled:                true,
						EnforcerPolicyFilePath: "/tmp/authz_policies.csv",
					},
					Publication: config.Publication{
						SchedulerInterval: 10 * time.Second,
						WorkerCount:       1,
						WorkerTimeout:     10 * time.Second,
					},
					Metrics: config.Metrics{
						Enabled: true,
						Address: ":9090",
					},
					Naming: naming.Naming{
						Enabled: naming.DefaultEnabled,
						TTL:     naming.DefaultTTL,
					},
					HTTPGateway: config.HTTPGateway{
						Enabled:       true,
						ListenAddress: "address:123",
						PublicURL:     "https://example.com",
						CatalogTitle:  "Cisco AI Catalog",
					},
					Events: config.Events{
						SubscriberBufferSize: config.DefaultEventsSubscriberBufferSize,
						LogSlowConsumers:     config.DefaultEventsLogSlowConsumers,
						LogPublishedEvents:   config.DefaultEventsLogPublishedEvents,
					},
				},
				Reconciler: config.Reconciler{
					Regsync: reconcilercfg.Regsync{
						Enabled:  true,
						Interval: reconcilercfg.DefaultRegsyncInterval,
						Timeout:  reconcilercfg.DefaultRegsyncTimeout,
						Authn:    auth.Authn{Enabled: false, Mode: auth.ModeX509},
					},
					Indexer: reconcilercfg.Indexer{
						Enabled:  true,
						Interval: reconcilercfg.DefaultIndexerInterval,
					},
					Name: reconcilercfg.Name{
						Enabled:       false,
						Interval:      reconcilercfg.DefaultNameInterval,
						TTL:           naming.DefaultTTL,
						RecordTimeout: reconcilercfg.DefaultNameRecordTimeout,
					},
					Signature: reconcilercfg.Signature{
						Enabled:       true,
						Interval:      reconcilercfg.DefaultSignatureInterval,
						TTL:           reconcilercfg.DefaultSignatureTTL,
						RecordTimeout: reconcilercfg.DefaultSignatureRecordTimeout,
					},
					Metrics: reconcilercfg.Metrics{
						Enabled:  true,
						Interval: reconcilercfg.DefaultMetricsInterval,
					},
				},
			},
		},
		{
			Name:    "Default config",
			EnvVars: map[string]string{},
			ExpectedConfig: &config.Config{
				OASFAPIValidation: config.OASFAPIValidation{
					SchemaURL: "",
				},
				Store: config.Registry{
					RegistryAddress: config.DefaultRegistryAddress,
					RepositoryName:  config.DefaultRepositoryName,
					RegistryAuth: config.RegistryAuth{
						Insecure: config.DefaultRegistryAuthInsecure,
					},
				},
				Database: config.Database{
					Type: config.DefaultDatabaseType,
					SQLite: config.SQLite{
						Path: config.DefaultSQLitePath(),
					},
					Postgres: config.Postgres{
						Host:     config.DefaultPostgresHost,
						Port:     config.DefaultPostgresPort,
						Database: config.DefaultPostgresDatabase,
						SSLMode:  config.DefaultPostgresSSLMode,
					},
				},
				APIServer: config.APIServer{
					ListenAddress: config.DefaultListenAddress,
					Connection:    config.DefaultConnection(),
					Authn: auth.Authn{
						Enabled:   false,
						Mode:      auth.ModeX509,
						Audiences: []string{},
					},
					Routing: config.Routing{
						ListenAddress:  config.DefaultRoutingListenAddress,
						BootstrapPeers: config.DefaultBootstrapPeers,
						GossipSub: config.GossipSub{
							Enabled: config.DefaultGossipSubEnabled,
						},
					},
					Sync: config.Sync{
						RegistryAuth: config.RegistryAuth{},
					},
					Authz: auth.Authz{
						Enabled:                false,
						EnforcerPolicyFilePath: config.DefaultConfigPath + "/authz_policies.csv",
					},
					Publication: config.Publication{
						SchedulerInterval: config.DefaultPublicationSchedulerInterval,
						WorkerCount:       config.DefaultPublicationWorkerCount,
						WorkerTimeout:     config.DefaultPublicationWorkerTimeout,
					},
					Metrics: config.Metrics{
						Enabled: config.DefaultMetricsEnabled,
						Address: config.DefaultMetricsAddress,
					},
					Naming: naming.Naming{
						Enabled: naming.DefaultEnabled,
						TTL:     naming.DefaultTTL,
					},
					HTTPGateway: config.HTTPGateway{
						Enabled:       config.DefaultHTTPGatewayEnabled,
						ListenAddress: config.DefaultHTTPGatewayAddress,
						PublicURL:     config.DefaultHTTPGatewayPublicURL,
						CatalogTitle:  config.DefaultHTTPGatewayCatalogTitle,
					},
					Events: config.Events{
						SubscriberBufferSize: config.DefaultEventsSubscriberBufferSize,
						LogSlowConsumers:     config.DefaultEventsLogSlowConsumers,
						LogPublishedEvents:   config.DefaultEventsLogPublishedEvents,
					},
				},
				Reconciler: config.Reconciler{
					Regsync: reconcilercfg.Regsync{
						Enabled:  true,
						Interval: reconcilercfg.DefaultRegsyncInterval,
						Timeout:  reconcilercfg.DefaultRegsyncTimeout,
						Authn:    auth.Authn{Enabled: false, Mode: auth.ModeX509},
					},
					Indexer: reconcilercfg.Indexer{
						Enabled:  true,
						Interval: reconcilercfg.DefaultIndexerInterval,
					},
					Name: reconcilercfg.Name{
						Enabled:       false,
						Interval:      reconcilercfg.DefaultNameInterval,
						TTL:           naming.DefaultTTL,
						RecordTimeout: reconcilercfg.DefaultNameRecordTimeout,
					},
					Signature: reconcilercfg.Signature{
						Enabled:       true,
						Interval:      reconcilercfg.DefaultSignatureInterval,
						TTL:           reconcilercfg.DefaultSignatureTTL,
						RecordTimeout: reconcilercfg.DefaultSignatureRecordTimeout,
					},
					Metrics: reconcilercfg.Metrics{
						Enabled:  true,
						Interval: reconcilercfg.DefaultMetricsInterval,
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			for k, v := range test.EnvVars {
				t.Setenv(k, v)
			}

			cfg, err := serverLoadConfig(t)
			assert.NoError(t, err)
			assert.Equal(t, *cfg, *test.ExpectedConfig)
		})
	}
}

// TestConfig_SchemaURL tests that OASF schema URL configuration is correctly parsed.
func TestConfig_SchemaURL(t *testing.T) {
	tests := []struct {
		name              string
		envVars           map[string]string
		expectedSchemaURL string
	}{
		{
			name:              "empty schema URL when not configured",
			envVars:           map[string]string{},
			expectedSchemaURL: "",
		},
		{
			name: "custom schema URL",
			envVars: map[string]string{
				"DIRECTORY_OASF_API_VALIDATION_SCHEMA_URL": "https://custom.schema.url",
			},
			expectedSchemaURL: "https://custom.schema.url",
		},
		{
			name: "explicitly empty schema URL",
			envVars: map[string]string{
				"DIRECTORY_OASF_API_VALIDATION_SCHEMA_URL": "",
			},
			expectedSchemaURL: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			cfg, err := serverLoadConfig(t)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedSchemaURL, cfg.OASFAPIValidation.SchemaURL)
		})
	}
}

// TestConfig_RateLimiting tests that rate limiting configuration is correctly parsed.
func TestConfig_RateLimiting(t *testing.T) {
	tests := []struct {
		name           string
		envVars        map[string]string
		expectedConfig config.RateLimit
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
			expectedConfig: config.RateLimit{
				Enabled:        true,
				GlobalRPS:      50.0,
				GlobalBurst:    100,
				PerClientRPS:   500.0,
				PerClientBurst: 1000,
				MethodLimits:   map[string]config.MethodLimit{},
			},
		},
		{
			name: "rate limiting disabled (default)",
			envVars: map[string]string{
				"DIRECTORY_SERVER_RATELIMIT_ENABLED": "false",
			},
			expectedConfig: config.RateLimit{
				Enabled:        false,
				GlobalRPS:      0,
				GlobalBurst:    0,
				PerClientRPS:   0,
				PerClientBurst: 0,
				MethodLimits:   map[string]config.MethodLimit{},
			},
		},
		{
			name: "rate limiting with partial configuration",
			envVars: map[string]string{
				"DIRECTORY_SERVER_RATELIMIT_ENABLED":      "true",
				"DIRECTORY_SERVER_RATELIMIT_GLOBAL_RPS":   "200.0",
				"DIRECTORY_SERVER_RATELIMIT_GLOBAL_BURST": "400",
			},
			expectedConfig: config.RateLimit{
				Enabled:        true,
				GlobalRPS:      200.0,
				GlobalBurst:    400,
				PerClientRPS:   0,
				PerClientBurst: 0,
				MethodLimits:   map[string]config.MethodLimit{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			cfg, err := serverLoadConfig(t)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedConfig.Enabled, cfg.APIServer.RateLimit.Enabled)
			assert.InDelta(t, tt.expectedConfig.GlobalRPS, cfg.APIServer.RateLimit.GlobalRPS, 0.001)
			assert.Equal(t, tt.expectedConfig.GlobalBurst, cfg.APIServer.RateLimit.GlobalBurst)
			assert.InDelta(t, tt.expectedConfig.PerClientRPS, cfg.APIServer.RateLimit.PerClientRPS, 0.001)
			assert.Equal(t, tt.expectedConfig.PerClientBurst, cfg.APIServer.RateLimit.PerClientBurst)
		})
	}
}

// TestConfig_RateLimitingValidation tests that invalid rate limiting configuration
// is properly validated.
func TestConfig_RateLimitingValidation(t *testing.T) {
	tests := []struct {
		name        string
		cfg         config.RateLimit
		shouldError bool
	}{
		{
			name: "valid rate limiting configuration",
			cfg: config.RateLimit{
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
			cfg: config.RateLimit{
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
			cfg: config.RateLimit{
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
			cfg: config.RateLimit{
				Enabled:        false,
				GlobalRPS:      -100.0,
				GlobalBurst:    -200,
				PerClientRPS:   -1000.0,
				PerClientBurst: -2000,
			},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
