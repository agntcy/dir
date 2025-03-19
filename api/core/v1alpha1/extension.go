// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package corev1alpha1

import "fmt"

func (e *Extension) Key() string {
	return fmt.Sprintf("%s/%s", e.GetName(), e.GetVersion())
}
