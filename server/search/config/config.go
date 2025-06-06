// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

const (
	DefaultDBType = "sqlite"
)

type Config struct {
	// DBType is the type of the search database.
	DB string `json:"db_type,omitempty" mapstructure:"db_type"`
}
