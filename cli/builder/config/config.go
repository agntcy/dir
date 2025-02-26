// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package config

type Config struct {
	Source       string   `yaml:"source"`
	SourceIgnore []string `yaml:"source-ignore"`

	LLMAnalyzer bool `yaml:"llmanalyzer"`
	CrewAI      bool `yaml:"crewai"`
}
