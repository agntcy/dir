// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:mnd
package searchv1alpha2

const ValidQueryTypes = "agent-name,agent-version,skill-id,skill-name,locator,extension"

func init() {
	// Override allowed names for RecordQueryType
	RecordQueryType_name = map[int32]string{
		0: "unspecified",
		1: "agent-name",
		2: "agent-version",
		3: "skill-id",
		4: "skill-name",
		5: "locator",
		7: "extension",
	}
	RecordQueryType_value = map[string]int32{
		"":              0,
		"unspecified":   0,
		"agent-name":    1,
		"agent-version": 2,
		"skill-id":      3,
		"skill-name":    4,
		"locator":       5,
		"extension":     6,
	}
}
