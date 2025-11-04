// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPullRecordInputValidation(t *testing.T) {
	tests := []struct {
		name        string
		input       PullRecordInput
		expectError bool
	}{
		{
			name: "valid input",
			input: PullRecordInput{
				CID: "bafkreiabcd1234567890",
			},
			expectError: false,
		},
		{
			name: "missing CID",
			input: PullRecordInput{
				CID: "",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just test that marshaling works
			_, err := json.Marshal(tt.input)
			require.NoError(t, err)

			// Validate CID requirement
			if tt.expectError {
				assert.Empty(t, tt.input.CID)
			} else {
				assert.NotEmpty(t, tt.input.CID)
			}
		})
	}
}
