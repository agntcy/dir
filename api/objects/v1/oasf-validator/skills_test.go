// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// nolint:testifylint,goconst
package oasfvalidator

import (
	"sync"
	"testing"

	objectsv1 "github.com/agntcy/dir/api/objects/v1"
	"github.com/stretchr/testify/assert"
)

func TestSkillValidator_HasSkill(t *testing.T) {
	validator := &SkillValidator{
		skills: map[uint64]*objectsv1.Skill{
			1: {ClassUid: 1},
		},
		mu: sync.RWMutex{},
	}

	assert.True(t, validator.HasSkill(1))
	assert.False(t, validator.HasSkill(2))
}

func TestSkillValidator_GetSkill(t *testing.T) {
	validator := &SkillValidator{
		skills: map[uint64]*objectsv1.Skill{
			1: {ClassUid: 1},
		},
		mu: sync.RWMutex{},
	}

	assert.NotNil(t, validator.GetSkill(1))
	assert.Nil(t, validator.GetSkill(2))
}

func TestSkillValidator_GetSkillByName(t *testing.T) {
	className := "Skill1"
	validator := &SkillValidator{
		skills: map[uint64]*objectsv1.Skill{
			1: {ClassUid: 1, ClassName: &className},
		},
		mu: sync.RWMutex{},
	}

	assert.NotNil(t, validator.GetSkillByName("Skill1"))
	assert.Nil(t, validator.GetSkillByName("Skill2"))
}

func TestSkillValidator_Validate(t *testing.T) {
	className := "Skill1"
	validator := &SkillValidator{
		skills: map[uint64]*objectsv1.Skill{
			1: {ClassUid: 1, ClassName: &className},
		},
		mu: sync.RWMutex{},
	}

	t.Run("Valid skills", func(t *testing.T) {
		skills := []*objectsv1.Skill{
			{ClassUid: 1},
		}

		err := validator.Validate(skills)
		assert.NoError(t, err)
	})

	t.Run("Invalid skills", func(t *testing.T) {
		skills := []*objectsv1.Skill{
			{ClassUid: 2},
		}

		err := validator.Validate(skills)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "'class_uid' not found")
	})

	t.Run("Missing class_uid", func(t *testing.T) {
		skills := []*objectsv1.Skill{
			{},
		}

		err := validator.Validate(skills)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "'class_uid' is required")
	})
}

func TestValidateSkills(t *testing.T) {
	className := "Skill1"
	Validator.skills = map[uint64]*objectsv1.Skill{
		1: {ClassUid: 1, ClassName: &className},
	}

	t.Run("Valid skills", func(t *testing.T) {
		skills := []*objectsv1.Skill{
			{ClassUid: 1},
		}

		err := ValidateSkills(skills)
		assert.NoError(t, err)
	})

	t.Run("Invalid skills", func(t *testing.T) {
		skills := []*objectsv1.Skill{
			{ClassUid: 2},
		}

		err := ValidateSkills(skills)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "'class_uid' not found")
	})
}
