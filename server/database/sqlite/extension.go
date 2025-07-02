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

func (extension *Extension) GetAnnotations() map[string]string {
	// SQLite extensions don't store annotations, return empty map
	return make(map[string]string)
}

func (extension *Extension) GetName() string {
	return extension.Name
}

func (extension *Extension) GetVersion() string {
	return extension.Version
}

func (extension *Extension) GetData() map[string]any {
	// SQLite extensions don't store data, return empty map
	return make(map[string]any)
}

func (d *DB) addExtensionTx(tx *gorm.DB, extension types.Extension, agentID uint) (uint, error) {
	sqliteExtension := &Extension{
		AgentID: agentID,
		Name:    extension.GetName(),
		Version: extension.GetVersion(),
	}

	if err := tx.Create(sqliteExtension).Error; err != nil {
		return 0, fmt.Errorf("failed to add extension to SQLite database: %w", err)
	}

	logger.Debug("Added extension to SQLite database", "agent_id", agentID, "extension_id", sqliteExtension.ID)

	return sqliteExtension.ID, nil
}
