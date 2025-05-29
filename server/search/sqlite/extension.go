// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sqlite

import (
	"fmt"
	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	searchtypes "github.com/agntcy/dir/server/search/types"
	"gorm.io/gorm"
)

func (s *SQLiteDB) AddExtensionTx(tx *gorm.DB, coreExtension *coretypes.Extension, agentID uint) (uint, error) {
	extension := &searchtypes.Extension{}
	if err := extension.FromCoreExtension(coreExtension, agentID); err != nil {
		return 0, fmt.Errorf("failed to convert core extension to search extension: %w", err)
	}

	if err := tx.Create(extension).Error; err != nil {
		return 0, fmt.Errorf("failed to add extension to SQLite search database: %w", err)
	}

	logger.Info("Added extension to SQLite search database", "name", extension.Name, "agentID", agentID)

	return extension.ID, nil
}
