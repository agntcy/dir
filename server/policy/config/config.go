// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

// Config points the server at a directory of file-based OPA policies.
// Named .rego files are loaded by validators with provider "opa" via
// config.file.
type Config struct {
	// Dir is the directory that holds policy files (one file per policy).
	Dir string `json:"dir,omitempty" mapstructure:"dir"`
}
