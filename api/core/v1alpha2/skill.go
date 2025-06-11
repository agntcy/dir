// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package corev1alpha2

import (
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
)

type SQLiteSkill struct {
	gorm.Model
	AgentID     uint   `gorm:"not null;index"`
	SkillID     uint32 `gorm:"not null"`
	Name        string `gorm:"not null"`
	Annotations string `gorm:"not null"`
}

func (s *Skill) ToSQLiteSkill(agentID uint) (*SQLiteSkill, error) {
	if s == nil {
		return nil, nil
	}

	skill := &SQLiteSkill{
		AgentID: agentID,
		SkillID: s.Id,
		Name:    s.Name,
	}

	if s.Annotations != nil {
		annotations, err := json.Marshal(s.Annotations)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal annotations: %w", err)
		}

		skill.Annotations = string(annotations)
	}

	return skill, nil
}
