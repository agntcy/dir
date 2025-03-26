// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package logging

import (
	"context"
	"log/slog"
	"os"
)

// Key for context storage
type contextKey string

const loggerKey contextKey = "AgntcyDirectoryServerContextLogger"

// getLogOutput determines where logs should be written
func getLogOutput(logFilePath string) *os.File {
	if logFilePath != "" {
		// Try to open or create the log file
		file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			return file
		}

		// If file creation fails, log to stderr and fallback to stdout
		slog.Error("Failed to open log file, defaulting to stdout", "error", err)
	}

	return os.Stdout
}

func WithLogger(ctx context.Context, logFilePath string) context.Context {
	logOutput := getLogOutput(logFilePath)
	logger := slog.New(slog.NewTextHandler(logOutput, nil))

	return context.WithValue(ctx, loggerKey, logger)
}

// Retrieve logger from context (fallback to default if missing)
func LoggerFromContext(ctx context.Context) *slog.Logger {
	logger, ok := ctx.Value(loggerKey).(*slog.Logger)
	if !ok {
		return slog.New(slog.NewJSONHandler(os.Stdout, nil)) // Default logger
	}

	return logger
}
