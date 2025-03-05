// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package build

import (
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	apicore "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/cli/builder"
	"github.com/agntcy/dir/cli/cmd/build/config"
	"github.com/agntcy/dir/cli/presenter"
	"github.com/agntcy/dir/cli/types"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "build",
	Short: "Build agent model to prepare for pushing",
	Long: `Usage example:

	dirctl build --config-file agntcy-config.yaml

`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runCommand(cmd)
	},
}

func runCommand(cmd *cobra.Command) error {
	if opts.ConfigFile == "" {
		return fmt.Errorf("config file is required")
	}

	// Get configuration from flags
	buildConfig := &config.Config{}
	err := buildConfig.LoadFromFile(opts.ConfigFile)
	if err != nil {
		return fmt.Errorf("failed to load config file: %w", err)
	}

	err = buildConfig.Validate()
	if err != nil {
		return fmt.Errorf("failed to validate builder config: %w", err)
	}

	locators, err := buildConfig.GetAPILocators()
	if err != nil {
		return fmt.Errorf("failed to get locators from config: %w", err)
	}

	// Build to obtain agent model
	extensions, err := builder.Build(cmd.Context(), &buildConfig.Builder)
	if err != nil {
		return fmt.Errorf("failed to build agent: %w", err)
	}

	// Append config extensions
	for _, ext := range buildConfig.Extensions {
		extension := types.AgentExtension{
			Name:    ext.Name,
			Version: ext.Version,
			Specs:   ext.Specs,
		}

		apiExt, err := extension.ToAPIExtension()
		if err != nil {
			return fmt.Errorf("failed to convert extension to API extension: %w", err)
		}

		extensions = append(extensions, &apiExt)
	}

	// Create agent data model
	agent := &apicore.Agent{
		Name:       buildConfig.Name,
		Version:    buildConfig.Version,
		Authors:    buildConfig.Authors,
		CreatedAt:  timestamppb.New(time.Now()),
		Locators:   locators,
		Extensions: extensions,
	}

	// Construct output
	agentRaw, err := json.MarshalIndent(agent, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal built data: %w", err)
	}

	// Print to output
	presenter.Print(cmd, string(agentRaw))

	return nil
}
