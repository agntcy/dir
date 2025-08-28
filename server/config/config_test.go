// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:testifylint
package config

import (
	"testing"
	"time"

	authz "github.com/agntcy/dir/server/authz/config"
	database "github.com/agntcy/dir/server/database/config"
	sqliteconfig "github.com/agntcy/dir/server/database/sqlite/config"
	routing "github.com/agntcy/dir/server/routing/config"
	localfs "github.com/agntcy/dir/server/store/localfs/config"
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
				"DIRECTORY_SERVER_HEALTHCHECK_ADDRESS":                  "example.com:18888",
				"DIRECTORY_SERVER_PROVIDER":                             "provider",
				"DIRECTORY_SERVER_LOCALFS_DIR":                          "local-dir-fs",
				"DIRECTORY_SERVER_OCI_LOCAL_DIR":                        "local-dir",
				"DIRECTORY_SERVER_OCI_REGISTRY_ADDRESS":                 "example.com:5001",
				"DIRECTORY_SERVER_OCI_REPOSITORY_NAME":                  "test-dir",
				"DIRECTORY_SERVER_OCI_AUTH_CONFIG_INSECURE":             "true",
				"DIRECTORY_SERVER_OCI_AUTH_CONFIG_USERNAME":             "username",
				"DIRECTORY_SERVER_OCI_AUTH_CONFIG_PASSWORD":             "password",
				"DIRECTORY_SERVER_OCI_AUTH_CONFIG_ACCESS_TOKEN":         "access-token",
				"DIRECTORY_SERVER_OCI_AUTH_CONFIG_REFRESH_TOKEN":        "refresh-token",
				"DIRECTORY_SERVER_ROUTING_LISTEN_ADDRESS":               "/ip4/1.1.1.1/tcp/1",
				"DIRECTORY_SERVER_ROUTING_BOOTSTRAP_PEERS":              "/ip4/1.1.1.1/tcp/1,/ip4/1.1.1.1/tcp/2",
				"DIRECTORY_SERVER_ROUTING_KEY_PATH":                     "/path/to/key",
				"DIRECTORY_SERVER_DATABASE_DB_TYPE":                     "sqlite",
				"DIRECTORY_SERVER_DATABASE_SQLITE_DB_PATH":              "sqlite.db",
				"DIRECTORY_SERVER_SYNC_SCHEDULER_INTERVAL":              "1s",
				"DIRECTORY_SERVER_SYNC_WORKER_COUNT":                    "1",
				"DIRECTORY_SERVER_SYNC_REGISTRY_MONITOR_CHECK_INTERVAL": "10s",
				"DIRECTORY_SERVER_SYNC_WORKER_TIMEOUT":                  "10s",
				"DIRECTORY_SERVER_AUTHZ_SOCKET_PATH":                    "/test/agent.sock",
				"DIRECTORY_SERVER_AUTHZ_TRUST_DOMAIN":                   "dir.com",
			},
			ExpectedConfig: &Config{
				ListenAddress:      "example.com:8889",
				HealthCheckAddress: "example.com:18888",
				Provider:           "provider",
				LocalFS: localfs.Config{
					Dir: "local-dir-fs",
				},
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
				Routing: routing.Config{
					ListenAddress: "/ip4/1.1.1.1/tcp/1",
					BootstrapPeers: []string{
						"/ip4/1.1.1.1/tcp/1",
						"/ip4/1.1.1.1/tcp/2",
					},
					KeyPath: "/path/to/key",
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
				},
				Authz: authz.Config{
					SocketPath:  "/test/agent.sock",
					TrustDomain: "dir.com",
				},
			},
		},
		{
			Name:    "Default config",
			EnvVars: map[string]string{},
			ExpectedConfig: &Config{
				ListenAddress:      DefaultListenAddress,
				HealthCheckAddress: DefaultHealthCheckAddress,
				Provider:           DefaultProvider,
				LocalFS: localfs.Config{
					Dir: localfs.DefaultDir,
				},
				OCI: oci.Config{
					RegistryAddress: oci.DefaultRegistryAddress,
					RepositoryName:  oci.DefaultRepositoryName,
					AuthConfig: oci.AuthConfig{
						Insecure: oci.DefaultAuthConfigInsecure,
					},
				},
				Routing: routing.Config{
					ListenAddress:  routing.DefaultListenAddress,
					BootstrapPeers: routing.DefaultBootstrapPeers,
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
