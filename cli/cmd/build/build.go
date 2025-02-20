// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package build

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

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
	// Override creation time if requested
	createdAt := time.Now()
	if opts.CreatedAt != "" {
		var err error
		createdAt, err = time.Parse(time.RFC3339, opts.CreatedAt)
		if err != nil {
			return fmt.Errorf("failed to parse create time: %w", err)
		}
	}

	// Load in artifacts
	var locators []*apicore.Locator
	for _, artifact := range opts.Artifacts {
		// Split artifact into type and URL
		parts := strings.SplitN(artifact, ":", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid artifact format, expected 'type:url'")
		}

		var ok bool
		var artifactType int32
		if artifactType, ok = apicore.LocatorType_value[parts[0]]; !ok {
			return fmt.Errorf("invalid artifact type: %s", parts[0])
		}

		locators = append(locators, &apicore.Locator{
			Type: apicore.LocatorType(artifactType),
			Source: &apicore.LocatorSource{
				Url: parts[1],
			},
		})
	}

	// Load config file if provided
	var config config.Config
	if opts.ConfigFile != "" {
		err := config.LoadFromFile(opts.ConfigFile)
		if err != nil {
			return fmt.Errorf("failed to load config file: %w", err)
		}
	}
	configLocators, err := config.GetAPILocators()
	if err != nil {
		return fmt.Errorf("failed to get locators from config: %w", err)
	}

	// Create agent data model from config and from options
	// Option values should override config values
	agent := &apicore.Agent{
		Name:      firstNonEmpty(opts.Name, config.Name),
		Version:   firstNonEmpty(opts.Version, config.Version),
		Authors:   firstNonEmptySlice(opts.Authors, config.Authors),
		CreatedAt: timestamppb.New(createdAt),
		Locators:  firstNonEmptySlice(locators, configLocators),
	}

	categories := firstNonEmptySlice(opts.Categories, config.Categories)
	llmanalyzer := opts.LLMAnalyzer

	// Build to obtain agent model
	err = builder.Build(cmd.Context(), agentPath, agent, categories, llmanalyzer)
	if err != nil {
		return fmt.Errorf("failed to build agent: %w", err)
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

func firstNonEmpty(opt, cfg string) string {
	if opt != "" {
		return opt
	}
	return cfg
}

func firstNonEmptySlice[T any](opt, cfg []T) []T {
	if len(opt) > 0 {
		return opt
	}
	return cfg
}
