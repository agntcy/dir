// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	sqliteconfig "github.com/agntcy/dir/server/database/sqlite/config"
)

const (
	DefaultDBType = "sqlite"
)

type Config struct {
	// DBType is the type of the database.
	DBType string `json:"db_type,omitempty" mapstructure:"db_type"`

	// Config for SQLite database.
	SQLite sqliteconfig.Config `json:"sqlite,omitempty" mapstructure:"sqlite"`
}
