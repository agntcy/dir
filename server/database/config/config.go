// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

const (
	DefaultDBType = "sqlite"

	DefaultSQLiteDBPath = "/tmp/dir.db"
)

type Config struct {
	// DBType is the type of the database.
	DBType string `json:"db_type,omitempty" mapstructure:"db_type"`

	// SQLite database configuration.
	SQLite SQLiteConfig `json:"sqlite" mapstructure:"sqlite"`
}

type SQLiteConfig struct {
	// DBPath is the path to the SQLite database file.
	DBPath string `json:"db_path,omitempty" mapstructure:"db_path"`
}
