// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package builder

type Config struct {
	Source       string   `yaml:"source"`
	SourceIgnore []string `yaml:"sourceignore"`

	LLMAnalyzer bool `yaml:"llmanalyzer"`
}
