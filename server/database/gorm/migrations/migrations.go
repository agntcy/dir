// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package migrations

import (
	"slices"
	"strings"

	"gorm.io/gorm"
)

type Migration struct {
	ID      string
	Details string
	Run     func(db *gorm.DB) error
}

// Register migrations in the order they should be applied. Migrations must be
// immutable once added to preserve the integrity of the migration history.
var migrations = []Migration{}

// Register adds a new migration to the list of available migrations. Migrations
// should be registered in the init() function of their respective files to ensure they are
// included in the migration process.
func register(m Migration) {
	migrations = append(migrations, m)

	// sort migrations by ID to ensure they are applied in the correct order
	slices.SortFunc(migrations, func(a, b Migration) int {
		return strings.Compare(a.ID, b.ID)
	})
}

func GetMigrations() []Migration {
	return migrations
}
