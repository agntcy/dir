// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sqlite

import (
	"fmt"
	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	searchtypes "github.com/agntcy/dir/server/search/types"
	"gorm.io/gorm"
)

func (s *SQLiteDB) AddLocatorTx(tx *gorm.DB, coreLocator *coretypes.Locator, agentID uint) (uint, error) {
	locator := &searchtypes.Locator{}
	if err := locator.FromCoreLocator(coreLocator, agentID); err != nil {
		return 0, fmt.Errorf("failed to convert core locator to search locator: %w", err)
	}

	if err := tx.Create(locator).Error; err != nil {
		return 0, fmt.Errorf("failed to add locator to SQLite search database: %w", err)
	}

	logger.Info("Added locator to SQLite search database", "type", locator.Type, "url", locator.URL)

	return locator.ID, nil
}
