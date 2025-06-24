// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:mnd
package searchv1alpha2

var ValidQueryTypes []string

func init() {
	// Override allowed names for RecordQueryType
	RecordQueryType_name = map[int32]string{
		0:  "unspecified",
		1:  "name",
		2:  "version",
		3:  "skill-id",
		4:  "skill-name",
		5:  "locator",
		6:  "locator-type",
		7:  "locator-url",
		8:  "extension",
		9:  "extension-name",
		10: "extension-version",
	}
	RecordQueryType_value = map[string]int32{
		"":                  0,
		"unspecified":       0,
		"name":              1,
		"version":           2,
		"skill-id":          3,
		"skill-name":        4,
		"locator":           5,
		"locator-type":      6,
		"locator-url":       7,
		"extension":         8,
		"extension-name":    9,
		"extension-version": 10,
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
