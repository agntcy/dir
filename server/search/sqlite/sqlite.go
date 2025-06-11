// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sqlite

import (
	"fmt"

	coretypesv2 "github.com/agntcy/dir/api/core/v1alpha2"

	"github.com/agntcy/dir/utils/logging"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var logger = logging.Logger("store/oci")

type SQLiteDB struct {
	gormDB *gorm.DB
}

func New() (*SQLiteDB, error) {
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SQLite database: %w", err)
	}

	// Migrate the schema
	if err := db.AutoMigrate(
		coretypesv2.Record{},
		coretypesv2.Extension{},
		coretypesv2.Locator{},
		coretypesv2.Signature{},
		coretypesv2.Skill{},
	); err != nil {
		return nil, fmt.Errorf("failed to migrate schema: %w", err)
	}

	return &SQLiteDB{
		gormDB: db,
	}, nil
}
