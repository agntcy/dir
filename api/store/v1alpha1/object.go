// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package storev1alpha1

func init() {
	// Override allowed names for object types
	ObjectType_name = map[int32]string{
		0: "unspecified",
		1: "agent",
		2: "extension",
	}
	ObjectType_value = map[string]int32{
		"":            0,
		"unspecified": 0,
		"agent":       1,
		"extension":   2,
	}
}
