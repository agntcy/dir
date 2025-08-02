// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sqlite

import (
	"time"

	"github.com/agntcy/dir/server/types"
)

type Skill struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	RecordCID string `gorm:"column:record_cid;not null;index"`
	SkillID   uint64 `gorm:"not null"`
	Name      string `gorm:"not null"`
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

// convertSkills transforms interface types to SQLite structs.
func convertSkills(skills []types.Skill, recordCID string) []Skill {
	result := make([]Skill, len(skills))
	for i, skill := range skills {
		result[i] = Skill{
			RecordCID: recordCID,
			SkillID:   skill.GetID(),
			Name:      skill.GetName(),
		}
	}

	return result
}
