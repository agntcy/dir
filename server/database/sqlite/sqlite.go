// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sqlite

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/agntcy/dir/utils/logging"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

var logger = logging.Logger("database/sqlite")

type DB struct {
	gormDB *gorm.DB
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

func New(path string) (*DB, error) {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		Logger: newCustomLogger(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SQLite database: %w", err)
	}

	// Migrate record-related schema
	if err := db.AutoMigrate(Record{}, Locator{}, Skill{}, Module{}, Domain{}); err != nil {
		return nil, fmt.Errorf("failed to migrate record schema: %w", err)
	}

	// Migrate sync-related schema
	if err := db.AutoMigrate(Sync{}); err != nil {
		return nil, fmt.Errorf("failed to migrate sync schema: %w", err)
	}

	// Migrate publication-related schema
	if err := db.AutoMigrate(Publication{}); err != nil {
		return nil, fmt.Errorf("failed to migrate publication schema: %w", err)
	}

	return &DB{
		gormDB: db,
	}, nil
}

// IsReady checks if the database connection is ready to serve traffic.
// Returns true if the database connection is established and can execute queries.
func (d *DB) IsReady(ctx context.Context) bool {
	if d.gormDB == nil {
		logger.Debug("Database not ready: gormDB is nil")

		return false
	}

	// Get the underlying SQL database
	sqlDB, err := d.gormDB.DB()
	if err != nil {
		logger.Debug("Database not ready: failed to get SQL DB", "error", err)

		return false
	}

	// Ping the database with context
	if err := sqlDB.PingContext(ctx); err != nil {
		logger.Debug("Database not ready: ping failed", "error", err)

		return false
	}

	logger.Debug("Database ready")

	return true
}
