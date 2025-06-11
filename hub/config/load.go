// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package config provides configuration management for the Agent Hub CLI and related applications.
package config

import (
	"errors"
	"fmt"

	"github.com/agntcy/dir/hub/utils/dir"
	"github.com/spf13/viper"
)

// LoadConfig loads the application configuration from a JSON file using Viper.
// It searches for the config file in the current directory and the application directory.
// Returns nil if the config file is not found, or an error if reading/parsing fails.
func LoadConfig() error {
	viper.SetConfigName("config")
	viper.SetConfigType("json")
	viper.AddConfigPath(".")
	viper.AddConfigPath(dir.GetAppDir())

	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) {
			return nil
		}

		return fmt.Errorf("failed to read config file: %w", err)
	}

	return nil
}
