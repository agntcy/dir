// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package build

import (
	"encoding/json"
	"fmt"

	apicore "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/cli/builder"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "build",
	Short: "Build agent model to prepare for pushing",
	Long: `Usage example:

	dirctl build \
		--name="agent-name" \
		--version="v1.0.0" \
		--artifact-url="http://ghcr.io/example/example" \
		--artifact-type="docker-image" \
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
	var ok bool
	var artifactType int32
	if artifactType, ok = apicore.LocatorType_value[opts.ArtifactType]; !ok {
		return fmt.Errorf("invalid artifact type: %s", opts.ArtifactType)
	}

	agent := &apicore.Agent{
		Name:    opts.Name,
		Version: opts.Version,
		Locators: []*apicore.Locator{
			{
				Type: apicore.LocatorType(artifactType),
				Source: &apicore.LocatorSource{
					Url: opts.ArtifactUrl,
				},
			},
		},
	}

	// Build to obtain agent model
	err := builder.Build(cmd.Context(), agentPath, agent, opts.Authors, opts.Categories, opts.LLMAnalyzer)
	if err != nil {
		return fmt.Errorf("failed to build agent: %w", err)
	}

	// Construct output
	outputRaw, err := json.MarshalIndent(agent, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal built data: %w", err)
	}

	// Print to output
	_, err = fmt.Fprint(cmd.OutOrStdout(), string(outputRaw))
	if err != nil {
		return fmt.Errorf("failed to print built data: %w", err)
	}

	return nil
}
