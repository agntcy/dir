// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

type Config struct {
	// Model path for Casbin
	ModelPath string `json:"model_path,omitempty" mapstructure:"model_path"`

	// Policy directory path for Casbin
	PolicyDir string `json:"policy_dir_path,omitempty" mapstructure:"policy_dir_path"`
}
