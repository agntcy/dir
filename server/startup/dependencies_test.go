// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package startup

import (
	"context"
	"errors"
	"testing"
	"time"

	dbconfig "github.com/agntcy/dir/server/database/config"
	startupconfig "github.com/agntcy/dir/server/startup/config"
	ociconfig "github.com/agntcy/dir/server/store/oci/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testStartupConfig(timeout time.Duration) startupconfig.Config {
	return startupconfig.Config{
		WaitPostgreSQL:  true,
		WaitOCIRegistry: true,
		Timeout:         timeout,
		InitialBackoff:  startupconfig.DefaultInitialBackoff,
		MaxBackoff:      startupconfig.DefaultMaxBackoff,
	}
}

func TestWaitForPostgreSQLSkipsNonPostgres(t *testing.T) {
	t.Parallel()

	cfg := dbconfig.Config{Type: "sqlite"}

	err := WaitForPostgreSQL(context.Background(), testStartupConfig(time.Second), cfg)
	require.NoError(t, err)
}

func TestWaitForPostgreSQLSkipsWhenDisabled(t *testing.T) {
	t.Parallel()

	cfg := dbconfig.Config{
		Type: "postgres",
		Postgres: dbconfig.PostgresConfig{
			Host:     "127.0.0.1",
			Port:     1,
			Database: "dir",
		},
	}
	startupCfg := startupconfig.Config{WaitPostgreSQL: false}

	err := WaitForPostgreSQL(context.Background(), startupCfg, cfg)
	require.NoError(t, err)
}

func TestWaitForOCIRegistrySkipsLocalDir(t *testing.T) {
	t.Parallel()

	cfg := ociconfig.Config{LocalDir: "/tmp/local-oci"}

	err := WaitForOCIRegistry(context.Background(), testStartupConfig(time.Second), cfg)
	require.NoError(t, err)
}

func TestWaitForOCIRegistrySkipsWhenDisabled(t *testing.T) {
	t.Parallel()

	cfg := ociconfig.Config{
		RegistryAddress: "127.0.0.1:1",
		RepositoryName:  "dir",
		AuthConfig: ociconfig.AuthConfig{
			Insecure: true,
		},
	}
	startupCfg := startupconfig.Config{WaitOCIRegistry: false}

	err := WaitForOCIRegistry(context.Background(), startupCfg, cfg)
	require.NoError(t, err)
}

func TestWaitForPostgreSQLTimesOut(t *testing.T) {
	t.Parallel()

	cfg := dbconfig.Config{
		Type: "postgres",
		Postgres: dbconfig.PostgresConfig{
			Host:     "127.0.0.1",
			Port:     1,
			Database: "dir",
			Username: "dir",
			Password: "dir",
		},
	}

	err := WaitForPostgreSQL(context.Background(), testStartupConfig(200*time.Millisecond), cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "postgresql not ready after")
}

func TestWaitForPostgreSQLRespectsContextCancellation(t *testing.T) {
	t.Parallel()

	cfg := dbconfig.Config{
		Type: "postgres",
		Postgres: dbconfig.PostgresConfig{
			Host:     "127.0.0.1",
			Port:     1,
			Database: "dir",
			Username: "dir",
			Password: "dir",
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := WaitForPostgreSQL(ctx, testStartupConfig(DefaultDependencyWaitTimeout), cfg)
	require.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled))
}

func TestWaitForPostgreSQLRetriesWithBackoff(t *testing.T) {
	t.Parallel()

	cfg := dbconfig.Config{
		Type: "postgres",
		Postgres: dbconfig.PostgresConfig{
			Host:     "127.0.0.1",
			Port:     1,
			Database: "dir",
			Username: "dir",
			Password: "dir",
		},
	}

	start := time.Now()
	err := WaitForPostgreSQL(context.Background(), testStartupConfig(300*time.Millisecond), cfg)
	elapsed := time.Since(start)

	require.Error(t, err)
	assert.GreaterOrEqual(t, elapsed, 450*time.Millisecond, "expected multiple backoff attempts before timeout")
}

func TestWaitForOCIRegistryTimesOut(t *testing.T) {
	t.Parallel()

	cfg := ociconfig.Config{
		RegistryAddress: "127.0.0.1:1",
		RepositoryName:  "dir",
		AuthConfig: ociconfig.AuthConfig{
			Insecure: true,
		},
	}

	err := WaitForOCIRegistry(context.Background(), testStartupConfig(200*time.Millisecond), cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "oci registry not ready after")
}

func TestWaitForDependenciesRunsPostgresThenOCI(t *testing.T) {
	t.Parallel()

	cfg := dbconfig.Config{
		Type: "postgres",
		Postgres: dbconfig.PostgresConfig{
			Host:     "127.0.0.1",
			Port:     1,
			Database: "dir",
			Username: "dir",
			Password: "dir",
		},
	}
	ociCfg := ociconfig.Config{
		RegistryAddress: "127.0.0.1:1",
		RepositoryName:  "dir",
		AuthConfig: ociconfig.AuthConfig{
			Insecure: true,
		},
	}

	err := WaitForDependencies(context.Background(), testStartupConfig(200*time.Millisecond), cfg, ociCfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "postgresql not ready after")
}

func TestIsPostgres(t *testing.T) {
	t.Parallel()

	assert.True(t, isPostgres(dbconfig.Config{}))
	assert.True(t, isPostgres(dbconfig.Config{Type: "postgres"}))
	assert.False(t, isPostgres(dbconfig.Config{Type: "sqlite"}))
}
