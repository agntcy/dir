// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

const (
	DefaultSQLiteDBPath = "/tmp/dir.db"
)

type Config struct {
	// DBPath is the path to the SQLite database file.
	DBPath string `json:"db_path,omitempty" mapstructure:"db_path"`
}
