// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package objectsv1

import "fmt"

func (skill *Skill) Key() string {
	if skill.GetClassName() == "" {
		return skill.GetCategoryName()
	}

	return fmt.Sprintf("%s/%s", skill.GetCategoryName(), skill.GetClassName())
}

func (skill *Skill) GetName() string {
	if skill.GetClassName() == "" {
		return skill.GetCategoryName()
	}

	return fmt.Sprintf("%s/%s", skill.GetCategoryName(), skill.GetClassName())
}

func (skill *Skill) GetID() uint64 {
	return skill.GetClassUid()
}
