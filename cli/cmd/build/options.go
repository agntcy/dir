// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package build

var opts = &options{}

type options struct {
	Name         string
	Version      string
	ArtifactUrl  string
	ArtifactType string
	CreatedAt    string
	LLMAnalyzer  bool

	// Base extension
	Authors []string

	// Category extension
	Categories []string
}

func init() {
	flags := Command.Flags()
	flags.StringVar(&opts.Name, "name", "", "Name of the agent")
	flags.StringVar(&opts.Version, "version", "", "Version of the agent")
	flags.StringVar(&opts.ArtifactUrl, "artifact-url", "", "Agent artifact URL")
	flags.StringVar(&opts.ArtifactType, "artifact-type", "", "Agent artifact type")
	flags.BoolVarP(&opts.LLMAnalyzer, "llmanalyzer", "l", false, "Enable LLMAnalyzer extension")

	// Base extension
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
}
