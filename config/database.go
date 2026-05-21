// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"path/filepath"
)

// Database type defaults.
const (
	// DefaultDatabaseType is the default database driver if not set in
	// configuration.
	DefaultDatabaseType = "postgres"

	// DefaultPostgresHost is the default PostgreSQL host.
	DefaultPostgresHost = "localhost"

	// DefaultPostgresPort is the default PostgreSQL port.
	DefaultPostgresPort = 5432

	// DefaultPostgresDatabase is the default PostgreSQL database name.
	DefaultPostgresDatabase = "dir"

	// DefaultPostgresSSLMode is the default PostgreSQL SSL mode.
	DefaultPostgresSSLMode = "disable"
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

// EnsureFilePath creates parent directories for path and returns its
// absolute form.
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

// Database holds connection configuration for the shared state store.
// Both the apiserver and the reconciler point at the same Database.
type Database struct {
	// Type is the database driver ("sqlite" or "postgres").
	Type string `json:"type,omitempty" mapstructure:"type"`

	// SQLite holds SQLite-specific configuration.
	SQLite SQLite `json:"sqlite" mapstructure:"sqlite"`

	// Postgres holds PostgreSQL-specific configuration.
	Postgres Postgres `json:"postgres" mapstructure:"postgres"`
}

// SQLite holds SQLite-specific database configuration.
type SQLite struct {
	// Path is the filesystem path to the SQLite database file.
	Path string `json:"path,omitempty" mapstructure:"path"`
}

// Postgres holds PostgreSQL-specific database configuration.
type Postgres struct {
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

	// SSLMode is the SSL mode for the connection.
	SSLMode string `json:"ssl_mode,omitempty" mapstructure:"ssl_mode"`
}
