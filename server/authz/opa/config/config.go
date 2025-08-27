// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

type Config struct {
	// OPA bundle path, supports directory path or a path to compiled bundle
	BundlePath string `json:"policy_dir_path,omitempty" mapstructure:"policy_dir_path"`
}
