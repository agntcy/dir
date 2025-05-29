// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sqlite

import (
	"fmt"
	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	searchtypes "github.com/agntcy/dir/server/search/types"
	"gorm.io/gorm"
)

func (s *SQLiteDB) AddSkillTx(tx *gorm.DB, coreSkill *coretypes.Skill, agentID uint) (uint, error) {
	skill := &searchtypes.Skill{}
	if err := skill.FromCoreSkill(coreSkill, agentID); err != nil {
		return 0, fmt.Errorf("failed to convert core skill to search skill: %w", err)
	}

	if err := tx.Create(skill).Error; err != nil {
		return 0, fmt.Errorf("failed to add skill to SQLite search database: %w", err)
	}

	logger.Info("Added skill to SQLite search database", "category name", skill.CategoryName, "class name", skill.ClassName, "agentID", agentID)

	return skill.ID, nil
}
