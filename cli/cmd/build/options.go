// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package build

var opts = &options{}

type options struct {
	Name        string
	Version     string
	LLMAnalyzer bool
	CrewAI      bool
	Authors     []string
	Locators    []string
	ConfigFile  string
}

func init() {
	flags := Command.Flags()
	flags.StringVar(&opts.Name, "name", "", "Name of the agent")
	flags.StringVar(&opts.Version, "version", "", "Version of the agent")

	// LLM Analyzer extension
	flags.BoolVarP(&opts.LLMAnalyzer, "llmanalyzer", "l", false, "Enable LLMAnalyzer extension")

	// CrewAI extension
	flags.BoolVarP(&opts.CrewAI, "crewai", "c", false, "Enable CrewAI extension")

	flags.StringSliceVar(
		&opts.Authors,
		"author",
		[]string{},
		"Authors to set for the agent. Overrides builder defaults. Example usage: --author author1 --author author2",
	)

	// Locators
	flags.StringSliceVar(
		&opts.Locators,
		"locator",
		[]string{},
		"Artifact locators to set for the agent. Each locator should be in the format 'type:url'. Example usage: --locator type1:url1 --locator type2:url2. Supported types: 'docker-image', 'python-package', 'helm-chart', 'source-code' and 'binary'.",
	)

	flags.StringVarP(&opts.ConfigFile, "config-file", "f", "", "Path to the agent build configuration file. Please note that other flags will override the build configuration from the file. Supported formats: YAML")
}
