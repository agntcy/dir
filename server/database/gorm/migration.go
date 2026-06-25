// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"fmt"
	"time"

	"github.com/agntcy/dir/server/database/gorm/migrations"
)

type Migration struct {
	ID        string `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	Details   string `gorm:"not null"`
}

func (db *DB) HasMigration(id string) (bool, error) {
	result := db.gormDB.
		Model(&Migration{}).
		Where("id = ?", id).
		Limit(1).
		Find(&Migration{})
	if err := result.Error; err != nil {
		return false, err
	} else if result.RowsAffected == 0 {
		return false, nil
	}

	return true, nil
}

func (db *DB) AddMigration(id, details string) error {
	migration := Migration{
		ID:      id,
		Details: details,
	}

	return db.gormDB.Create(&migration).Error
}

// migrate executes all pending migrations in order, logging progress and
// returning an error if any migration fails. Migrations that have already been
// applied are skipped, so it's safe to run this on every startup.
func (db *DB) migrate() error {
	// Migrate migration schema
	if err := db.gormDB.AutoMigrate(Migration{}); err != nil {
		return fmt.Errorf("failed to migrate migration schema: %w", err)
	}

	// Run custom migrations (SQL only)
	for _, m := range migrations.GetMigrations() {
		logger.Info("running migration", "id", m.ID, "details", m.Details)

		// Check if it should be executed
		if hasMigration, err := db.HasMigration(m.ID); err != nil {
			return fmt.Errorf("failed to check migration history for %s: %w", m.ID, err)
		} else if hasMigration {
			logger.Info("skipping migration that has already been applied", "id", m.ID)

			continue
		}

		// Run migration
		if err := m.Run(db.gormDB); err != nil {
			return fmt.Errorf("migration %s failed: %w", m.ID, err)
		}

		logger.Info("migration completed successfully", "id", m.ID)

		// Write migration status
		if err := db.AddMigration(m.ID, m.Details); err != nil {
			return fmt.Errorf("failed to write migration history for %s: %w", m.ID, err)
		}

		logger.Info("migration history updated", "id", m.ID)
	}

	// Migrate object schemas
	if err := db.gormDB.AutoMigrate(
		Record{},
		Locator{},
		Skill{},
		Module{},
		Domain{},
		Annotation{},
		Sync{},
		Publication{},
		NameVerification{},
		SignatureVerification{},
		RecordUsageMetrics{},
	); err != nil {
		return fmt.Errorf("failed to migrate object schema: %w", err)
	}

	return nil
}
