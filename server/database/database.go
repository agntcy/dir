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
	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type DB string

const (
	SQLite   DB = "sqlite"
	Postgres DB = "postgres"
)

func New(opts types.APIOptions) (types.DatabaseAPI, error) {
	switch db := DB(opts.Config().Database.DBType); db {
	case SQLite:
		sqliteDB, err := newSQLite(opts.Config().Database.SQLite)
		if err != nil {
			return nil, fmt.Errorf("failed to create SQLite database: %w", err)
		}

		return sqliteDB, nil
	case Postgres:
		postgresDB, err := newPostgres(opts.Config().Database.Postgres)
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

// newSQLite creates a new database connection using SQLite driver.
func newSQLite(cfg config.SQLiteConfig) (*gormdb.DB, error) {
	path := cfg.DBPath
	if path == "" {
		path = config.DefaultSQLiteDBPath
	}

	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		Logger: newCustomLogger(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SQLite database: %w", err)
	}

	gdb, err := gormdb.New(db)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize SQLite database: %w", err)
	}

	return gdb, nil
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

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, cfg.Username, cfg.Password, database)

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
