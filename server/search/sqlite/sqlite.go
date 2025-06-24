// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sqlite

import (
	"database/sql"
	"fmt"
	"regexp"

	"github.com/agntcy/dir/utils/logging"
	"github.com/mattn/go-sqlite3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var logger = logging.Logger("store/oci")

type DB struct {
	gormDB *gorm.DB
}

func New(path string) (*DB, error) {
	// Register REGEXP with sqlite3 driver
	sql.Register("custom_sqlite3", &sqlite3.SQLiteDriver{
		ConnectHook: func(conn *sqlite3.SQLiteConn) error {
			return conn.RegisterFunc("regexp", func(re, s string) (bool, error) {
				r, err := regexp.Compile(re)
				if err != nil {
					logger.Debug("Invalid regex pattern", "pattern", re, "error", err)

					// Instead of returning error, return false for invalid regex
					return false, nil
				}

				return r.MatchString(s), nil
			}, true)
		},
	})

	sqldb, err := sql.Open("custom_sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database: %w", err)
	}

	db, err := gorm.Open(sqlite.Dialector{Conn: sqldb}, &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SQLite database: %w", err)
	}

	// Migrate the schema
	if err := db.AutoMigrate(Record{}, Extension{}, Locator{}, Skill{}); err != nil {
		return nil, fmt.Errorf("failed to migrate schema: %w", err)
	}

	return &DB{
		gormDB: db,
	}, nil
}
