// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v2"
)

type Builder struct {
	BaseModelPath  string   `yaml:"base-model"`
	SourceIgnore   []string `yaml:"source-ignore"`
	LLMAnalyzer    bool     `yaml:"llmanalyzer"`
	Runtime        bool     `yaml:"runtime"`
	OASFValidation bool     `yaml:"oasf-validation"`
}

type Config struct {
	Builder Builder `yaml:"builder"`
}

func (c *Config) LoadFromFile(path string) error {
	reader, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read data: %w", err)
	}

	err = yaml.Unmarshal(data, c)
	if err != nil {
		return fmt.Errorf("failed to unmarshal data: %w", err)
	}

	return nil
}
