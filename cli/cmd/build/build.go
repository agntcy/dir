// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package build

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	oasf "github.com/agntcy/dir/api/core/v1alpha1/oasf-validator"
	"github.com/agntcy/dir/cli/builder"
	"github.com/agntcy/dir/cli/builder/config"
	"github.com/agntcy/dir/cli/presenter"
	"github.com/spf13/cobra"
)

const ConfigFile = "build.config.yml"

var Command = &cobra.Command{
	Use:   "build",
	Short: "Build agent model to prepare for pushing",
	Long: `
This command handles the build process for agent data models
from source code. It generates a JSON object that 
describes an agent and satisfies the **Open Agent Schema Framework** specification.

Usage examples:

1. When build config is present under the agent source code:

	dirctl build ./path-to-agent

2. When build config is either not present or we want to override config from path:

	dirctl build ./path-to-agent --config build.yml

`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var path string
		if len(args) == 0 {
			path = "." //nolint:ineffassign
		}
		if len(args) == 1 {
			path = args[0]
		} else {
			return errors.New("arg missing: only one path can be specified allowed")
		}

		return runCommand(cmd, path)
	},
}

func runCommand(cmd *cobra.Command, agentPath string) error {
	// Get configuration file path
	configFile := opts.ConfigFile
	if configFile == "" {
		configFilePath := filepath.Join(agentPath, ConfigFile)
		if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
			return fmt.Errorf("config file not specified and not found in agent path: %s", configFilePath)
		}

		configFile = configFilePath
	}

	// Get configuration from file
	cfg := &config.Config{}

	err := cfg.LoadFromFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config file: %w", err)
	}

	builderInstance := builder.NewBuilder(agentPath, cfg)

	err = builderInstance.RegisterPlugins()
	if err != nil {
		return fmt.Errorf("failed to register plugins: %w", err)
	}

	// load base agent from file
	agent := &coretypes.Agent{}
	baseAgentPath := filepath.Join(filepath.Dir(configFile), cfg.Builder.BaseModelPath)

	_, err = agent.LoadFromFile(baseAgentPath)
	if err != nil {
		return fmt.Errorf("failed to load agent from file %s: %w", baseAgentPath, err)
	}

	// run plugins
	builderAgent, err := builderInstance.BuildAgent(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to build plugins: %w", err)
	}

	// Merge Agent Model from user config with Agent Model from plugins
	// User model will override plugin model
	agent.Merge(builderAgent)

	if cfg.Builder.OASFValidation {
		// Validate skills
		err = oasf.ValidateSkills(agent.GetSkills())
		if err != nil {
			return fmt.Errorf("failed to validate skills: %w", err)
		}
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
