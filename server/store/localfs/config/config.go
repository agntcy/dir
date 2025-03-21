// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

const DefaultDir = "/tmp"

type Config struct {
	Dir string `json:"dir,omitempty" mapstructure:"dir"`
}
