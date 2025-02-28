// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package corev1alpha1

import "fmt"

func (skill *Skill) FQDN() string {
	return fmt.Sprintf("%s/%s", skill.GetCategoryName(), skill.GetClassName())
}
