// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v2"

	apicore "github.com/agntcy/dir/api/core/v1alpha1"
)

type Locator struct {
	Type string `yaml:"type"`
	URL  string `yaml:"url"`
}

type Extension struct {
	Name    string         `yaml:"name"`
	Version string         `yaml:"version"`
	Specs   map[string]any `yaml:"specs"`
}

type Model struct {
	Name       string      `yaml:"name"`
	Version    string      `yaml:"version"`
	Authors    []string    `yaml:"authors"`
	Locators   []Locator   `yaml:"locators"`
	Skills     []string    `yaml:"skills"`
	Extensions []Extension `yaml:"extensions"`
}

type Builder struct {
	Source       string   `yaml:"source"`
	SourceIgnore []string `yaml:"source-ignore"`
	LLMAnalyzer  bool     `yaml:"llmanalyzer"`
	CrewAI       bool     `yaml:"crewai"`
	Runtime      bool     `yaml:"runtime"`
}

type Config struct {
	Model   Model   `yaml:"model"`
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

func (c *Config) GetAPILocators() ([]*apicore.Locator, error) {
	var locators []*apicore.Locator
	for _, locator := range c.Model.Locators {
		var ok bool
		var locatorType int32
		if locatorType, ok = apicore.LocatorType_value[locator.Type]; !ok {
			return nil, fmt.Errorf("invalid locator type: %s", locator.Type)
		}

		locators = append(locators, &apicore.Locator{
			Name: apicore.LocatorType_name[locatorType],
			Source: &apicore.LocatorSource{
				Url: locator.URL,
			},
		})
	}

	return locators, nil
}
