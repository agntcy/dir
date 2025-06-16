// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sqlite

import (
	"fmt"

	"github.com/agntcy/dir/server/types"
	"gorm.io/gorm"
)

type Extension struct {
	gorm.Model
	AgentID uint   `gorm:"not null;index"`
	Name    string `gorm:"not null"`
	Version string `gorm:"not null"`
}

func (extension *Extension) GetName() string {
	return extension.Name
}

func (extension *Extension) GetVersion() string {
	return extension.Version
}

func (d *DB) addExtensionTx(tx *gorm.DB, extensionObject types.ExtensionObject, agentID uint) (uint, error) {
	extension := &Extension{
		AgentID: agentID,
		Name:    extensionObject.GetName(),
		Version: extensionObject.GetVersion(),
	}

	if err := tx.Create(extension).Error; err != nil {
		return 0, fmt.Errorf("failed to add extension to SQLite search database: %w", err)
	}

	logger.Debug("Added extension to SQLite search database", "agent_id", agentID, "extension_id", extension.ID)

	return extension.ID, nil
}
