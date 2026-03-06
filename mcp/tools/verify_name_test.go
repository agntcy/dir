// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifyNameInputValidation(t *testing.T) {
	tests := []struct {
		name        string
		input       VerifyNameInput
		expectError bool
		description string
	}{
		{
			name: "valid input with CID",
			input: VerifyNameInput{
				CID: "bafkreiabcd1234567890",
			},
			expectError: false,
			description: "CID alone should be valid",
		},
		{
			name: "valid input with name",
			input: VerifyNameInput{
				Name: "example.com/my-agent",
			},
			expectError: false,
			description: "Name alone should be valid",
		},
		{
			name: "valid input with name and version",
			input: VerifyNameInput{
				Name:    "example.com/my-agent",
				Version: "v1.0.0",
			},
			expectError: false,
			description: "Name with version should be valid",
		},
		{
			name:        "missing both CID and name",
			input:       VerifyNameInput{},
			expectError: true,
			description: "Either CID or name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that marshaling works
			data, err := json.Marshal(tt.input)
			require.NoError(t, err)

			// Test that unmarshaling works
			var decoded VerifyNameInput

			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)

			// Validate that either CID or Name is present
			hasInput := decoded.CID != "" || decoded.Name != ""
			if tt.expectError {
				assert.False(t, hasInput, tt.description)
			} else {
				assert.True(t, hasInput, tt.description)
			}
		})
	}
}

func TestVerifyNameOutputSerialization(t *testing.T) {
	tests := []struct {
		name   string
		output VerifyNameOutput
	}{
		{
			name: "verified output",
			output: VerifyNameOutput{
				Verified: true,
				DomainVerification: &DomainVerificationDetail{
					Domain:       "example.com",
					Method:       "wellknown",
					MatchedKeyID: "key-123",
					VerifiedAt:   "2025-01-21T12:00:00Z",
				},
			},
		},
		{
			name: "not verified output",
			output: VerifyNameOutput{
				Verified:     false,
				ErrorMessage: "no valid signature found for domain",
			},
		},
		{
			name: "error output",
			output: VerifyNameOutput{
				ErrorMessage: "Failed to connect to server",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that marshaling works
			data, err := json.Marshal(tt.output)
			require.NoError(t, err)

			// Test that unmarshaling works
			var decoded VerifyNameOutput

			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)

			// Verify fields are preserved
			assert.Equal(t, tt.output.Verified, decoded.Verified)
			assert.Equal(t, tt.output.ErrorMessage, decoded.ErrorMessage)

			if tt.output.DomainVerification != nil {
				require.NotNil(t, decoded.DomainVerification)
				assert.Equal(t, tt.output.DomainVerification.Domain, decoded.DomainVerification.Domain)
				assert.Equal(t, tt.output.DomainVerification.Method, decoded.DomainVerification.Method)
				assert.Equal(t, tt.output.DomainVerification.MatchedKeyID, decoded.DomainVerification.MatchedKeyID)
			}
		})
	}
}
