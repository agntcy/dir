// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sqlite

import (
	"fmt"

	"github.com/agntcy/dir/server/types"
	"gorm.io/gorm"
)

type Skill struct {
	gorm.Model
	AgentID uint   `gorm:"not null;index"`
	SkillID uint64 `gorm:"not null"`
	Name    string `gorm:"not null"`
}

func (skill *Skill) GetAnnotations() map[string]string {
	// SQLite skills don't store annotations, return empty map
	return make(map[string]string)
}

func (skill *Skill) GetID() uint64 {
	return skill.SkillID
}

func (skill *Skill) GetName() string {
	return skill.Name
}

func (d *DB) addSkillTx(tx *gorm.DB, skill types.Skill, agentID uint) (uint, error) {
	sqliteSkill := &Skill{
		AgentID: agentID,
		SkillID: skill.GetID(),
		Name:    skill.GetName(),
	}

	if err := tx.Create(sqliteSkill).Error; err != nil {
		return 0, fmt.Errorf("failed to add skill to SQLite database: %w", err)
	}

	logger.Debug("Added skill to SQLite database", "agent_id", agentID, "skill_id", sqliteSkill.ID)

	return sqliteSkill.ID, nil
}
