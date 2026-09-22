// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package logging

import (
	"io"
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

	// Log stream types.
	streamStdout = "stdout"
	streamStderr = "stderr"
)

var once sync.Once

// switchableWriter is an io.Writer whose target can be swapped after
// construction. slog handlers bind to a writer once, at construction time -
// including the ~60 `var logger = logging.Logger(component)` package-level
// loggers in this repo, all constructed before any binary's main() runs.
// Routing writes through a switchableWriter lets SetDefaultOutput redirect
// those already-constructed loggers later, without a custom slog.Handler.
type switchableWriter struct {
	mu sync.RWMutex
	w  io.Writer
}

func newSwitchableWriter(w io.Writer) *switchableWriter {
	return &switchableWriter{w: w}
}

func (s *switchableWriter) Write(p []byte) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	//nolint:wrapcheck // io.Writer implementations pass through the underlying error unwrapped.
	return s.w.Write(p)
}

func (s *switchableWriter) Set(w io.Writer) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.w = w
}

// defaultWriter is the process-wide fallback output target, used only when
// InitLogger did not bind the handler to an explicit log file or an explicit
// DIRECTORY_LOGGER_LOG_STREAM. Binaries pick their own default via
// SetDefaultOutput.
var defaultWriter = newSwitchableWriter(os.Stdout)

// getFileOutput attempts to open the configured log file. It returns nil if
// no file is configured, or if opening it fails - the failure is logged so
// the caller can fall back to the next output in the precedence order.
func getFileOutput(logFilePath string) *os.File {
	if logFilePath == "" {
		return nil
	}

	file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, filePermission)
	if err != nil {
		slog.Error("Failed to open log file, falling back", "error", err)

		return nil
	}

	return file
}

// resolveExplicitStream returns the writer for an explicit
// DIRECTORY_LOGGER_LOG_STREAM value. ok is false when the stream is unset or
// invalid, meaning the caller should fall back to the binary-default writer.
func resolveExplicitStream(logStream string) (io.Writer, bool) {
	switch strings.ToLower(strings.TrimSpace(logStream)) {
	case "":
		return nil, false
	case streamStdout:
		return os.Stdout, true
	case streamStderr:
		return os.Stderr, true
	default:
		slog.Warn("Invalid log stream, using default output", "log_stream", logStream)

		return nil, false
	}
}

// resolveOutput implements the output-selection precedence required by
// #2009:
//  1. An explicit log file that opens successfully always wins.
//  2. Otherwise an explicit LogStream (stdout/stderr) wins.
//  3. Otherwise logs flow through defaultWriter, which the binary can
//     redirect later via SetDefaultOutput.
func resolveOutput(cfg *Config) io.Writer {
	if file := getFileOutput(cfg.LogFile); file != nil {
		return file
	}

	if w, ok := resolveExplicitStream(cfg.LogStream); ok {
		return w
	}

	return defaultWriter
}

// InitLogger initializes the global logger with the provided configuration.
// It supports multiple output formats: text, json.
// This function is idempotent and thread-safe - it will only initialize once.
func InitLogger(cfg *Config) {
	once.Do(func() {
		var logLevel slog.Level

		// Parse log level; default to INFO if invalid.
		if err := logLevel.UnmarshalText([]byte(strings.ToLower(cfg.LogLevel))); err != nil {
			slog.Warn("Invalid log level, defaulting to INFO", "error", err)

			logLevel = slog.LevelInfo
		}

		output := resolveOutput(cfg)

		// Create handler based on format
		var handler slog.Handler

		opts := &slog.HandlerOptions{Level: logLevel}

		switch strings.ToLower(cfg.LogFormat) {
		case formatJSON:
			handler = slog.NewJSONHandler(output, opts)
		case formatText:
			handler = slog.NewTextHandler(output, opts)
		default:
			slog.Warn("Invalid log format, defaulting to text", "format", cfg.LogFormat)

			handler = slog.NewTextHandler(output, opts)
		}

		// Set global logger before other packages initialize.
		slog.SetDefault(slog.New(handler))
	})
}

// SetDefaultOutput redirects defaultWriter (see above). It has no effect on
// loggers bound to an explicit log file or DIRECTORY_LOGGER_LOG_STREAM, since
// those write directly to their configured target instead.
func SetDefaultOutput(w io.Writer) {
	defaultWriter.Set(w)
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
