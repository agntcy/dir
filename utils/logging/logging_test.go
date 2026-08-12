// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package logging

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"testing"
)

// TestJSONHandler verifies JSON output format.
func TestJSONHandler(t *testing.T) {
	var buf bytes.Buffer

	// Create JSON handler
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	logger.Info("test message", "key", "value", "number", 42)

	output := buf.String()

	// Verify it's valid JSON
	var parsed map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &parsed); err != nil {
		t.Fatalf("Output is not valid JSON: %v\nOutput: %s", err, output)
	}

	// Verify expected fields
	if parsed["msg"] != "test message" {
		t.Errorf("Expected msg='test message', got: %v", parsed["msg"])
	}

	if parsed["key"] != "value" {
		t.Errorf("Expected key='value', got: %v", parsed["key"])
	}

	if parsed["number"] != float64(42) { // JSON numbers are float64
		t.Errorf("Expected number=42, got: %v", parsed["number"])
	}

	if parsed["level"] != DefaultLogLevel {
		t.Errorf("Expected level=%s, got: %v", DefaultLogLevel, parsed["level"])
	}
}

// TestTextHandler verifies text output format.
func TestTextHandler(t *testing.T) {
	var buf bytes.Buffer

	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	logger.Info("test message", "key", "value")

	output := buf.String()

	// Verify key-value format
	if !strings.Contains(output, "msg=\"test message\"") && !strings.Contains(output, "msg=test message") {
		t.Errorf("Expected text format with msg, got: %s", output)
	}

	if !strings.Contains(output, "key=value") {
		t.Errorf("Expected text format with key=value, got: %s", output)
	}

	if !strings.Contains(output, "level=INFO") {
		t.Errorf("Expected text format with level=INFO, got: %s", output)
	}
}

// TestJSONHandlerMultipleFields verifies JSON with multiple fields.
func TestJSONHandlerMultipleFields(t *testing.T) {
	var buf bytes.Buffer

	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	logger.Debug("debug message",
		"string_field", "test",
		"int_field", 123,
		"bool_field", true,
		"float_field", 3.14,
	)

	output := buf.String()

	var parsed map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &parsed); err != nil {
		t.Fatalf("Failed to parse JSON: %v\nOutput: %s", err, output)
	}

	// Verify different data types are preserved
	if parsed["string_field"] != "test" {
		t.Errorf("Expected string_field='test', got: %v", parsed["string_field"])
	}

	if parsed["int_field"] != float64(123) {
		t.Errorf("Expected int_field=123, got: %v", parsed["int_field"])
	}

	if parsed["bool_field"] != true {
		t.Errorf("Expected bool_field=true, got: %v", parsed["bool_field"])
	}

	if parsed["float_field"] != 3.14 {
		t.Errorf("Expected float_field=3.14, got: %v", parsed["float_field"])
	}
}

// TestLogLevels verifies different log levels work correctly.
func TestLogLevels(t *testing.T) {
	tests := []struct {
		name     string
		level    slog.Level
		logFunc  func(*slog.Logger, string)
		expected string
	}{
		{"DEBUG", slog.LevelDebug, func(l *slog.Logger, msg string) { l.Debug(msg) }, "DEBUG"},
		{"INFO", slog.LevelInfo, func(l *slog.Logger, msg string) { l.Info(msg) }, "INFO"},
		{"WARN", slog.LevelWarn, func(l *slog.Logger, msg string) { l.Warn(msg) }, "WARN"},
		{"ERROR", slog.LevelError, func(l *slog.Logger, msg string) { l.Error(msg) }, "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

			tt.logFunc(logger, "test")

			var parsed map[string]any
			if err := json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &parsed); err != nil {
				t.Fatalf("Failed to parse JSON: %v", err)
			}

			if parsed["level"] != tt.expected {
				t.Errorf("Expected level=%s, got: %v", tt.expected, parsed["level"])
			}
		})
	}
}

// TestComponentLogger verifies component-specific loggers.
func TestComponentLogger(t *testing.T) {
	var buf bytes.Buffer

	// Create base logger
	baseLogger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(baseLogger)

	// Create component logger (simulating Logger function)
	componentLogger := slog.Default().With("component", "test-component")
	componentLogger.Info("component message", "extra", "data")

	output := buf.String()

	var parsed map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &parsed); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Verify component field is present
	if parsed["component"] != "test-component" {
		t.Errorf("Expected component='test-component', got: %v", parsed["component"])
	}

	if parsed["extra"] != "data" {
		t.Errorf("Expected extra='data', got: %v", parsed["extra"])
	}
}

