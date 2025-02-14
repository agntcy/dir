// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package corev1alpha1

func init() {
	// Override allowed names for locator types
	LocatorType_name = map[int32]string{
		0: "unspecified",
		1: "helm-chart",
		2: "docker-image",
		3: "python-package",
	}
	LocatorType_value = map[string]int32{
		"unspecified":    0,
		"helm-chart":     1,
		"docker-image":   2,
		"python-package": 3,
	}
}
