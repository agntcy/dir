// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

//nolint:testifylint
package crewai

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	expectedKey   = "agent.communicator.backstory"
	expectedValue = "You are skilled at creating effective outreach strategies and templates to engage candidates. Your communication tactics ensure high response rates from potential candidates.\n"

	inputKey   = "inputs.research_candidates_task"
	inputValue = "string"
)

func TestBuilder(t *testing.T) {
	builder := New("./testdata", []string{})

	// build
	gotExtensions, err := builder.Build(t.Context())
	assert.NoError(t, err)

	// validate
	data, ok := gotExtensions[0].Data.(map[string]string)
	assert.True(t, ok)

	data[expectedKey] = expectedValue
	data[inputKey] = inputValue
}
