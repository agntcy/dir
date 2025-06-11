// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sqlite

import (
	"fmt"
	coretypesv2 "github.com/agntcy/dir/api/core/v1alpha2"
	"gorm.io/gorm"
)

type SkillObject interface {
	ToSQLiteSkill(agentID uint) (*coretypesv2.SQLiteSkill, error)
}

func (s *SQLiteDB) addSkillTx(tx *gorm.DB, skill SkillObject, agentID uint) (uint, error) {
	SQLSkill, err := skill.ToSQLiteSkill(agentID)
	if err != nil {
		return 0, fmt.Errorf("failed to convert skill to SQLite skill: %w", err)
	}

	if err := tx.Create(SQLSkill).Error; err != nil {
		return 0, fmt.Errorf("failed to add skill to SQLite search database: %w", err)
	}

	logger.Info("Added skill to SQLite search database", "agent_id", agentID, "SQLSkill_ID", SQLSkill.ID)

	return SQLSkill.ID, nil
}
