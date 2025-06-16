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

func (locator *Locator) GetType() string {
	return locator.Type
}

func (locator *Locator) GetUrl() string { //nolint:revive,stylecheck
	return locator.URL
}

func (d *DB) addLocatorTx(tx *gorm.DB, locatorObject types.LocatorObject, agentID uint) (uint, error) {
	locator := &Locator{
		AgentID: agentID,
		Type:    locatorObject.GetType(),
		URL:     locatorObject.GetUrl(),
	}

	if err := tx.Create(locator).Error; err != nil {
		return 0, fmt.Errorf("failed to add locator to SQLite search database: %w", err)
	}

	logger.Debug("Added locator to SQLite search database", "agent_id", agentID, "locator_id", locator.ID)

	return locator.ID, nil
}
