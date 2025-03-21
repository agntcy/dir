// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package corev1alpha1

import "fmt"

func (skill *Skill) Key() string {
	return fmt.Sprintf("%s/%s", skill.GetCategoryName(), skill.GetClassName())
}
