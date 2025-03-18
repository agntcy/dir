// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var invalidQueries = []string{
	"",
	"/",
	"/agents",
	"/agents/agentX",
	"/skills/",
	"/locators",
	"/extensions",
	"skills/",
	"locators/",
	"extensions/",
}

// nolint:testifylint
func TestList_InvalidQuery(t *testing.T) {
	r := &routing{ds: nil}

	for _, q := range invalidQueries {
		t.Run("Invalid query: "+q, func(t *testing.T) {
			_, err := r.List(t.Context(), q)
			assert.Error(t, err)
			assert.Equal(t, "invalid query: "+q, err.Error())
		})
	}
}

// TODO Test valid queries once publish is implemented
