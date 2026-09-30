// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"context"
	"fmt"

	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	"gorm.io/gorm"
)

var logger = logging.Logger("database/gorm")

type DB struct {
	gormDB *gorm.DB

	// enforcement returns what reads must satisfy; see applyPolicyGate and
	// IsRecordServable. Nil applies no policy.
	enforcement func() types.PolicyEnforcement

	// observer is told about fetches the policies exclude. Nil tells no one.
	observer types.PolicyGateObserver
}

// New creates a new DB instance from a gorm.DB connection and runs migrations.
// Its reads apply no content policy; see Served.
func New(db *gorm.DB) (*DB, error) {
	database := &DB{gormDB: db}

	// Execute migrations
	if err := database.migrate(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return database, nil
}

// Served returns a view of d, on the same connection, whose reads apply the
// content policies enforcement returns. enforcement is called per read, so
// what is enforced may change at runtime. d itself keeps applying none:
// background tasks read through it and must see every record.
func (d *DB) Served(enforcement func() types.PolicyEnforcement, observer types.PolicyGateObserver) *DB {
	return &DB{gormDB: d.gormDB, enforcement: enforcement, observer: observer}
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

// Close closes the database connection.
func (d *DB) Close() error {
	if d.gormDB == nil {
		return nil
	}

	sqlDB, err := d.gormDB.DB()
	if err != nil {
		return fmt.Errorf("failed to get SQL DB: %w", err)
	}

	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("failed to close SQL DB: %w", err)
	}

	return nil
}
