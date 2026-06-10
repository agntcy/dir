// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package migrations

import (
	"fmt"

	"gorm.io/gorm"
)

func init() {
	register(Migration{
		ID:      "01_catalog_sync",
		Details: "Sync the database by dropping existing fields to regenrate index data.",
		Run: func(db *gorm.DB) error {
			// Drop tables to force regeneration of index data. This is a brute-force approach that
			// will cause downtime, but it's simpler than writing a reversible migration that
			// selectively backfills the new fields for existing records. Since this is a one-time
			// migration to fix a specific issue, the downtime is acceptable.
			if err := db.Migrator().
				DropTable(
					"records",
					"modules",
					"skills",
					"domains",
					"annotations",
					"signature_verifications",
					"name_verifications",
				); err != nil {
				return fmt.Errorf("failed to drop tables for catalog sync: %w", err)
			}

			return nil
		},
	})
}
