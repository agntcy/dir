// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

// Test ToAPIExtension method
package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAgentExtension_ToAPIExtension(t *testing.T) {
	type ExtensionSpecs struct {
		TestString string            `json:"test_string,omitempty"`
		TestSlice  []string          `json:"test_slice,omitempty"`
		TestMap    map[string]string `json:"test_map,omitempty"`
	}

	agentExtension := AgentExtension{
		Name:    "base",
		Version: "v0.0.0",
		Specs: ExtensionSpecs{
			TestString: "test",
			TestSlice:  []string{"test1", "test2"},
			TestMap: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		},
	}

	apiExtension, err := agentExtension.ToAPIExtension()
	assert.Nil(t, err)
	assert.Equal(t, apiExtension.Name, agentExtension.Name)
	assert.Equal(t, apiExtension.Version, agentExtension.Version)
	assert.Equal(t, apiExtension.Specs, map[string]string{
		"test_string": "test",
		"test_slice":  "[test1, test2]",
		"test_map":    "{\"key1\":\"value1\",\"key2\":\"value2\"}",
	})
}
