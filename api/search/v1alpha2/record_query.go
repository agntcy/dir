// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:mnd
package searchv1alpha2

func init() {
	// Override allowed names for RecordQueryType
	RecordQueryType_name = map[int32]string{
		0: "unspecified",
		1: "agent-name",
		2: "agent-version",
		3: "skill-id",
		4: "skill-name",
		5: "locator-type",
		6: "locator-url",
		7: "extension-name",
		8: "extension-version",
	}
	RecordQueryType_value = map[string]int32{
		"":                  0,
		"unspecified":       0,
		"agent-name":        1,
		"agent-version":     2,
		"skill-id":          3,
		"skill-name":        4,
		"locator-type":      5,
		"locator-url":       6,
		"extension-name":    7,
		"extension-version": 8,
	}
}
