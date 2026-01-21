// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"log/slog"
	"os"
)

// Logger wraps slog.Logger with predefined attributes.
type Logger struct {
	*slog.Logger
}

var defaultLogger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
	Level: slog.LevelInfo,
}))

// NewLogger creates a logger with kind and type attributes for the component.
func NewLogger(kind, componentType string) *Logger {
	return &Logger{
		Logger: defaultLogger.With(
			slog.String("kind", kind),
			slog.String("type", componentType),
		),
	}
}

// Info logs an info message.
func (l *Logger) Info(msg string, args ...any) {
	l.Logger.Info(msg, args...)
}

// Error logs an error message.
func (l *Logger) Error(msg string, args ...any) {
	l.Logger.Error(msg, args...)
}

// Warn logs a warning message.
func (l *Logger) Warn(msg string, args ...any) {
	l.Logger.Warn(msg, args...)
}

// Debug logs a debug message.
func (l *Logger) Debug(msg string, args ...any) {
	l.Logger.Debug(msg, args...)
}

// Fatal logs an error message and exits.
func (l *Logger) Fatal(msg string, args ...any) {
	l.Logger.Error(msg, args...)
	os.Exit(1)
}

// With returns a new Logger with the provided attributes.
func (l *Logger) With(args ...any) *Logger {
	return &Logger{
		Logger: l.Logger.With(args...),
	}
}
