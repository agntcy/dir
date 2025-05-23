// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:testifylint
package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAgentExtension_ToStruct(t *testing.T) {
	type ExtensionData struct {
		TestString string            `json:"test_string,omitempty"`
		TestSlice  []string          `json:"test_slice,omitempty"`
		TestMap    map[string]string `json:"test_map,omitempty"`
	}

	extension := ExtensionData{
		TestString: "test",
		TestSlice:  []string{"test1", "test2"},
		TestMap: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
	}

	extensionStruct, err := ToStruct(extension)
	assert.NoError(t, err)
	assert.Equal(t, map[string]interface{}{
		"test_string": "test",
		"test_slice":  []interface{}{"test1", "test2"},
		"test_map":    map[string]interface{}{"key1": "value1", "key2": "value2"},
	}, extensionStruct.AsMap())
}
