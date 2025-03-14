// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package build

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/agntcy/dir/cli/builder"
	"github.com/agntcy/dir/cli/builder/config"
	"github.com/agntcy/dir/cli/presenter"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "build",
	Short: "Build agent model to prepare for pushing",
	Long: `Usage example:

	dirctl build --config agntcy-config.yaml

`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runCommand(cmd)
	},
}

func runCommand(cmd *cobra.Command) error {
	if opts.ConfigFile == "" {
		return errors.New("config file is required")
	}

	// Get configuration from flags
	cfg := &config.Config{}

	err := cfg.LoadFromFile(opts.ConfigFile)
	if err != nil {
		return fmt.Errorf("failed to load config file: %w", err)
	}

	builderInstance := builder.NewBuilder(cfg)

	err = builderInstance.RegisterPlugins()
	if err != nil {
		return fmt.Errorf("failed to register plugins: %w", err)
	}

	agent, err := builderInstance.BuildUserAgent()
	if err != nil {
		return fmt.Errorf("failed to build user agent: %w", err)
	}

	builderAgent, err := builderInstance.BuildAgent(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to build plugins: %w", err)
	}

	// Merge Agent Model from user config with Agent Model from plugins
	// User model will override plugin model
	agent.Merge(builderAgent)

	// Construct output
	agentRaw, err := json.MarshalIndent(agent, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal built data: %w", err)
	}

	// Print to output
	presenter.Print(cmd, string(agentRaw))

	return nil
}