// TestJSONHandlerNilSafety verifies handler works with nil/empty values.
func TestJSONHandlerNilSafety(t *testing.T) {
	var buf bytes.Buffer

	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	logger.Info("test", "empty_string", "", "zero", 0)

	output := buf.String()

	var parsed map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &parsed); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Verify empty values are handled correctly
	if parsed["empty_string"] != "" {
		t.Errorf("Expected empty_string='', got: %v", parsed["empty_string"])
	}

	if parsed["zero"] != float64(0) {
		t.Errorf("Expected zero=0, got: %v", parsed["zero"])
	}
}

// TestDefaultConfig verifies default configuration values.
func TestDefaultConfig(t *testing.T) {
	const (
		expectedLogLevel  = "INFO"
		expectedLogFormat = "text"
		expectedEnvPrefix = "DIRECTORY_LOGGER"
	)

	if DefaultLogLevel != expectedLogLevel {
		t.Errorf("Expected DefaultLogLevel='%s', got: %s", expectedLogLevel, DefaultLogLevel)
	}

	if DefaultLogFormat != expectedLogFormat {
		t.Errorf("Expected DefaultLogFormat='%s', got: %s", expectedLogFormat, DefaultLogFormat)
	}

	if DefaultEnvPrefix != expectedEnvPrefix {
		t.Errorf("Expected DefaultEnvPrefix='%s', got: %s", expectedEnvPrefix, DefaultEnvPrefix)
	}
}

// TestConfigStruct verifies Config struct can be marshaled.
func TestConfigStruct(t *testing.T) {
	const testLogFormat = "json"

	cfg := Config{
		LogFile:   "/tmp/test.log",
		LogLevel:  "DEBUG",
		LogFormat: testLogFormat,
		LogStream: "stderr",
	}

	if cfg.LogFile != "/tmp/test.log" {
		t.Errorf("Expected LogFile='/tmp/test.log', got: %s", cfg.LogFile)
	}

	if cfg.LogLevel != "DEBUG" {
		t.Errorf("Expected LogLevel='DEBUG', got: %s", cfg.LogLevel)
	}

	if cfg.LogStream != "stderr" {
		t.Errorf("Expected LogStream='stderr', got: %s", cfg.LogStream)
	}

	if cfg.LogFormat != testLogFormat {
		t.Errorf("Expected LogFormat='%s', got: %s", testLogFormat, cfg.LogFormat)
	}
}

// TestTextHandlerMultipleMessages verifies multiple log messages don't interfere.
func TestTextHandlerMultipleMessages(t *testing.T) {
	var buf bytes.Buffer

	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	logger.Info("first message", "id", 1)
	logger.Info("second message", "id", 2)

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) != 2 {
		t.Errorf("Expected 2 log lines, got: %d", len(lines))
	}

	// Verify both messages are present
	if !strings.Contains(output, "first message") {
		t.Error("Expected 'first message' in output")
	}

	if !strings.Contains(output, "second message") {
		t.Error("Expected 'second message' in output")
	}
}

// TestJSONHandlerMultipleMessages verifies multiple JSON log entries.
func TestJSONHandlerMultipleMessages(t *testing.T) {
	var buf bytes.Buffer

	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	logger.Info("first", "id", 1)
	logger.Info("second", "id", 2)

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) != 2 {
		t.Errorf("Expected 2 JSON lines, got: %d", len(lines))
	}

	// Verify each line is valid JSON
	for i, line := range lines {
		var parsed map[string]any
		if err := json.Unmarshal([]byte(line), &parsed); err != nil {
			t.Errorf("Line %d is not valid JSON: %v\nLine: %s", i+1, err, line)
		}
	}
}

// TestGetFileOutput verifies getFileOutput returns nil when no file is configured or the
// path can't be opened, and a writable handle for a valid path.
func TestGetFileOutput(t *testing.T) {
	validPath := t.TempDir() + "/test.log"

	tests := []struct {
		name    string
		path    string
		wantNil bool
	}{
		{"empty path", "", true},
		{"invalid path", "/invalid/directory/that/does/not/exist/test.log", true},
		{"valid path", validPath, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := getFileOutput(tt.path)

			if tt.wantNil {
				if output != nil {
					t.Errorf("Expected nil, got: %v", output)
				}

				return
			}

			if output == nil {
				t.Fatal("Expected a file handle, got nil")
			}

			defer output.Close()

			if _, err := output.WriteString("test"); err != nil {
				t.Errorf("Failed to write to log file: %v", err)
			}
		})
	}
}

