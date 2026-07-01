// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"path/filepath"
)

const (
	// DefaultDatabaseType is the database driver used by the standalone
	// apiserver and reconciler when no type is configured.
	// The daemon overrides this to "sqlite" via its embedded config YAML.
	DefaultDatabaseType = "postgres"

	DefaultPostgresHost     = "localhost"
	DefaultPostgresPort     = 5432
	DefaultPostgresDatabase = "dir"
	DefaultPostgresSSLMode  = "disable"
)

// Database holds connection configuration for the shared state store.
// Both the apiserver and the reconciler point at the same Database instance.
type Database struct {
	// Type is the database driver: "sqlite" or "postgres".
	Type string `json:"type,omitempty" mapstructure:"type"`

	SQLite   SQLite   `json:"sqlite"   mapstructure:"sqlite"`
	Postgres Postgres `json:"postgres" mapstructure:"postgres"`
}

// SQLite holds SQLite-specific database configuration.
type SQLite struct {
	// Path is the filesystem path to the SQLite database file.
	// Defaults to ~/.dir/dir.db when empty.
	Path string `json:"path,omitempty" mapstructure:"path"`
}

// Postgres holds PostgreSQL-specific database configuration.
type Postgres struct {
	Host     string `json:"host,omitempty"     mapstructure:"host"`
	Port     int    `json:"port,omitempty"     mapstructure:"port"`
	Database string `json:"database,omitempty" mapstructure:"database"`
	Username string `json:"username,omitempty" mapstructure:"username"`

	//nolint:gosec // G117: intentional config field for database auth
	Password string `json:"password,omitempty" mapstructure:"password"`
	SSLMode  string `json:"ssl_mode,omitempty" mapstructure:"ssl_mode"`
}

// DefaultSQLitePath returns the default path for the SQLite database file
// (~/.dir/dir.db), creating the parent directory if needed.
// Called at load time, not at package init, so it never runs in read-only
// environments when SQLite isn't used.
func DefaultSQLitePath() string {
	return EnsureFilePath(filepath.Join(defaultDataDir(), "dir.db"))
}

// defaultDataDir returns ~/.dir, falling back to os.TempDir()/.dir.
func defaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.TempDir()
	}

	return filepath.Join(home, ".dir")
}

// EnsureFilePath creates parent directories for path and returns its absolute form.
// Returns path unchanged if MkdirAll or Abs fails.
func EnsureFilePath(path string) string {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil { //nolint:mnd
		return path
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return path
	}

	return absPath
}
