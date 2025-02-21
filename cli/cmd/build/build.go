// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package build

import (
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/types/known/timestamppb"

	apicore "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/cli/builder"
	"github.com/agntcy/dir/cli/cmd/build/config"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "build",
	Short: "Build agent model to prepare for pushing",
	Long: `Usage example:

	dirctl build \
		--name="agent-name" \
		--version="v1.0.0" \
		--artifact="docker-image:http://ghcr.io/example/example" \
		--artifact="python-package:http://ghcr.io/example/example" \
		--author="author1" \
		--author="author2" \
		--category="category1" \
		--category="category2" \
		./path-to-agent

`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("arg missing: must provide path to agent")
		}
		return runCommand(cmd, args[0])
	},
}

func runCommand(cmd *cobra.Command, agentPath string) error {
	// Get configuration from flags
	buildConfig := &config.Config{}
	err := buildConfig.LoadFromFlags(opts.Name, opts.Version, opts.CreatedAt, opts.LLMAnalyzer, opts.Authors, opts.Categories, opts.Artifacts)
	if err != nil {
		return fmt.Errorf("failed to load config from flags: %w", err)
	}

	// Get configuration from file
	if opts.ConfigFile != "" {
		fileConfig := &config.Config{}
		err := fileConfig.LoadFromFile(opts.ConfigFile)
		if err != nil {
			return fmt.Errorf("failed to load config file: %w", err)
		}

		// Merge file config with flags config
		// Flags should override file config
		buildConfig.Merge(fileConfig)
	}

	// Set source to agent path
	buildConfig.Source = agentPath

	locators, err := buildConfig.GetAPILocators()
	if err != nil {
		return fmt.Errorf("failed to get locators from config: %w", err)
	}

	// Build to obtain agent model
	extensions, err := builder.Build(cmd.Context(), buildConfig)
	if err != nil {
		return fmt.Errorf("failed to build agent: %w", err)
	}

	// Create agent data model
	agent := &apicore.Agent{
		Name:       buildConfig.Name,
		Version:    buildConfig.Version,
		Authors:    buildConfig.Authors,
		CreatedAt:  timestamppb.New(buildConfig.CreatedAt),
		Locators:   locators,
		Extensions: extensions,
	}

	// Construct output
	agentRaw, err := json.MarshalIndent(agent, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal built data: %w", err)
	}

	// Print to output
	cmd.Print(string(agentRaw))

	return nil
}
