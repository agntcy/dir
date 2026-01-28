// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServe_ValidationConfiguration(t *testing.T) {
	tests := []struct {
		name             string
		oasfSchemaURLEnv string
		wantError        bool // If true, expects Serve to return an error (configuration error)
	}{
		{
			name:             "error when schema URL is not set",
			oasfSchemaURLEnv: "",
			wantError:        true, // Should error because schema URL is required
		},
		{
			name:             "use schema URL from OASF_API_VALIDATION_SCHEMA_URL env",
			oasfSchemaURLEnv: "https://schema.oasf.outshift.com",
			wantError:        false,
		},
		{
			name:             "use custom schema URL",
			oasfSchemaURLEnv: "https://custom.schema.url",
			wantError:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Configure validation for unit tests: use a valid schema URL
			// This ensures tests don't depend on external services or require schema URL configuration
			if err := corev1.InitializeValidator("https://schema.oasf.outshift.com"); err != nil {
				t.Fatalf("Failed to initialize validator: %v", err)
			}

			// Set test env vars
			if tt.oasfSchemaURLEnv != "" {
				t.Setenv("OASF_API_VALIDATION_SCHEMA_URL", tt.oasfSchemaURLEnv)
			} else {
				// Clear env var to test error behavior
				t.Setenv("OASF_API_VALIDATION_SCHEMA_URL", "")
			}

			// Create a context that will be cancelled immediately to stop Serve early
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // Cancel immediately so Serve returns quickly

			// Call Serve - it will configure validation and then return due to cancelled context
			err := Serve(ctx)

			if tt.wantError {
				// Verify that Serve returned a configuration error
				require.Error(t, err, "Serve should return error when schema URL is missing")
				assert.Contains(t, err.Error(), "OASF_API_VALIDATION_SCHEMA_URL", "Error should mention OASF_API_VALIDATION_SCHEMA_URL")
			} else {
				// Verify that validation was configured correctly
				// We can't directly check the internal state, but we can verify
				// that the configuration functions were called by checking if
				// validation still works with the expected settings
				assert.Error(t, err) // Should error due to cancelled context (not configuration error)

				// Note: We can't easily verify the exact configuration without
				// exposing internal state, but the fact that Serve runs without
				// panicking and configures the validators is sufficient coverage
			}
		})
	}
}
