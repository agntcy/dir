// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package build

var opts = &options{}

type options struct {
	Name        string
	Version     string
	CreatedAt   string
	LLMAnalyzer bool
	Authors     []string
	Categories  []string
	Artifacts   []string
	ConfigFile  string
}

func init() {
	flags := Command.Flags()
	flags.StringVar(&opts.Name, "name", "", "Name of the agent")
	flags.StringVar(&opts.Version, "version", "", "Version of the agent")

	// LLM Analyzer extension
	flags.BoolVarP(&opts.LLMAnalyzer, "llmanalyzer", "l", false, "Enable LLMAnalyzer extension")

	flags.StringSliceVar(
		&opts.Authors,
		"author",
		[]string{},
		"Authors to set for the agent. Overrides builder defaults. Example usage: --author author1 --author author2",
	)

	// Category extension
	flags.StringSliceVar(
		&opts.Categories,
		"category",
		[]string{},
		"Categories to set for the agent. Overrides builder defaults. Example usage: --category category1 --category category2",
	)

	// Creation time (only for dev purposes)
	flags.StringVar(&opts.CreatedAt, "created-at", "", "Agent creation time in RFC3339 format")
	_ = flags.MarkHidden("created-at")

	// Artifacts
	flags.StringSliceVar(
		&opts.Artifacts,
		"artifact",
		[]string{},
		"Artifacts to set for the agent. Each artifact should be in the format 'type:url'. Example usage: --artifact type1:url1 --artifact type2:url2. Supported types: 'docker-image', 'python-package', 'helm-chart', 'source-code' and 'binary'.",
	)

	flags.StringVarP(&opts.ConfigFile, "config-file", "c", "", "Path to the agent build configuration file. Please note that other flags will override the build configuration from the file. Supported formats: YAML")
}
