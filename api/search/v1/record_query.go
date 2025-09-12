// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:mnd
package v1

var ValidQueryTypes []string

func init() {
	// Override allowed names for RecordQueryType
	RecordQueryType_name = map[int32]string{
		0: "unspecified",
		1: "name",
		2: "version",
		3: "skill-id",
		4: "skill-name",
		5: "locator",
		6: "extension",
	}
	RecordQueryType_value = map[string]int32{
		"":            0,
		"unspecified": 0,
		"name":        1,
		"version":     2,
		"skill-id":    3,
		"skill-name":  4,
		"locator":     5,
		"extension":   6,
	}

	ValidQueryTypes = []string{
		"name",
		"version",
		"skill-id",
		"skill-name",
		"locator",
		"extension",
	}
}
