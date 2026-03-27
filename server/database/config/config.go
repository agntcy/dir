// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"path/filepath"
)

const (
	// DefaultType is the default database type if not specified in the config.
	DefaultType = "postgres"

	// PostgreSQL defaults.
	DefaultPostgresHost     = "localhost"
	DefaultPostgresPort     = 5432
	DefaultPostgresDatabase = "dir"
	DefaultPostgresSSLMode  = "disable"
)

// DefaultDataDir is the persistent data directory (~/.dir).
var DefaultDataDir = EnsureFilePath(filepath.Join(GetDataDir(), ".dir"))

// DefaultSQLitePath is the default path for the SQLite database file.
var DefaultSQLitePath = EnsureFilePath(filepath.Join(DefaultDataDir, "dir.db"))

// GetDataDir returns the user home directory, falling back to os.TempDir().
func GetDataDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return os.TempDir()
	}

	return homeDir
}

// EnsureFilePath creates parent directories for path and returns its absolute form.
func EnsureFilePath(path string) string {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil { //nolint:mnd
		panic(err)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return path
	}

	return absPath
}

type Config struct {
	// Type is the type of the database (sqlite or postgres).
	Type string `json:"type,omitempty" mapstructure:"type"`

	// SQLite database configuration.
	SQLite SQLiteConfig `json:"sqlite" mapstructure:"sqlite"`

	// PostgreSQL database configuration.
	Postgres PostgresConfig `json:"postgres" mapstructure:"postgres"`
}

type SQLiteConfig struct {
	// Path is the filesystem path to the SQLite database file.
	Path string `json:"path,omitempty" mapstructure:"path"`
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
	//nolint:gosec // G117: intentional config field for database auth
	Password string `json:"password,omitempty" mapstructure:"password"`

	// SSLMode indicates the SSL mode for the connection.
	SSLMode string `json:"ssl_mode,omitempty" mapstructure:"ssl_mode"`
}