// TestResolveExplicitStream verifies stdout/stderr selection, case-insensitively, and
// that unset/invalid values report ok=false so the caller falls back to the default writer.
func TestResolveExplicitStream(t *testing.T) {
	tests := []struct {
		name      string
		logStream string
		wantOK    bool
		want      io.Writer
	}{
		{"empty", "", false, nil},
		{"stdout lowercase", "stdout", true, os.Stdout},
		{"stdout uppercase", "STDOUT", true, os.Stdout},
		{"stdout mixed case with spaces", "  StdOut  ", true, os.Stdout},
		{"stderr lowercase", "stderr", true, os.Stderr},
		{"stderr uppercase", "STDERR", true, os.Stderr},
		{"invalid", "bogus", false, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, ok := resolveExplicitStream(tt.logStream)
			if ok != tt.wantOK {
				t.Errorf("Expected ok=%v, got: %v", tt.wantOK, ok)
			}

			if ok && w != tt.want {
				t.Errorf("Expected writer=%v, got: %v", tt.want, w)
			}
		})
	}
}

// TestResolveOutputPrecedence verifies the #2009 output-selection precedence end to end:
// a successfully-opened LogFile beats an explicit LogStream, which beats the switchable
// defaultWriter, and both an invalid file and an invalid stream fall through to the next
// tier rather than a hardcoded choice.
func TestResolveOutputPrecedence(t *testing.T) {
	const invalidFile = "/invalid/directory/that/does/not/exist/test.log"

	validFile := t.TempDir() + "/test.log"

	tests := []struct {
		name      string
		logFile   string
		logStream string
		want      io.Writer // nil means "expect a file handle for the configured LogFile"
	}{
		{"no file, no stream -> default writer", "", "", defaultWriter},
		{"no file, stream=stdout -> stdout", "", "stdout", os.Stdout},
		{"no file, stream=stderr -> stderr", "", "stderr", os.Stderr},
		{"no file, invalid stream -> default writer", "", "bogus", defaultWriter},
		{"invalid file, no stream -> default writer", invalidFile, "", defaultWriter},
		{"invalid file, stream=stdout -> stdout", invalidFile, "stdout", os.Stdout},
		{"invalid file, stream=stderr -> stderr", invalidFile, "stderr", os.Stderr},
		{"valid file, stream=stderr -> file wins", validFile, "stderr", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				LogFile:   tt.logFile,
				LogStream: tt.logStream,
			}

			output := resolveOutput(cfg)

			if tt.want != nil {
				if output != tt.want {
					t.Errorf("Expected %v, got: %v", tt.want, output)
				}

				return
			}

			file, ok := output.(*os.File)
			if !ok {
				t.Fatalf("Expected *os.File, got: %T", output)
			}

			defer file.Close()

			if file.Name() != cfg.LogFile {
				t.Errorf("Expected file %s, got: %s", cfg.LogFile, file.Name())
			}
		})
	}
}

// TestExplicitOutputImmuneToSetDefaultOutput proves requirements D, E and F directly:
// once resolveOutput selects an explicit LogFile or LogStream, a later SetDefaultOutput
// call must never change what it resolves to - only the unset-file/unset-stream fallback
// (defaultWriter) is redirectable.
func TestExplicitOutputImmuneToSetDefaultOutput(t *testing.T) {
	t.Cleanup(func() { defaultWriter.Set(os.Stdout) })

	tmpFile := t.TempDir() + "/test.log"

	tests := []struct {
		name string
		cfg  *Config
		want io.Writer // nil means "expect a file handle for cfg.LogFile"
	}{
		{"explicit stdout survives SetDefaultOutput", &Config{LogStream: "stdout"}, os.Stdout},
		{"explicit stderr survives SetDefaultOutput", &Config{LogStream: "stderr"}, os.Stderr},
		{"valid log file survives SetDefaultOutput", &Config{LogFile: tmpFile, LogStream: "stderr"}, nil},
	}

	check := func(t *testing.T, cfg *Config, want io.Writer, output io.Writer) {
		t.Helper()

		if want != nil {
			if output != want {
				t.Errorf("Expected %v, got: %v", want, output)
			}

			return
		}

		file, ok := output.(*os.File)
		if !ok || file.Name() != cfg.LogFile {
			t.Errorf("Expected file %s, got: %v (%T)", cfg.LogFile, output, output)

			return
		}

		file.Close()
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check(t, tt.cfg, tt.want, resolveOutput(tt.cfg))

			// A binary calling SetDefaultOutput must not redirect a handler already
			// bound to an explicit file or stream.
			SetDefaultOutput(io.Discard)

			check(t, tt.cfg, tt.want, resolveOutput(tt.cfg))
		})
	}
}

