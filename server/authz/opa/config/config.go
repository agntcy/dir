// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

type Config struct {
	// Path to the OPA policy directory
	PolicyDirPath string `json:"policy_dir_path,omitempty" mapstructure:"policy_dir_path"`
}
