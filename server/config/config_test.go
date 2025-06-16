// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:testifylint
package config

import (
	"testing"

	routing "github.com/agntcy/dir/server/routing/config"
	search "github.com/agntcy/dir/server/search/config"
	sqliteconfig "github.com/agntcy/dir/server/search/sqlite/config"
	localfs "github.com/agntcy/dir/server/store/localfs/config"
	oci "github.com/agntcy/dir/server/store/oci/config"
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
				"DIRECTORY_SERVER_LISTEN_ADDRESS":                "example.com:8889",
				"DIRECTORY_SERVER_HEALTHCHECK_ADDRESS":           "example.com:18888",
				"DIRECTORY_SERVER_PROVIDER":                      "provider",
				"DIRECTORY_SERVER_LOCALFS_DIR":                   "local-dir-fs",
				"DIRECTORY_SERVER_OCI_LOCAL_DIR":                 "local-dir",
				"DIRECTORY_SERVER_OCI_REGISTRY_ADDRESS":          "example.com:5001",
				"DIRECTORY_SERVER_OCI_REPOSITORY_NAME":           "test-dir",
				"DIRECTORY_SERVER_OCI_AUTH_CONFIG_INSECURE":      "true",
				"DIRECTORY_SERVER_OCI_AUTH_CONFIG_USERNAME":      "username",
				"DIRECTORY_SERVER_OCI_AUTH_CONFIG_PASSWORD":      "password",
				"DIRECTORY_SERVER_OCI_AUTH_CONFIG_ACCESS_TOKEN":  "access-token",
				"DIRECTORY_SERVER_OCI_AUTH_CONFIG_REFRESH_TOKEN": "refresh-token",
				"DIRECTORY_SERVER_ROUTING_LISTEN_ADDRESS":        "/ip4/1.1.1.1/tcp/1",
				"DIRECTORY_SERVER_ROUTING_BOOTSTRAP_PEERS":       "/ip4/1.1.1.1/tcp/1,/ip4/1.1.1.1/tcp/2",
				"DIRECTORY_SERVER_ROUTING_KEY_PATH":              "/path/to/key",
				"DIRECTORY_SERVER_SEARCH_DB_TYPE":                "sqlite",
				"DIRECTORY_SERVER_SEARCH_SQLITE_DB_PATH":         "sqlite.db",
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
				Search: search.Config{
					DBType: "sqlite",
					SQLite: sqliteconfig.Config{
						DBPath: "sqlite.db",
					},
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
				Search: search.Config{
					DBType: search.DefaultDBType,
					SQLite: sqliteconfig.Config{
						DBPath: sqliteconfig.DefaultSQLiteDBPath,
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

			config, err := LoadConfig()
			assert.NoError(t, err)
			assert.Equal(t, *config, *test.ExpectedConfig)
		})
	}
}
