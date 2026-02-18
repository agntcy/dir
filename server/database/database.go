// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/agntcy/dir/server/database/config"
	gormdb "github.com/agntcy/dir/server/database/gorm"
	"github.com/agntcy/dir/server/types"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type DB string

const (
	Postgres DB = "postgres"
)

func New(config config.Config) (types.DatabaseAPI, error) {
	switch db := DB(config.Type); db {
	case Postgres:
		postgresDB, err := newPostgres(config.Postgres)
		if err != nil {
			return nil, fmt.Errorf("failed to create PostgreSQL database: %w", err)
		}

		return postgresDB, nil
	default:
		return nil, fmt.Errorf("unsupported database=%s", db)
	}
}

func newCustomLogger() gormlogger.Interface {
	// Create a custom logger configuration that ignores "record not found" errors
	// since these are expected during normal operation (checking if records exist)
	return gormlogger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		gormlogger.Config{
			SlowThreshold:             200 * time.Millisecond, //nolint:mnd
			LogLevel:                  gormlogger.Warn,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)
}

// newPostgres creates a new database connection using PostgreSQL driver.
func newPostgres(cfg config.PostgresConfig) (*gormdb.DB, error) {
	host := cfg.Host
	if host == "" {
		host = config.DefaultPostgresHost
	}

	port := cfg.Port
	if port == 0 {
		port = config.DefaultPostgresPort
	}

	database := cfg.Database
	if database == "" {
		database = config.DefaultPostgresDatabase
	}

	sslMode := "disable"
	if cfg.SSLMode != "" {
		sslMode = cfg.SSLMode
	}

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		host, port, cfg.Username, cfg.Password, database, sslMode)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: newCustomLogger(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL database: %w", err)
	}

	gdb, err := gormdb.New(db)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize PostgreSQL database: %w", err)
	}

	return gdb, nil
}
