// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package llmanalyzer

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

const (
	DefaultEnvPrefix = "LLM_ANALYZER"
)

type Config struct {
	AzureOpenAIAPIKey         string `json:"azure_openai_api_key,omitempty"         mapstructure:"azure_openai_api_key"`
	AzureOpenAIAPIEndpoint    string `json:"azure_openai_api_endpoint,omitempty"    mapstructure:"azure_openai_api_endpoint"`
	AzureOpenAIDeploymentName string `json:"azure_openai_deployment_name,omitempty" mapstructure:"azure_openai_deployment_name"`
}

func NewConfig() (*Config, error) {
	v := viper.NewWithOptions(
		viper.KeyDelimiter("."),
		viper.EnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_")),
	)

	v.SetEnvPrefix(DefaultEnvPrefix)
	v.AllowEmptyEnv(true)
	v.AutomaticEnv()

	_ = v.BindEnv("azure_openai_api_key")
	_ = v.BindEnv("azure_openai_api_endpoint")
	_ = v.BindEnv("azure_openai_deployment_name")

	decodeHooks := mapstructure.ComposeDecodeHookFunc(
		mapstructure.TextUnmarshallerHookFunc(),
		mapstructure.StringToTimeDurationHookFunc(),
		mapstructure.StringToSliceHookFunc(","),
	)

	config := &Config{}
	if err := v.Unmarshal(config, viper.DecodeHook(decodeHooks)); err != nil {
		return nil, fmt.Errorf("failed to load LLMAnalyzer configuration: %w", err)
	}

	return config, nil
}

func (c *Config) Validate() error {
	if c.AzureOpenAIAPIKey == "" {
		return errors.New("azure OpenAI API key is required")
	}

	if c.AzureOpenAIAPIEndpoint == "" {
		return errors.New("azure OpenAI API endpoint is required")
	}

	if c.AzureOpenAIDeploymentName == "" {
		return errors.New("azure OpenAI deployment name is required")
	}

	return nil
}
