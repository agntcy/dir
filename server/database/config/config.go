// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

const (
	DefaultDBType = "postgres"

	// SQLite defaults.
	DefaultSQLiteDBPath = "/tmp/dir.db"

	// PostgreSQL defaults.
	DefaultPostgresHost     = "localhost"
	DefaultPostgresPort     = 5432
	DefaultPostgresDatabase = "dir"
)

type Config struct {
	// DBType is the type of the database (sqlite or postgres).
	DBType string `json:"db_type,omitempty" mapstructure:"db_type"`

	// SQLite database configuration.
	SQLite SQLiteConfig `json:"sqlite" mapstructure:"sqlite"`

	// PostgreSQL database configuration.
	Postgres PostgresConfig `json:"postgres" mapstructure:"postgres"`
}

type SQLiteConfig struct {
	// DBPath is the path to the SQLite database file.
	DBPath string `json:"db_path,omitempty" mapstructure:"db_path"`
}

type PostgresConfig struct {
	// Host is the PostgreSQL server hostname.
	Host string `json:"host,omitempty" mapstructure:"host"`

	// Port is the PostgreSQL server port.
	Port int `json:"port,omitempty" mapstructure:"port"`

	// Database is the name of the database to connect to.
	Database string `json:"database,omitempty" mapstructure:"database"`

	// Username is the database user.
	Username string `json:"username,omitempty" mapstructure:"username"`

	// Password is the database password.
	Password string `json:"password,omitempty" mapstructure:"password"`
}
