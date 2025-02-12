// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package crewai

import (
	"context"
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
	builder := New("./testdata")

	// build
	gotExtension, err := builder.Build(context.Background())
	assert.Nil(t, err)

	// validate
	specs, ok := gotExtension.Specs.(map[string]string)
	assert.True(t, ok)
	specs[expectedKey] = expectedValue
	specs[inputKey] = inputValue
}
