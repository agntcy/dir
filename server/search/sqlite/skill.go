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
	SkillID uint32 `gorm:"not null"`
	Name    string `gorm:"not null"`
}

func (skill *Skill) GetID() uint32 {
	return skill.SkillID
}

func (skill *Skill) GetName() string {
	return skill.Name
}

func (d *DB) addSkillTx(tx *gorm.DB, skillObject types.SkillObject, agentID uint) (uint, error) {
	skill := &Skill{
		AgentID: agentID,
		SkillID: skillObject.GetID(),
		Name:    skillObject.GetName(),
	}

	if err := tx.Create(skill).Error; err != nil {
		return 0, fmt.Errorf("failed to add skill to SQLite search database: %w", err)
	}

	logger.Debug("Added skill to SQLite search database", "agent_id", agentID, "skill_id", skill.ID)

	return skill.ID, nil
}
