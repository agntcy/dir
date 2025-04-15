// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package pyprojectparser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// nolint:testifylint
func TestPyProjectParser(t *testing.T) {
	// Define the test data directory
	sources := []string{
		"./testdata/project",
		"./testdata/poetry",
	}

	for _, source := range sources {
		// Create a new instance of the pyprojectparser plugin
		parser := New(source)
		assert.NotNil(t, parser, "Parser instance should not be nil")

		// Call the Build method
		agent, err := parser.Build(t.Context())
		assert.NoError(t, err, "Build should not return an error")
		assert.NotNil(t, agent, "Agent should not be nil")

		// Validate the extracted metadata
		assert.Equal(t, "example-project", agent.GetName(), "Agent name should match")
		assert.Equal(t, "1.0.0", agent.GetVersion(), "Agent version should match")
		assert.Equal(t, "An example project for testing.", agent.GetDescription(), "Agent description should match")
		assert.ElementsMatch(t, []string{"Author One", "Author Two"}, agent.GetAuthors(), "Agent authors should match")
	}
}
