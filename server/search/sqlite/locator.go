// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sqlite

import (
	"fmt"
	coretypesv2 "github.com/agntcy/dir/api/core/v1alpha2"
	"gorm.io/gorm"
)

type LocatorObject interface {
	ToSQLiteLocator(agentID uint) (*coretypesv2.SQLiteLocator, error)
}

func (s *SQLiteDB) addLocatorTx(tx *gorm.DB, locator LocatorObject, agentID uint) (uint, error) {
	SQLLocator, err := locator.ToSQLiteLocator(agentID)
	if err != nil {
		return 0, fmt.Errorf("failed to convert locator to SQLite locator: %w", err)
	}

	if err := tx.Create(SQLLocator).Error; err != nil {
		return 0, fmt.Errorf("failed to add locator to SQLite search database: %w", err)
	}

	logger.Info("Added locator to SQLite search database", "agent_id", agentID, "SQLLocator_ID", SQLLocator.ID)

	return SQLLocator.ID, nil
}
