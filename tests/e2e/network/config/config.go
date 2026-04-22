// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

type Config struct {
	// CliPath is the path to the DIRCTL executable to use for tests.
	CliPath string `json:"cli_path,omitempty" mapstructure:"cli_path"`

	// Peer1 connection details
	Peer1ServerAddress         string   `json:"peer1_server_address,omitempty"          mapstructure:"peer1_server_address"`
	Peer1InternalServerAddress string   `json:"peer1_internal_server_address,omitempty" mapstructure:"peer1_internal_server_address"`
	Peer1CliExtraArgs          []string `json:"peer1_cli_extra_args,omitempty"          mapstructure:"peer1_cli_extra_args"`

	// Peer2 connection details
	Peer2ServerAddress string   `json:"peer2_server_address,omitempty" mapstructure:"peer2_server_address"`
	Peer2CliExtraArgs  []string `json:"peer2_cli_extra_args,omitempty" mapstructure:"peer2_cli_extra_args"`

	// Peer3 connection details
	Peer3ServerAddress string   `json:"peer3_server_address,omitempty" mapstructure:"peer3_server_address"`
	Peer3CliExtraArgs  []string `json:"peer3_cli_extra_args,omitempty" mapstructure:"peer3_cli_extra_args"`
}
