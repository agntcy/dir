// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package migrations

import (
	"fmt"

	"gorm.io/gorm"
)

const nameVerificationsTable = "name_verifications"

func init() {
	register(Migration{
		ID:      "005_drop_name_verifications",
		Details: "Drop the name_verifications table; ownership is now tracked by identity_claims.",
		Run:     runDropNameVerifications,
	})
}

func runDropNameVerifications(db *gorm.DB) error {
	if !db.Migrator().HasTable(nameVerificationsTable) {
		return nil
	}

	if err := db.Migrator().DropTable(nameVerificationsTable); err != nil {
		return fmt.Errorf("drop %s table: %w", nameVerificationsTable, err)
	}

	return nil
}