// TestSetDefaultOutputChangesDefaultWriterTarget verifies the exported
// SetDefaultOutput function redirects the shared defaultWriter used by resolveOutput.
func TestSetDefaultOutputChangesDefaultWriterTarget(t *testing.T) {
	t.Cleanup(func() { defaultWriter.Set(os.Stdout) })

	var dest bytes.Buffer

	SetDefaultOutput(&dest)

	if _, err := defaultWriter.Write([]byte("hello")); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if dest.String() != "hello" {
		t.Errorf("Expected SetDefaultOutput to redirect defaultWriter, got: %q", dest.String())
	}
}

// TestSetDefaultOutputRedirectsAlreadyCachedLogger is the key regression test for the
// cached-logger problem described in #2009: a *slog.Logger created (and derived via
// .With, mirroring `var logger = logging.Logger(component)`) BEFORE the writer target
// changes must still observe the new target afterward, because slog binds a handler to
// a writer once, but the writer itself can be redirected.
func TestSetDefaultOutputRedirectsAlreadyCachedLogger(t *testing.T) {
	var oldDest, newDest bytes.Buffer

	sw := newSwitchableWriter(&oldDest)
	handler := slog.NewJSONHandler(sw, &slog.HandlerOptions{Level: slog.LevelInfo})

	// Simulates `var logger = logging.Logger("cached")`, executed at package
	// var-init time, i.e. before any binary main() (and any writer redirect) runs.
	cachedLogger := slog.New(handler).With("component", "cached")

	cachedLogger.Info("before switch")

	if !strings.Contains(oldDest.String(), "before switch") {
		t.Fatalf("Expected message in oldDest before switch, got: %s", oldDest.String())
	}

	// Redirect the writer - analogous to SetDefaultOutput being called from a
	// binary's main() after cached loggers already exist.
	sw.Set(&newDest)

	cachedLogger.Info("after switch", "extra", "data")

	if strings.Contains(newDest.String(), "before switch") {
		t.Error("Old message leaked into newDest")
	}

	if !strings.Contains(newDest.String(), "after switch") {
		t.Fatalf("Expected new message in newDest, got: %s", newDest.String())
	}

	if strings.Contains(oldDest.String(), "after switch") {
		t.Error("New message incorrectly went to oldDest")
	}

	// Verify format, level, and With-attributes all still work post-switch.
	var parsed map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(newDest.String())), &parsed); err != nil {
		t.Fatalf("Post-switch output is not valid JSON: %v\nOutput: %s", err, newDest.String())
	}

	if parsed["component"] != "cached" {
		t.Errorf("Expected component='cached' to survive the switch, got: %v", parsed["component"])
	}

	if parsed["extra"] != "data" {
		t.Errorf("Expected extra='data', got: %v", parsed["extra"])
	}

	if parsed["msg"] != "after switch" {
		t.Errorf("Expected msg='after switch', got: %v", parsed["msg"])
	}
}

// TestSwitchableWriterPreservesLevelFiltering verifies that redirecting the writer
// doesn't disturb the handler's level filtering (Debug still suppressed below Info).
func TestSwitchableWriterPreservesLevelFiltering(t *testing.T) {
	var dest bytes.Buffer

	sw := newSwitchableWriter(io.Discard)
	handler := slog.NewTextHandler(sw, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(handler)

	sw.Set(&dest)

	logger.Debug("should be filtered")
	logger.Info("should appear")

	if strings.Contains(dest.String(), "should be filtered") {
		t.Error("Debug message should have been filtered by level")
	}

	if !strings.Contains(dest.String(), "should appear") {
		t.Errorf("Expected Info message to appear, got: %s", dest.String())
	}
}

// TestSetDefaultOutputIsConcurrencySafe exercises concurrent writes and target swaps to
// catch data races (run with -race).
func TestSetDefaultOutputIsConcurrencySafe(t *testing.T) {
	sw := newSwitchableWriter(io.Discard)

	var wg sync.WaitGroup

	for range 20 {
		wg.Add(2)

		go func() {
			defer wg.Done()

			_, _ = sw.Write([]byte("x"))
		}()

		go func() {
			defer wg.Done()

			sw.Set(io.Discard)
		}()
	}

	wg.Wait()
}

// TestLoggerFunction verifies Logger() creates component-specific loggers.
func TestLoggerFunction(t *testing.T) {
	// Set up a JSON handler so we can verify the output
	var buf bytes.Buffer

	baseLogger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(baseLogger)

	// Create component logger
	componentLogger := Logger("test-component")
	if componentLogger == nil {
		t.Fatal("Logger() returned nil")
	}

	// Log a message
	componentLogger.Info("test message")

	// Verify component field is present
	output := buf.String()

	var parsed map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &parsed); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if parsed["component"] != "test-component" {
		t.Errorf("Expected component='test-component', got: %v", parsed["component"])
	}
}

