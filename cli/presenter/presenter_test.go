// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package presenter

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
)

// Test constants for frequently used values with semantic meaning.
const (
	emptyString = ""
)

func TestPrint(t *testing.T) {
	var buf bytes.Buffer

	cmd := &cobra.Command{}
	cmd.SetOut(&buf)

	Print(cmd, "test message")

	expected := "test message"
	if got := buf.String(); got != expected {
		t.Errorf("Print() = %q, want %q", got, expected)
	}
}

func TestPrintln(t *testing.T) {
	var buf bytes.Buffer

	cmd := &cobra.Command{}
	cmd.SetOut(&buf)

	Println(cmd, "test message")

	expected := "test message\n"
	if got := buf.String(); got != expected {
		t.Errorf("Println() = %q, want %q", got, expected)
	}
}

func TestPrintf(t *testing.T) {
	var buf bytes.Buffer

	cmd := &cobra.Command{}
	cmd.SetOut(&buf)

	Printf(cmd, "hello %s, count: %d", "world", 42)

	expected := "hello world, count: 42"
	if got := buf.String(); got != expected {
		t.Errorf("Printf() = %q, want %q", got, expected)
	}
}

func TestError(t *testing.T) {
	var buf bytes.Buffer

	cmd := &cobra.Command{}
	cmd.SetErr(&buf)

	Error(cmd, "error message")

	expected := "error message"
	if got := buf.String(); got != expected {
		t.Errorf("Error() = %q, want %q", got, expected)
	}
}

func TestErrorf(t *testing.T) {
	var buf bytes.Buffer

	cmd := &cobra.Command{}
	cmd.SetErr(&buf)

	Errorf(cmd, "error: %s (code: %d)", "failed", 500)

	expected := "error: failed (code: 500)"
	if got := buf.String(); got != expected {
		t.Errorf("Errorf() = %q, want %q", got, expected)
	}
}

func TestPrintSmartfHumanFormat(t *testing.T) {
	var (
		stdoutBuf bytes.Buffer
		stderrBuf bytes.Buffer
	)

	cmd := &cobra.Command{}
	cmd.SetOut(&stdoutBuf)
	cmd.SetErr(&stderrBuf)
	AddOutputFlags(cmd)

	// Human format (default) - should write to stdout
	PrintSmartf(cmd, "metadata message\n")

	expected := "metadata message\n"
	if got := stdoutBuf.String(); got != expected {
		t.Errorf("PrintSmartf(human) stdout = %q, want %q", got, expected)
	}

	if got := stderrBuf.String(); got != emptyString {
		t.Errorf("PrintSmartf(human) stderr = %q, want %q", got, emptyString)
	}
}

func TestPrintSmartfJSONFormat(t *testing.T) {
	var (
		stdoutBuf bytes.Buffer
		stderrBuf bytes.Buffer
	)

	cmd := &cobra.Command{}
	cmd.SetOut(&stdoutBuf)
	cmd.SetErr(&stderrBuf)
	AddOutputFlags(cmd)

	// Set JSON format - should write to stderr
	if err := cmd.Flags().Set("output", string(FormatJSON)); err != nil {
		t.Fatalf("failed to set flag: %v", err)
	}

	PrintSmartf(cmd, "metadata message\n")

	expected := "metadata message\n"

	if got := stdoutBuf.String(); got != emptyString {
		t.Errorf("PrintSmartf(json) stdout = %q, want %q", got, emptyString)
	}

	if got := stderrBuf.String(); got != expected {
		t.Errorf("PrintSmartf(json) stderr = %q, want %q", got, expected)
	}
}

func TestPrintSmartfJSONLFormat(t *testing.T) {
	var (
		stdoutBuf bytes.Buffer
		stderrBuf bytes.Buffer
	)

	cmd := &cobra.Command{}
	cmd.SetOut(&stdoutBuf)
	cmd.SetErr(&stderrBuf)
	AddOutputFlags(cmd)

	// Set JSONL format - should write to stderr
	if err := cmd.Flags().Set("output", string(FormatJSONL)); err != nil {
		t.Fatalf("failed to set flag: %v", err)
	}

	PrintSmartf(cmd, "Listening to events...\n")

	expected := "Listening to events...\n"

	if got := stdoutBuf.String(); got != emptyString {
		t.Errorf("PrintSmartf(jsonl) stdout = %q, want %q", got, emptyString)
	}

	if got := stderrBuf.String(); got != expected {
		t.Errorf("PrintSmartf(jsonl) stderr = %q, want %q", got, expected)
	}
}

func TestPrintSmartfRawFormat(t *testing.T) {
	var (
		stdoutBuf bytes.Buffer
		stderrBuf bytes.Buffer
	)

	cmd := &cobra.Command{}
	cmd.SetOut(&stdoutBuf)
	cmd.SetErr(&stderrBuf)
	AddOutputFlags(cmd)

	// Set raw format - should write to stderr
	if err := cmd.Flags().Set("output", string(FormatRaw)); err != nil {
		t.Fatalf("failed to set flag: %v", err)
	}

	PrintSmartf(cmd, "Processing...\n")

	expected := "Processing...\n"

	if got := stdoutBuf.String(); got != emptyString {
		t.Errorf("PrintSmartf(raw) stdout = %q, want %q", got, emptyString)
	}

	if got := stderrBuf.String(); got != expected {
		t.Errorf("PrintSmartf(raw) stderr = %q, want %q", got, expected)
	}
}

func TestPrintSmartfWithFormatting(t *testing.T) {
	tests := []struct {
		name           string
		outputFormat   string
		format         string
		args           []interface{}
		expectedStdout string
		expectedStderr string
	}{
		{
			name:           "human with formatting",
			outputFormat:   string(FormatHuman),
			format:         "Found %d items in %s\n",
			args:           []interface{}{42, "database"},
			expectedStdout: "Found 42 items in database\n",
			expectedStderr: emptyString,
		},
		{
			name:           "json with formatting",
			outputFormat:   string(FormatJSON),
			format:         "Processing item %d of %d\n",
			args:           []interface{}{5, 10},
			expectedStdout: emptyString,
			expectedStderr: "Processing item 5 of 10\n",
		},
		{
			name:           "jsonl with formatting",
			outputFormat:   string(FormatJSONL),
			format:         "CID: %s, Status: %s\n",
			args:           []interface{}{"bafy123", "complete"},
			expectedStdout: emptyString,
			expectedStderr: "CID: bafy123, Status: complete\n",
		},
		{
			name:           "raw with formatting",
			outputFormat:   string(FormatRaw),
			format:         "Sync ID: %s\n",
			args:           []interface{}{"sync-abc-123"},
			expectedStdout: emptyString,
			expectedStderr: "Sync ID: sync-abc-123\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				stdoutBuf bytes.Buffer
				stderrBuf bytes.Buffer
			)

			cmd := &cobra.Command{}
			cmd.SetOut(&stdoutBuf)
			cmd.SetErr(&stderrBuf)
			AddOutputFlags(cmd)

			if tt.outputFormat != string(FormatHuman) {
				if err := cmd.Flags().Set("output", tt.outputFormat); err != nil {
					t.Fatalf("failed to set flag: %v", err)
				}
			}

			PrintSmartf(cmd, tt.format, tt.args...)

			if got := stdoutBuf.String(); got != tt.expectedStdout {
				t.Errorf("PrintSmartf() stdout = %q, want %q", got, tt.expectedStdout)
			}

			if got := stderrBuf.String(); got != tt.expectedStderr {
				t.Errorf("PrintSmartf() stderr = %q, want %q", got, tt.expectedStderr)
			}
		})
	}
}
