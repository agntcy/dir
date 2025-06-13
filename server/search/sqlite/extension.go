// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sqlite

import (
	"fmt"
	coretypesv2 "github.com/agntcy/dir/api/core/v1alpha2"
	"gorm.io/gorm"
)

type ExtensionObject interface {
	ToSQLiteExtension(agentID uint) (*coretypesv2.SQLiteExtension, error)
}

func (s *SQLiteDB) addExtensionTx(tx *gorm.DB, extension ExtensionObject, agentID uint) (uint, error) {
	SQLExtension, err := extension.ToSQLiteExtension(agentID)
	if err != nil {
		return 0, fmt.Errorf("failed to convert core extension to SQLite extension: %w", err)
	}

	if err := tx.Create(SQLExtension).Error; err != nil {
		return 0, fmt.Errorf("failed to add extension to SQLite search database: %w", err)
	}

	logger.Info("Added extension to SQLite search database", "agent_id", agentID, "SQLExtension_ID", SQLExtension.ID)

	return SQLExtension.ID, nil
}
