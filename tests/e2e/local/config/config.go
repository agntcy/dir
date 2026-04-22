// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

type Config struct {
	// ServerAddress is the address the server binds to.
	ServerAddress string `json:"server_address,omitempty" mapstructure:"server_address"`

	// MetricsAddress is the address the metric endpoint binds to.
	MetricsAddress string `json:"metrics_address,omitempty" mapstructure:"metrics_address"`

	// CliPath is the path to the CLI binary.
	CliPath string `json:"cli_path,omitempty" mapstructure:"cli_path"`

	// CliExtraArgs are extra arguments to pass to the CLI.
	CliExtraArgs []string `json:"cli_extra_args,omitempty" mapstructure:"cli_extra_args"`
}
