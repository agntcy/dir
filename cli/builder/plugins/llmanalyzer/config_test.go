// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:testifylint
package llmanalyzer

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewConfig(t *testing.T) {
	tests := []struct {
		name       string
		envVars    map[string]string
		wantConfig *Config
		wantError  error
	}{
		{
			name:    "Default values",
			envVars: map[string]string{},
			wantConfig: &Config{
				AzureOpenAIAPIKey:         "",
				AzureOpenAIAPIEndpoint:    "",
				AzureOpenAIDeploymentName: "",
			},
			wantError: nil,
		},
		{
			name: "Custom values",
			envVars: map[string]string{
				"LLM_ANALYZER_AZURE_OPENAI_API_KEY":         "test-api-key",
				"LLM_ANALYZER_AZURE_OPENAI_API_ENDPOINT":    "https://api.openai.com",
				"LLM_ANALYZER_AZURE_OPENAI_DEPLOYMENT_NAME": "test-deployment",
			},
			wantConfig: &Config{
				AzureOpenAIAPIKey:         "test-api-key",
				AzureOpenAIAPIEndpoint:    "https://api.openai.com",
				AzureOpenAIDeploymentName: "test-deployment",
			},
			wantError: nil,
		},
	}

	for _, tt := range tests {
		ttp := tt
		t.Run(ttp.name, func(t *testing.T) {
			for envKey, envVal := range ttp.envVars {
				t.Setenv(envKey, envVal)
			}

			t.Cleanup(func() {
				os.Clearenv()
			})

			config, err := NewConfig()
			if err != nil {
				assert.EqualError(t, ttp.wantError, err.Error(), "Unexpected error message")
			} else {
				assert.NoError(t, ttp.wantError, "Expected no error but got one")
			}

			if ttp.wantConfig != nil {
				assert.Equal(t, ttp.wantConfig, config, "Unexpected config")
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		wantError error
	}{
		{
			name: "Valid config",
			config: &Config{
				AzureOpenAIAPIKey:         "test-api-key",
				AzureOpenAIAPIEndpoint:    "https://api.openai.com",
				AzureOpenAIDeploymentName: "test-deployment",
			},
			wantError: nil,
		},
		{
			name: "Missing API Key",
			config: &Config{
				AzureOpenAIAPIKey:         "",
				AzureOpenAIAPIEndpoint:    "https://api.openai.com",
				AzureOpenAIDeploymentName: "test-deployment",
			},
			wantError: errors.New("azure OpenAI API key is required"),
		},
		{
			name: "Missing API Endpoint",
			config: &Config{
				AzureOpenAIAPIKey:         "test-api-key",
				AzureOpenAIAPIEndpoint:    "",
				AzureOpenAIDeploymentName: "test-deployment",
			},
			wantError: errors.New("azure OpenAI API endpoint is required"),
		},
		{
			name: "Missing Deployment Name",
			config: &Config{
				AzureOpenAIAPIKey:         "test-api-key",
				AzureOpenAIAPIEndpoint:    "https://api.openai.com",
				AzureOpenAIDeploymentName: "",
			},
			wantError: errors.New("azure OpenAI deployment name is required"),
		},
	}

	for _, tt := range tests {
		ttp := tt
		t.Run(ttp.name, func(t *testing.T) {
			err := ttp.config.Validate()
			if err != nil {
				assert.EqualError(t, err, ttp.wantError.Error(), "Unexpected error message")
			} else {
				assert.NoError(t, ttp.wantError, "Expected an error but got nil")
			}
		})
	}
}
