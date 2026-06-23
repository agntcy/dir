// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package startup provides dependency readiness waits used during service boot.
package startup

import (
	"context"
	"fmt"
	"time"

	"github.com/agntcy/dir/server/database"
	dbconfig "github.com/agntcy/dir/server/database/config"
	startupconfig "github.com/agntcy/dir/server/startup/config"
	"github.com/agntcy/dir/server/store/oci"
	ociconfig "github.com/agntcy/dir/server/store/oci/config"
	"github.com/agntcy/dir/utils/logging"
)

const dependencyCheckTimeout = 3 * time.Second

var logger = logging.Logger("startup")

// DefaultDependencyWaitTimeout is kept for backward compatibility with callers using the constant directly.
const DefaultDependencyWaitTimeout = startupconfig.DefaultDependencyWaitTimeout

// DefaultInitialBackoff is kept for backward compatibility with callers using the constant directly.
const DefaultInitialBackoff = startupconfig.DefaultInitialBackoff

// DefaultMaxBackoff is kept for backward compatibility with callers using the constant directly.
const DefaultMaxBackoff = startupconfig.DefaultMaxBackoff

// WaitForDependencies waits for PostgreSQL and the OCI registry to become ready.
func WaitForDependencies(
	ctx context.Context,
	startupCfg startupconfig.Config,
	dbCfg dbconfig.Config,
	ociCfg ociconfig.Config,
) error {
	if err := WaitForPostgreSQL(ctx, startupCfg, dbCfg); err != nil {
		return err
	}

	return WaitForOCIRegistry(ctx, startupCfg, ociCfg)
}

// WaitForPostgreSQL waits until PostgreSQL accepts connections or the timeout expires.
// No-op when waiting is disabled or the configured database type is not PostgreSQL.
func WaitForPostgreSQL(ctx context.Context, startupCfg startupconfig.Config, cfg dbconfig.Config) error {
	startupCfg = startupCfg.WithDefaults()
	if !startupCfg.WaitPostgreSQL || !isPostgres(cfg) {
		return nil
	}

	host := cfg.Postgres.Host
	if host == "" {
		host = dbconfig.DefaultPostgresHost
	}

	port := cfg.Postgres.Port
	if port == 0 {
		port = dbconfig.DefaultPostgresPort
	}

	logger.Info("Waiting for PostgreSQL to become ready",
		"host", host,
		"port", port,
		"timeout", startupCfg.Timeout)

	return waitUntil(ctx, startupCfg, "postgresql", func(checkCtx context.Context) (bool, error) {
		if err := pingPostgreSQL(checkCtx, cfg.Postgres); err != nil {
			return false, err
		}

		return true, nil
	})
}

// WaitForOCIRegistry waits until the remote OCI registry is reachable or the timeout expires.
// No-op when waiting is disabled or a local OCI directory is configured.
func WaitForOCIRegistry(ctx context.Context, startupCfg startupconfig.Config, cfg ociconfig.Config) error {
	startupCfg = startupCfg.WithDefaults()
	if !startupCfg.WaitOCIRegistry || cfg.LocalDir != "" {
		return nil
	}

	addr, err := cfg.GetRegistryAddress()
	if err != nil {
		return fmt.Errorf("invalid OCI registry configuration: %w", err)
	}

	logger.Info("Waiting for OCI registry to become ready",
		"registry", addr,
		"timeout", startupCfg.Timeout)

	store, err := oci.New(cfg)
	if err != nil {
		return fmt.Errorf("failed to create OCI store probe: %w", err)
	}

	return waitUntil(ctx, startupCfg, "oci registry", func(checkCtx context.Context) (bool, error) {
		return store.IsReady(checkCtx), nil
	})
}

func isPostgres(cfg dbconfig.Config) bool {
	dbType := cfg.Type
	if dbType == "" {
		dbType = dbconfig.DefaultType
	}

	return dbType == string(database.Postgres)
}

func pingPostgreSQL(ctx context.Context, cfg dbconfig.PostgresConfig) error {
	db, err := database.NewPostgresGormDb(cfg)
	if err != nil {
		return fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get SQL DB handle: %w", err)
	}

	defer sqlDB.Close()

	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	return nil
}

func waitUntil(
	ctx context.Context,
	startupCfg startupconfig.Config,
	resourceName string,
	check func(context.Context) (bool, error),
) error {
	startupCfg = startupCfg.WithDefaults()

	deadline := time.Now().Add(startupCfg.Timeout)
	backoff := startupCfg.InitialBackoff
	attempt := 0

	for {
		attempt++

		checkCtx, cancel := context.WithTimeout(ctx, dependencyCheckTimeout)

		ready, err := check(checkCtx)

		cancel()

		if err == nil && ready {
			if attempt > 1 {
				logger.Info("Dependency became ready", "dependency", resourceName, "attempts", attempt)
			}

			return nil
		}

		if time.Now().After(deadline) {
			if err != nil {
				return fmt.Errorf("%s not ready after %s: %w", resourceName, startupCfg.Timeout, err)
			}

			return fmt.Errorf("%s not ready after %s", resourceName, startupCfg.Timeout)
		}

		logger.Warn("Dependency not ready, retrying",
			"dependency", resourceName,
			"attempt", attempt,
			"next_backoff", backoff,
			"error", err)

		if err := sleepWithContext(ctx, backoff); err != nil {
			return fmt.Errorf("waiting for %s cancelled: %w", resourceName, err)
		}

		backoff *= 2
		if backoff > startupCfg.MaxBackoff {
			backoff = startupCfg.MaxBackoff
		}
	}
}

func sleepWithContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return fmt.Errorf("sleep interrupted: %w", ctx.Err())
	case <-timer.C:
		return nil
	}
}
