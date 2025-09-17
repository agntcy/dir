// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

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
}

func LoadConfig() (*Config, error) {
	v := viper.NewWithOptions(
		viper.KeyDelimiter("."),
		viper.EnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_")),
	)

	v.SetEnvPrefix(DefaultEnvPrefix)
	v.AllowEmptyEnv(true)
	v.AutomaticEnv()

	_ = v.BindEnv("deployment_mode")
	v.SetDefault("deployment_mode", DefaultDeploymentMode)

	// Load configuration into struct
	decodeHooks := mapstructure.ComposeDecodeHookFunc(
		mapstructure.TextUnmarshallerHookFunc(),
		mapstructure.StringToTimeDurationHookFunc(),
		mapstructure.StringToSliceHookFunc(","),
	)

	config := &Config{}
	if err := v.Unmarshal(config, viper.DecodeHook(decodeHooks)); err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	return config, nil
}
