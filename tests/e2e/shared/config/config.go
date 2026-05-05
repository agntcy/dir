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

type Config struct {
	Client  client.Config  `json:"client,omitzero"  mapstructure:"client"`
	Local   local.Config   `json:"local,omitzero"   mapstructure:"local"`
	Network network.Config `json:"network,omitzero" mapstructure:"network"`
	Daemon  daemon.Config  `json:"daemon,omitzero"  mapstructure:"daemon"`
}

func init() {
	// Require users to specify a config file path via a command-line flag
	flag.String("test-config", "", "Absolute path to test configuration file (optional)")
}

func LoadConfig() (*Config, error) {
	v := viper.NewWithOptions(
		viper.KeyDelimiter("."),
		viper.EnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_")),
	)

	configPathFlag := flag.Lookup("test-config")
	if configPathFlag == nil {
		return nil, fmt.Errorf("failed to lookup -test-config flag")
	}

	v.SetConfigFile(configPathFlag.Value.String())
	v.AllowEmptyEnv(true)
	v.AutomaticEnv()

	// Read the config data (from env/file)
	if err := v.ReadInConfig(); err != nil {
		// Config file was explicitly provided but could not be read
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) {
			return nil, fmt.Errorf("failed to read configuration file: %w", err)
		}
	}

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
