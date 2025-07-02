// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sqlite

import (
	"fmt"

	"github.com/agntcy/dir/utils/logging"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

var logger = logging.Logger("database/sqlite")

type DB struct {
	gormDB *gorm.DB
}

func New(path string) (*DB, error) {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
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
