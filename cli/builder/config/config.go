// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package config

import "fmt"

type FrameworkType string

const (
	crewAI    FrameworkType = "crewai"
	autogen   FrameworkType = "autogen"
	llmaIndex FrameworkType = "llma-index"
	langchain FrameworkType = "langchain"
)

type Framework struct {
	Type    string `yaml:"type"`
	Version string `yaml:"version"`
}

type Language struct {
	Type    string `yaml:"type"`
	Version string `yaml:"version"`
}

type Config struct {
	Source       string   `yaml:"source"`
	SourceIgnore []string `yaml:"source-ignore"`

	LLMAnalyzer bool `yaml:"llmanalyzer"`
	CrewAI      bool `yaml:"crewai"`

	Framework Framework `yaml:"framework"`
	Language  Language  `yaml:"language"`
	Skills    []string  `yaml:"skills"`
}

func (c *Config) Validate() error {
	switch FrameworkType(c.Framework.Type) {
	case crewAI, autogen, llmaIndex, langchain:
	default:
		return fmt.Errorf("invalid framework type: %s", c.Framework.Type)
	}

	return nil
}
