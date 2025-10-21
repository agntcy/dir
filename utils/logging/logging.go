// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package logging

import (
	"log/slog"
	"os"
	"strings"
	"sync"
)

const (
	filePermission = 0o644

	// Log format types.
	formatJSON = "json"
	formatText = "text"
)

var once sync.Once

// getLogOutput determines where logs should be written.
func getLogOutput(logFilePath string) *os.File {
	if logFilePath != "" {
		// Try to open or create the log file.
		file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, filePermission)
		if err == nil {
			return file
		}

		slog.Error("Failed to open log file, defaulting to stdout", "error", err)
	}

	return os.Stdout
}

// InitLogger initializes the global logger with the provided configuration.
// It supports multiple output formats: text, json.
// This function is idempotent and thread-safe - it will only initialize once.
func InitLogger(cfg *Config) {
	once.Do(func() {
		var logLevel slog.Level

		logOutput := getLogOutput(cfg.LogFile)

		// Parse log level; default to INFO if invalid.
		if err := logLevel.UnmarshalText([]byte(strings.ToLower(cfg.LogLevel))); err != nil {
			slog.Warn("Invalid log level, defaulting to INFO", "error", err)
			logLevel = slog.LevelInfo
		}

		// Create handler based on format
		var handler slog.Handler

		opts := &slog.HandlerOptions{Level: logLevel}

		switch strings.ToLower(cfg.LogFormat) {
		case formatJSON:
			handler = slog.NewJSONHandler(logOutput, opts)
		case formatText:
			handler = slog.NewTextHandler(logOutput, opts)
		default:
			slog.Warn("Invalid log format, defaulting to text", "format", cfg.LogFormat)
			handler = slog.NewTextHandler(logOutput, opts)
		}

		// Set global logger before other packages initialize.
		slog.SetDefault(slog.New(handler))
	})
}

func Logger(component string) *slog.Logger {
	return slog.Default().With("component", component)
}

func init() {
	cfg, err := LoadConfig()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	InitLogger(cfg)
}
