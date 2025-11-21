// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:dupl // Intentional duplication with skills test for separate domain/skill testing
package tools

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSchemaDomains(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name          string
		input         GetSchemaDomainsInput
		expectError   bool
		expectDomains bool
		checkCallback func(t *testing.T, output GetSchemaDomainsOutput)
	}{
		{
			name: "Get top-level domains for version 0.7.0",
			input: GetSchemaDomainsInput{
				Version: "0.7.0",
			},
			expectError:   false,
			expectDomains: true,
			checkCallback: func(t *testing.T, output GetSchemaDomainsOutput) {
				t.Helper()

				assert.Equal(t, "0.7.0", output.Version)
				assert.Empty(t, output.ErrorMessage)
				assert.NotEmpty(t, output.Domains)

				// Check that top-level domains have expected fields
				for _, domain := range output.Domains {
					assert.NotEmpty(t, domain.Name, "Each domain should have a name")
				}
			},
		},
		{
			name: "Get sub-domains for a parent domain",
			input: GetSchemaDomainsInput{
				Version:      "0.7.0",
				ParentDomain: "technology",
			},
			expectError:   false,
			expectDomains: true,
			checkCallback: func(t *testing.T, output GetSchemaDomainsOutput) {
				t.Helper()

				assert.Equal(t, "0.7.0", output.Version)
				assert.Equal(t, "technology", output.ParentDomain)
				assert.Empty(t, output.ErrorMessage)
				assert.NotEmpty(t, output.Domains)

				// All returned domains should be sub-domains
				for _, domain := range output.Domains {
					assert.NotEmpty(t, domain.Name, "Each sub-domain should have a name")
				}
			},
		},
		{
			name: "Invalid version",
			input: GetSchemaDomainsInput{
				Version: "99.99.99",
			},
			expectError:   false,
			expectDomains: false,
			checkCallback: func(t *testing.T, output GetSchemaDomainsOutput) {
				t.Helper()

				assert.NotEmpty(t, output.ErrorMessage)
				assert.Contains(t, output.ErrorMessage, "invalid version")
				assert.NotEmpty(t, output.AvailableVersions)
			},
		},
		{
			name: "Missing version parameter",
			input: GetSchemaDomainsInput{
				Version: "",
			},
			expectError:   false,
			expectDomains: false,
			checkCallback: func(t *testing.T, output GetSchemaDomainsOutput) {
				t.Helper()

				assert.NotEmpty(t, output.ErrorMessage)
				assert.Contains(t, output.ErrorMessage, "version parameter is required")
				assert.NotEmpty(t, output.AvailableVersions)
			},
		},
		{
			name: "Non-existent parent domain",
			input: GetSchemaDomainsInput{
				Version:      "0.7.0",
				ParentDomain: "non_existent_domain",
			},
			expectError:   false,
			expectDomains: false,
			checkCallback: func(t *testing.T, output GetSchemaDomainsOutput) {
				t.Helper()

				assert.NotEmpty(t, output.ErrorMessage)
				assert.Contains(t, output.ErrorMessage, "not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, output, err := GetSchemaDomains(ctx, nil, tt.input)

			if tt.expectError {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Nil(t, result) // Tool handlers typically return nil for result

			if tt.expectDomains {
				assert.NotEmpty(t, output.Domains, "Expected to receive domains")
			}

			if tt.checkCallback != nil {
				tt.checkCallback(t, output)
			}
		})
	}
}

func TestParseItemFromSchemaForDomains(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		defMap   map[string]interface{}
		expected schemaClass
	}{
		{
			name: "Parse domain with name, caption (title), and ID",
			defMap: map[string]interface{}{
				"title": "Test Domain Caption",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"const": "test_domain",
					},
					"id": map[string]interface{}{
						"const": float64(123),
					},
				},
			},
			expected: schemaClass{
				Name:    "test_domain",
				Caption: "Test Domain Caption",
				ID:      123,
			},
		},
		{
			name: "Parse domain with missing fields",
			defMap: map[string]interface{}{
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"const": "minimal_domain",
					},
				},
			},
			expected: schemaClass{
				Name: "minimal_domain",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := parseItemFromSchema(tt.defMap)
			assert.Equal(t, tt.expected.Name, result.Name)
			assert.Equal(t, tt.expected.Caption, result.Caption)
			assert.Equal(t, tt.expected.ID, result.ID)
		})
	}
}
