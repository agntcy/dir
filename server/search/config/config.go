// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

const (
	DefaultDBType = "sqlite"
)

type Config struct {
	// DBType is the type of the search database.
	DBType string `json:"db_type,omitempty" mapstructure:"db_type"`

	// SQLiteDBPath is the path to the SQLite database file.
	SQLiteDBPath string `json:"sqlite_db_path,omitempty" mapstructure:"sqlite_db_path"`
}
