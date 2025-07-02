// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sqlite

import (
	"fmt"

	"github.com/agntcy/dir/server/types"
	"gorm.io/gorm"
)

type Locator struct {
	gorm.Model
	AgentID uint   `gorm:"not null;index"`
	Type    string `gorm:"not null"`
	URL     string `gorm:"not null"`
}

func (locator *Locator) GetAnnotations() map[string]string {
	// SQLite locators don't store annotations, return empty map
	return make(map[string]string)
}

func (locator *Locator) GetType() string {
	return locator.Type
}

func (locator *Locator) GetURL() string {
	return locator.URL
}

func (locator *Locator) GetSize() uint64 {
	// SQLite locators don't store size information
	return 0
}

func (locator *Locator) GetDigest() string {
	// SQLite locators don't store digest information
	return ""
}

func (d *DB) addLocatorTx(tx *gorm.DB, locator types.Locator, agentID uint) (uint, error) {
	sqliteLocator := &Locator{
		AgentID: agentID,
		Type:    locator.GetType(),
		URL:     locator.GetURL(),
	}

	if err := tx.Create(sqliteLocator).Error; err != nil {
		return 0, fmt.Errorf("failed to add locator to SQLite database: %w", err)
	}

	logger.Debug("Added locator to SQLite database", "agent_id", agentID, "locator_id", sqliteLocator.ID)

	return sqliteLocator.ID, nil
}
