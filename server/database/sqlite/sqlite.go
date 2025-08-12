// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sqlite

import (
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
	if err := db.AutoMigrate(Record{}, Extension{}, Locator{}, Skill{}); err != nil {
		return nil, fmt.Errorf("failed to migrate record schema: %w", err)
	}

	// Migrate sync-related schema
	if err := db.AutoMigrate(Sync{}); err != nil {
		return nil, fmt.Errorf("failed to migrate sync schema: %w", err)
	}

	return &DB{
		gormDB: db,
	}, nil
}
