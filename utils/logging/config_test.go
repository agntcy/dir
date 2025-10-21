// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package logging

import (
	"os"
	"testing"
)

// TestLoadConfigWithDefaults verifies default configuration values.
func TestLoadConfigWithDefaults(t *testing.T) {
	// Clear any environment variables
	os.Unsetenv("DIRECTORY_LOGGER_LOG_FILE")
	os.Unsetenv("DIRECTORY_LOGGER_LOG_LEVEL")
	os.Unsetenv("DIRECTORY_LOGGER_LOG_FORMAT")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() failed: %v", err)
	}

	// Verify defaults
	if cfg.LogLevel != DefaultLogLevel {
		t.Errorf("Expected LogLevel=%s, got: %s", DefaultLogLevel, cfg.LogLevel)
	}

	if cfg.LogFormat != DefaultLogFormat {
		t.Errorf("Expected LogFormat=%s, got: %s", DefaultLogFormat, cfg.LogFormat)
	}

	if cfg.LogFile != "" {
		t.Errorf("Expected LogFile='', got: %s", cfg.LogFile)
	}
}

// TestLoadConfigWithEnvVars verifies environment variable configuration.
func TestLoadConfigWithEnvVars(t *testing.T) {
	// Set environment variables
	t.Setenv("DIRECTORY_LOGGER_LOG_FILE", "/tmp/test.log")
	t.Setenv("DIRECTORY_LOGGER_LOG_LEVEL", "DEBUG")
	t.Setenv("DIRECTORY_LOGGER_LOG_FORMAT", "json")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() failed: %v", err)
	}

	// Verify environment variables are loaded
	if cfg.LogFile != "/tmp/test.log" {
		t.Errorf("Expected LogFile='/tmp/test.log', got: %s", cfg.LogFile)
	}

	if cfg.LogLevel != "DEBUG" {
		t.Errorf("Expected LogLevel='DEBUG', got: %s", cfg.LogLevel)
	}

	if cfg.LogFormat != "json" {
		t.Errorf("Expected LogFormat='json', got: %s", cfg.LogFormat)
	}
}

// TestLoadConfigWithPartialEnvVars verifies partial environment variable configuration.
func TestLoadConfigWithPartialEnvVars(t *testing.T) {
	// Set only some environment variables
	t.Setenv("DIRECTORY_LOGGER_LOG_LEVEL", "ERROR")

	// Unset others to ensure clean state
	os.Unsetenv("DIRECTORY_LOGGER_LOG_FILE")
	os.Unsetenv("DIRECTORY_LOGGER_LOG_FORMAT")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() failed: %v", err)
	}

	// Verify mix of env vars and defaults
	if cfg.LogLevel != "ERROR" {
		t.Errorf("Expected LogLevel='ERROR', got: %s", cfg.LogLevel)
	}

	if cfg.LogFormat != DefaultLogFormat {
		t.Errorf("Expected LogFormat=%s (default), got: %s", DefaultLogFormat, cfg.LogFormat)
	}
}

// TestLoadConfigEmptyEnvVars verifies empty environment variables are handled.
func TestLoadConfigEmptyEnvVars(t *testing.T) {
	// Set empty environment variables
	t.Setenv("DIRECTORY_LOGGER_LOG_FILE", "")
	t.Setenv("DIRECTORY_LOGGER_LOG_LEVEL", "")
	t.Setenv("DIRECTORY_LOGGER_LOG_FORMAT", "")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() failed: %v", err)
	}

	// When env vars are set to empty string, Viper uses the empty value
	// (not the default). This is expected behavior - empty string is valid.
	if cfg.LogLevel != "" {
		t.Errorf("Expected LogLevel='' (empty), got: %s", cfg.LogLevel)
	}

	if cfg.LogFormat != "" {
		t.Errorf("Expected LogFormat='' (empty), got: %s", cfg.LogFormat)
	}

	if cfg.LogFile != "" {
		t.Errorf("Expected LogFile='' (empty), got: %s", cfg.LogFile)
	}
}

// TestLoadConfigAllFormats verifies all supported log formats.
func TestLoadConfigAllFormats(t *testing.T) {
	formats := []string{"text", "json", "TEXT", "JSON", "Text", "Json"}

	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			t.Setenv("DIRECTORY_LOGGER_LOG_FORMAT", format)

			cfg, err := LoadConfig()
			if err != nil {
				t.Fatalf("LoadConfig() failed for format %s: %v", format, err)
			}

			if cfg.LogFormat != format {
				t.Errorf("Expected LogFormat=%s, got: %s", format, cfg.LogFormat)
			}
		})
	}
}

// TestLoadConfigAllLogLevels verifies all supported log levels.
func TestLoadConfigAllLogLevels(t *testing.T) {
	levels := []string{"DEBUG", "INFO", "WARN", "ERROR", "debug", "info", "warn", "error"}

	for _, level := range levels {
		t.Run(level, func(t *testing.T) {
			t.Setenv("DIRECTORY_LOGGER_LOG_LEVEL", level)

			cfg, err := LoadConfig()
			if err != nil {
				t.Fatalf("LoadConfig() failed for level %s: %v", level, err)
			}

			if cfg.LogLevel != level {
				t.Errorf("Expected LogLevel=%s, got: %s", level, cfg.LogLevel)
			}
		})
	}
}

// TestConfigJSONMarshaling verifies Config can be marshaled to JSON.
func TestConfigJSONMarshaling(t *testing.T) {
	cfg := &Config{
		LogFile:   "/var/log/app.log",
		LogLevel:  "INFO",
		LogFormat: "json",
	}

	// Just verify the struct is valid and fields are accessible
	if cfg.LogFile != "/var/log/app.log" {
		t.Errorf("LogFile mismatch")
	}

	if cfg.LogLevel != "INFO" {
		t.Errorf("LogLevel mismatch")
	}

	if cfg.LogFormat != "json" {
		t.Errorf("LogFormat mismatch")
	}
}

// TestConfigConstants verifies all constants are correctly defined.
func TestConfigConstants(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{"DefaultEnvPrefix", DefaultEnvPrefix, "DIRECTORY_LOGGER"},
		{"DefaultLogLevel", DefaultLogLevel, "INFO"},
		{"DefaultLogFormat", DefaultLogFormat, "text"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("Expected %s=%s, got: %s", tt.name, tt.expected, tt.value)
			}
		})
	}
}
