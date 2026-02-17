// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

const (
	// DefaultDBType is the default database type if not specified in the config.
	DefaultDBType = "postgres"

	// PostgreSQL defaults.
	DefaultPostgresHost     = "localhost"
	DefaultPostgresPort     = 5432
	DefaultPostgresDatabase = "dir"
	DefaultPostgresSSLMode  = "disable"
)

type Config struct {
	// DBType is the type of the database (postgres).
	DBType string `json:"db_type,omitempty" mapstructure:"db_type"`

	// PostgreSQL database configuration.
	Postgres PostgresConfig `json:"postgres" mapstructure:"postgres"`
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

	// SSLMode indicates the SSL mode for the connection.
	SSLMode string `json:"ssl_mode,omitempty" mapstructure:"ssl_mode"`
}
