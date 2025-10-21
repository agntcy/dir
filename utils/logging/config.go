// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package logging

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

const (
	DefaultEnvPrefix = "DIRECTORY_LOGGER"
	DefaultLogLevel  = "INFO"
	DefaultLogFormat = "text"
)

type Config struct {
	LogFile   string `json:"log_file,omitempty"   mapstructure:"log_file"`
	LogLevel  string `json:"log_level,omitempty"  mapstructure:"log_level"`
	LogFormat string `json:"log_format,omitempty" mapstructure:"log_format"`
}

func LoadConfig() (*Config, error) {
	v := viper.NewWithOptions(
		viper.KeyDelimiter("."),
		viper.EnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_")),
	)

	v.SetEnvPrefix(DefaultEnvPrefix)
	v.AllowEmptyEnv(true)
	v.AutomaticEnv()

	_ = v.BindEnv("log_file")

	_ = v.BindEnv("log_level")
	v.SetDefault("log_level", DefaultLogLevel)

	_ = v.BindEnv("log_format")
	v.SetDefault("log_format", DefaultLogFormat)

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
