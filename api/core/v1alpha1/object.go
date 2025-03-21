// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package corev1alpha1

func init() {
	// Override allowed names for object types
	ObjectType_name = map[int32]string{
		0: "raw",
		1: "agent",
	}
	ObjectType_value = map[string]int32{
		"":      0,
		"raw":   0,
		"agent": 1,
	}
}
