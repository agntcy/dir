// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sql

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// StoreType is the identifier for the SQLite store type.
const StoreTypeSqlite = "sqlite"

const DefaultSqlitePath = "/tmp/dir.runtime.db"

// Config holds etcd connection configuration.
type SqliteConfig struct {
	// Path is the filesystem path to the SQLite database file.
	Path string `json:"path,omitempty" mapstructure:"path"`
}

// newSQLite creates a new database connection using the pure-Go SQLite driver.
func NewSqlite(cfg SqliteConfig) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(cfg.Path), &gorm.Config{
		Logger: gormlogger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			gormlogger.Config{
				SlowThreshold:             200 * time.Millisecond, //nolint:mnd
				LogLevel:                  gormlogger.Warn,
				IgnoreRecordNotFoundError: true,
				Colorful:                  true,
			},
		),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SQLite database: %w", err)
	}

	// SQLite does not enforce foreign keys by default; enable for CASCADE support.
	if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		return nil, fmt.Errorf("failed to enable SQLite foreign keys: %w", err)
	}

	return db, nil
}
