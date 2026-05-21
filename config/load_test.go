// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"strings"
	"testing"

	"github.com/agntcy/dir/config"
	"github.com/agntcy/dir/config/naming"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig_Defaults(t *testing.T) {
	t.Parallel()

	cfg, err := config.LoadConfig(
		config.WithReader(strings.NewReader(""), "yaml"),
	)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.Equal(t, config.DefaultRegistryAddress, cfg.Registry.RegistryAddress)
	require.Equal(t, config.DefaultRepositoryName, cfg.Registry.RepositoryName)

	require.Equal(t, config.DefaultDatabaseType, cfg.Database.Type)
	require.Equal(t, config.DefaultSQLitePath, cfg.Database.SQLite.Path)

	require.Equal(t, config.DefaultListenAddress, cfg.APIServer.ListenAddress)
	require.Equal(t, config.DefaultMetricsAddress, cfg.APIServer.Metrics.Address)

	require.Equal(t, config.DefaultMaxConcurrentStreams, int(cfg.APIServer.Connection.MaxConcurrentStreams))
	require.Equal(t, config.DefaultMaxRecvMsgSize, cfg.APIServer.Connection.MaxRecvMsgSize)
	require.Equal(t, config.DefaultKeepaliveTime, cfg.APIServer.Connection.Keepalive.Time)

	require.Equal(t, naming.DefaultTTL, cfg.APIServer.Naming.TTL)

	require.True(t, cfg.Reconciler.Regsync.Enabled)
	require.True(t, cfg.Reconciler.Indexer.Enabled)
	require.True(t, cfg.Reconciler.Signature.Enabled)
}

func TestLoadConfig_FromYAML(t *testing.T) {
	t.Parallel()

	yaml := `
registry:
  registry_address: "registry.example.com"
  repository_name: "custom/repo"
database:
  type: postgres
  postgres:
    host: db.example.com
    port: 6432
apiserver:
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

	require.Equal(t, "registry.example.com", cfg.Registry.RegistryAddress)
	require.Equal(t, "custom/repo", cfg.Registry.RepositoryName)
	require.Equal(t, "postgres", cfg.Database.Type)
	require.Equal(t, "db.example.com", cfg.Database.Postgres.Host)
	require.Equal(t, 6432, cfg.Database.Postgres.Port)
	require.Equal(t, "0.0.0.0:9999", cfg.APIServer.ListenAddress)
	require.True(t, cfg.APIServer.Authn.Enabled)
	require.Equal(t, "5m0s", cfg.Reconciler.Regsync.Interval.String())
}

func TestLoadConfig_EnvOverride(t *testing.T) {
	t.Setenv("DIRECTORY_REGISTRY_REGISTRY_ADDRESS", "env.registry:5000")
	t.Setenv("DIRECTORY_DATABASE_TYPE", "postgres")
	t.Setenv("DIRECTORY_APISERVER_LISTEN_ADDRESS", "0.0.0.0:7777")

	cfg, err := config.LoadConfig(config.WithReader(strings.NewReader(""), "yaml"))
	require.NoError(t, err)

	require.Equal(t, "env.registry:5000", cfg.Registry.RegistryAddress)
	require.Equal(t, "postgres", cfg.Database.Type)
	require.Equal(t, "0.0.0.0:7777", cfg.APIServer.ListenAddress)
}
