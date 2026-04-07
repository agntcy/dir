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

// newSQLite creates a new database connection using the pure-Go SQLite driver.
func NewSqlite() (*gorm.DB, error) {
	// In case of SQLite, we always use a temporary file for the database
	// since the state will be managed by the runtime server and wont persist across sessions.
	// The runtime rebuilds the database on startup.
	tmpFile, err := os.CreateTemp("", "dir-runtime-*.db")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary file for SQLite database: %w", err)
	}

	// Close the file immediately since gorm will manage the connection to it. We just need the path.
	if err := tmpFile.Close(); err != nil {
		return nil, fmt.Errorf("failed to close temporary file for SQLite database: %w", err)
	}

	// Create database
	db, err := gorm.Open(sqlite.Open(tmpFile.Name()), &gorm.Config{
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
