// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package build

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	apicore "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/cli/builder"
	"github.com/agntcy/dir/cli/builder/config"
	"github.com/agntcy/dir/cli/presenter"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/types/known/timestamppb"
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

	locators, err := cfg.GetAPILocators()
	if err != nil {
		return fmt.Errorf("failed to get locators from config: %w", err)
	}

	manager := builder.NewBuilder(cfg)

	err = manager.RegisterExtensions()
	if err != nil {
		return fmt.Errorf("failed to register extensions: %w", err)
	}

	extensions, err := manager.Run(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to run extension manager: %w", err)
	}

	// Create agent data model
	agent := &apicore.Agent{
		Name:       cfg.Model.Name,
		Version:    cfg.Model.Version,
		Authors:    cfg.Model.Authors,
		CreatedAt:  timestamppb.New(time.Now()),
		Skills:     cfg.Model.Skills,
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