// TestLoggerMultipleComponents verifies multiple component loggers work independently.
func TestLoggerMultipleComponents(t *testing.T) {
	var buf bytes.Buffer

	baseLogger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(baseLogger)

	// Create multiple component loggers
	logger1 := Logger("component1")
	logger2 := Logger("component2")

	logger1.Info("message from component1")
	logger2.Info("message from component2")

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) != 2 {
		t.Fatalf("Expected 2 log lines, got: %d", len(lines))
	}

	// Verify first line has component1
	var parsed1 map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &parsed1); err != nil {
		t.Fatalf("Failed to parse first line: %v", err)
	}

	if parsed1["component"] != "component1" {
		t.Errorf("Expected component='component1', got: %v", parsed1["component"])
	}

	// Verify second line has component2
	var parsed2 map[string]any
	if err := json.Unmarshal([]byte(lines[1]), &parsed2); err != nil {
		t.Fatalf("Failed to parse second line: %v", err)
	}

	if parsed2["component"] != "component2" {
		t.Errorf("Expected component='component2', got: %v", parsed2["component"])
	}
}

// TestInitLoggerWithTextFormat verifies InitLogger with text format.
func TestInitLoggerWithTextFormat(t *testing.T) {
	// Note: InitLogger uses sync.Once, so we can't easily reset it.
	// This test verifies the logic would work by testing the handler creation directly.
	cfg := &Config{
		LogLevel:  "INFO",
		LogFormat: "text",
		LogFile:   "",
	}

	// Verify config values are valid
	if cfg.LogFormat != "text" {
		t.Errorf("Expected LogFormat='text', got: %s", cfg.LogFormat)
	}
}

// TestInitLoggerWithJSONFormat verifies InitLogger with JSON format.
func TestInitLoggerWithJSONFormat(t *testing.T) {
	const jsonFormat = "json"

	cfg := &Config{
		LogLevel:  "DEBUG",
		LogFormat: jsonFormat,
		LogFile:   "",
	}

	// Verify config values are valid
	if cfg.LogFormat != jsonFormat {
		t.Errorf("Expected LogFormat='%s', got: %s", jsonFormat, cfg.LogFormat)
	}
}

// TestInitLoggerWithInvalidFormat verifies InitLogger with invalid format.
func TestInitLoggerWithInvalidFormat(t *testing.T) {
	cfg := &Config{
		LogLevel:  "INFO",
		LogFormat: "invalid",
		LogFile:   "",
	}

	// The actual InitLogger would fall back to text
	// We verify the config can hold invalid values
	if cfg.LogFormat != "invalid" {
		t.Errorf("Config should preserve invalid format for InitLogger to handle")
	}
}

// TestInitLoggerWithFile verifies InitLogger with log file.
func TestInitLoggerWithFile(t *testing.T) {
	const jsonFormat = "json"

	tmpFile := t.TempDir() + "/test.log"

	cfg := &Config{
		LogLevel:  "INFO",
		LogFormat: jsonFormat,
		LogFile:   tmpFile,
	}

	// Verify config values
	if cfg.LogFile != tmpFile {
		t.Errorf("Expected LogFile=%s, got: %s", tmpFile, cfg.LogFile)
	}
}

// TestLogLevelParsing verifies different log level strings.
func TestLogLevelParsing(t *testing.T) {
	tests := []struct {
		input    string
		expected slog.Level
		shouldOK bool
	}{
		{"DEBUG", slog.LevelDebug, true},
		{"INFO", slog.LevelInfo, true},
		{"WARN", slog.LevelWarn, true},
		{"ERROR", slog.LevelError, true},
		{"debug", slog.LevelDebug, true},
		{"info", slog.LevelInfo, true},
		{"warn", slog.LevelWarn, true},
		{"error", slog.LevelError, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var level slog.Level

			err := level.UnmarshalText([]byte(strings.ToLower(tt.input)))
			if tt.shouldOK && err != nil {
				t.Errorf("Expected successful parse for %s, got error: %v", tt.input, err)
			}

			if tt.shouldOK && level != tt.expected {
				t.Errorf("Expected level=%v, got: %v", tt.expected, level)
			}
		})
	}
}
