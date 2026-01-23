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
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type DB string

const (
	SQLite DB = "sqlite"
)

func New(opts types.APIOptions) (types.DatabaseAPI, error) {
	switch db := DB(opts.Config().Database.DBType); db {
	case SQLite:
		sqliteDB, err := newSQLite(opts.Config().Database.SQLite)
		if err != nil {
			return nil, fmt.Errorf("failed to create SQLite database: %w", err)
		}

		return sqliteDB, nil
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

	return gormdb.InitDB(db)
}
