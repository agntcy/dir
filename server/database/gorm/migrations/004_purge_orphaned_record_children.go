// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package migrations

import (
	"fmt"

	"gorm.io/gorm"
)

func init() {
	register(Migration{
		ID:      "004_purge_orphaned_record_children",
		Details: "Delete rows in record child tables whose record no longer exists.",
		Run:     runPurgeOrphanedRecordChildren,
	})
}

// orphanedChildTables are the tables that hang off records via ON DELETE
// CASCADE. SQLite enforces foreign keys per connection, and the pragma used to
// be issued as a statement after Open, so it only reached one connection in the
// pool. Deletes served by any other connection dropped the record and left its
// children behind.
//
// Orphaned label rows are the visible symptom: GetRecordLabels looks them up by
// record_cid, so re-pushing a deleted record (same content, same CID) returned
// each of its labels once per past life.
var orphanedChildTables = []string{
	"skills",
	"domains",
	"modules",
	"locators",
	"annotations",
	"signature_verifications",
	"scan_reports",
	"name_verifications",
	"record_usage_metrics",
}

func runPurgeOrphanedRecordChildren(db *gorm.DB) error {
	if !db.Migrator().HasTable("records") {
		return nil
	}

	for _, table := range orphanedChildTables {
		if !db.Migrator().HasTable(table) {
			continue
		}

		stmt := fmt.Sprintf(
			"DELETE FROM %s WHERE record_cid NOT IN (SELECT record_cid FROM records)",
			table,
		)

		if err := db.Exec(stmt).Error; err != nil {
			return fmt.Errorf("purge orphaned rows from %s: %w", table, err)
		}
	}

	return nil
}
