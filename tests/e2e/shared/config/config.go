// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"errors"
	"flag"
	"fmt"
	"strings"

	client "github.com/agntcy/dir/tests/e2e/client/config"
	daemon "github.com/agntcy/dir/tests/e2e/daemon/config"
	local "github.com/agntcy/dir/tests/e2e/local/config"
	network "github.com/agntcy/dir/tests/e2e/network/config"
	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

// Holds the path to the configuration file (if provided via command-line flag).
var configPath string

func init() {
	flag.StringVar(&configPath, "config", "", "Absolute path to test configuration file (optional)")
}

type DeploymentMode string

const (
	DeploymentModeLocal   DeploymentMode = "local"
	DeploymentModeNetwork DeploymentMode = "network"
)

const (
	DefaultEnvPrefix = "DIRECTORY_E2E"

	DefaultDeploymentMode = DeploymentModeLocal
)

type Config struct {
	DeploymentMode DeploymentMode `json:"deployment_mode,omitempty" mapstructure:"deployment_mode"`
	Client         client.Config  `json:"client,omitzero"           mapstructure:"client"`
	Local          local.Config   `json:"local,omitzero"            mapstructure:"local"`
	Network        network.Config `json:"network,omitzero"          mapstructure:"network"`
	Daemon         daemon.Config  `json:"daemon,omitzero"           mapstructure:"daemon"`
}

func LoadConfig() (*Config, error) {
	v := viper.NewWithOptions(
		viper.KeyDelimiter("."),
		viper.EnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_")),
	)

	if configPath != "" {
		v.SetConfigFile(configPath)
	}

	v.SetEnvPrefix(DefaultEnvPrefix)
	v.AllowEmptyEnv(true)
	v.AutomaticEnv()

	// Read the config data (from env/file)
	if err := v.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) {
			// Config file was explicitly provided but could not be read
			return nil, fmt.Errorf("failed to read configuration file: %w", err)
		}
	}

	//
	// E2E test configuration
	//
	_ = v.BindEnv("deployment_mode")
	v.SetDefault("deployment_mode", DefaultDeploymentMode)

	// Load configuration into struct
	decodeHooks := mapstructure.ComposeDecodeHookFunc(
		mapstructure.TextUnmarshallerHookFunc(),
		mapstructure.StringToTimeDurationHookFunc(),
		mapstructure.StringToSliceHookFunc(","),
	)

	cfg := &Config{}
	if err := v.Unmarshal(cfg, viper.DecodeHook(decodeHooks)); err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	return cfg, nil
}
