// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package logging

import (
	"log/slog"
	"os"
	"strings"
	"sync"
)

const filePermission = 0o644

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

func InitLogger(cfg *Config) {
	once.Do(func() {
		var logLevel slog.Level

		logOutput := getLogOutput(cfg.LogFile)

		// Parse log level; default to INFO if invalid.
		if err := logLevel.UnmarshalText([]byte(strings.ToLower(cfg.LogLevel))); err != nil {
			slog.Warn("Invalid log level, defaulting to INFO", "error", err)
			logLevel = slog.LevelInfo
		}

		// Set global logger before other packages initialize.
		slog.SetDefault(slog.New(slog.NewTextHandler(logOutput, &slog.HandlerOptions{Level: logLevel})))
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
