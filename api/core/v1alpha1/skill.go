// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package corev1alpha1

import (
	"fmt"

	objectsv1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/objects/v1"
)

type Skill struct {
	*objectsv1.Skill
}

func (s *Skill) Key() string {
	if s.GetClassName() == "" {
		return s.GetCategoryName()
	}

	return fmt.Sprintf("%s/%s", s.GetCategoryName(), s.GetClassName())
}

func (s *Skill) ToOASFSchema() *objectsv1.Skill {
	if s == nil || s.Skill == nil {
		return nil
	}

	return s.Skill
}

func (s *Skill) GetName() string {
	if s.GetClassName() == "" {
		return s.GetCategoryName()
	}

	return fmt.Sprintf("%s/%s", s.GetCategoryName(), s.GetClassName())
}

func (s *Skill) GetID() uint64 {
	return s.GetClassUid()
}
