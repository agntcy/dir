// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"bytes"
	"context"
	_ "embed"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Embedded test data files
//
//go:embed testdata/record_invalid.json
var testRecordInvalid []byte

//go:embed testdata/record_valid.json
var testRecordValid []byte

//go:embed testdata/record_valid_with_warnings.json
var testRecordValidWithWarnings []byte

// TestValidateCommand_NoFileArgs tests that the command reads from stdin when no file path is provided.
// Note: --url flag is still required for validation.
func TestValidateCommand_NoFileArgs(t *testing.T) {
	// Reset opts to ensure clean state - cobra will populate it during Execute()
	opts.SchemaURL = ""

	cmd := Command
	cmd.SetArgs([]string{"--url", "https://schema.oasf.outshift.com"})
	cmd.SetContext(context.Background())

	// Provide JSON via stdin (no file path argument)
	cmd.SetIn(bytes.NewReader(testRecordValid))

	var stdout bytes.Buffer

	var stderr bytes.Buffer

	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	// Should succeed with valid schema URL
	err := cmd.Execute()
	require.NoError(t, err, "Command should succeed. stdout: %q, stderr: %q", stdout.String(), stderr.String())

	// Verify output contains success message
	output := stdout.String() + stderr.String()
	assert.Contains(t, output, "Record is valid", "Should show validation success message")
}

// TestValidateCommand_TooManyArgs tests that the command only accepts one file path.
func TestValidateCommand_TooManyArgs(t *testing.T) {
	// Reset opts to ensure clean state
	opts.SchemaURL = ""

	cmd := Command
	cmd.SetArgs([]string{"--url", "https://schema.oasf.outshift.com", "file1.json", "file2.json"})
	cmd.SetContext(context.Background())

	var stderr bytes.Buffer
	cmd.SetErr(&stderr)

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "only one file path is allowed")
}

// TestValidateCommand_FileNotFound tests error handling for non-existent files.
func TestValidateCommand_FileNotFound(t *testing.T) {
	// Reset opts to ensure clean state
	opts.SchemaURL = ""

	cmd := Command
	cmd.SetArgs([]string{"--url", "https://schema.oasf.outshift.com", "nonexistent.json"})
	cmd.SetContext(context.Background())

	var stderr bytes.Buffer
	cmd.SetErr(&stderr)

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read file")
}

// TestValidateCommand_InvalidJSON tests error handling for invalid JSON.
func TestValidateCommand_InvalidJSON(t *testing.T) {
	// Create a temporary file with invalid JSON
	tmpDir := t.TempDir()
	invalidJSONFile := filepath.Join(tmpDir, "invalid.json")
	err := os.WriteFile(invalidJSONFile, []byte("{ invalid json }"), 0o600)
	require.NoError(t, err)

	// Reset opts to ensure clean state
	opts.SchemaURL = ""

	cmd := Command
	cmd.SetArgs([]string{"--url", "https://schema.oasf.outshift.com", invalidJSONFile})
	cmd.SetContext(context.Background())

	var stderr bytes.Buffer
	cmd.SetErr(&stderr)

	err = cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse record JSON")
}

// TestValidateCommand_CommandInitialization tests that the command is properly initialized.
func TestValidateCommand_CommandInitialization(t *testing.T) {
	assert.NotNil(t, Command)
	assert.Contains(t, Command.Use, "validate")
	assert.NotEmpty(t, Command.Short)
	assert.Contains(t, Command.Short, "file or stdin")
	assert.NotEmpty(t, Command.Long)
	assert.Contains(t, Command.Long, "piped from stdin")
	assert.NotNil(t, Command.RunE)

	// Check that flags are registered
	flags := Command.Flags()
	assert.NotNil(t, flags.Lookup("url"))
}

// TestValidateCommand_EmptySchemaURL tests that the command returns an error when schema URL is empty.
func TestValidateCommand_EmptySchemaURL(t *testing.T) {
	// Reset opts to ensure clean state
	opts.SchemaURL = ""

	cmd := Command
	cmd.SetArgs([]string{})
	cmd.SetContext(context.Background())

	// Provide JSON via stdin
	cmd.SetIn(bytes.NewReader(testRecordValid))

	var stderr bytes.Buffer
	cmd.SetErr(&stderr)

	// Should fail because empty schema URL is not allowed
	err := cmd.Execute()
	require.Error(t, err)
	// Check for either the schema URL required message or validator initialization failure
	assert.True(t,
		strings.Contains(err.Error(), "schema URL is required") ||
			strings.Contains(err.Error(), "failed to initialize validator"),
		"Error should mention schema URL requirement or validator initialization: %s", err.Error())
}

// TestValidateCommand_Stdin tests that the command can read from stdin.
func TestValidateCommand_Stdin(t *testing.T) {
	// Reset opts to ensure clean state
	opts.SchemaURL = ""

	cmd := Command
	cmd.SetArgs([]string{"--url", "https://schema.oasf.outshift.com"})
	cmd.SetContext(context.Background())

	// Provide JSON via stdin - need to ensure the reader is positioned correctly
	stdinReader := bytes.NewReader(testRecordValid)
	cmd.SetIn(stdinReader)

	var stdout bytes.Buffer

	var stderr bytes.Buffer

	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.Execute()
	require.NoError(t, err)

	// Check both stdout and stderr for validation success message
	output := stdout.String() + stderr.String()
	assert.True(t, strings.Contains(output, "Record is valid") || strings.Contains(output, "valid") || strings.Contains(output, "schema version"), "Output should indicate validation success: stdout=%q stderr=%q", stdout.String(), stderr.String())
}

// TestRunCommand_InvalidRecordStructure tests handling of records with invalid structure.
func TestRunCommand_InvalidRecordStructure(t *testing.T) {
	// Create a file with valid JSON but invalid OASF structure (missing schema_version)
	invalidRecord := `{
		"not": "a valid oasf record",
		"missing": "required fields"
	}`

	tmpDir := t.TempDir()
	invalidJSONFile := filepath.Join(tmpDir, "invalid.json")
	err := os.WriteFile(invalidJSONFile, []byte(invalidRecord), 0o600)
	require.NoError(t, err)

	// Reset opts
	opts.SchemaURL = ""

	cmd := Command
	cmd.SetArgs([]string{"--url", "https://schema.oasf.outshift.com", invalidJSONFile})
	cmd.SetContext(context.Background())

	var stdout bytes.Buffer

	var stderr bytes.Buffer

	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err = cmd.Execute()
	// Should fail with parsing/decoding errors (happens before validation)
	require.Error(t, err)
	// Should show error about failed to parse or decode
	output := stdout.String() + stderr.String()
	hasParseError := assert.Contains(t, output, "failed to parse") ||
		assert.Contains(t, output, "failed to decode") ||
		assert.Contains(t, output, "schema_version")
	assert.True(t, hasParseError, "Error output should mention parsing/decoding issue: %s", output)
}

// TestValidateCommand_RealFiles tests validation of actual testdata files.
func TestValidateCommand_RealFiles(t *testing.T) {
	tests := []struct {
		name              string
		fileData          []byte
		fileName          string
		expectValid       bool
		expectErrorOutput bool // If true, expects error output
		expectWarnings    bool // If true, expects warning messages in output
		expectEmptyOutput bool // If true, expects no validation messages (just success)
	}{
		{
			name:              "record_invalid should fail validation and return error",
			fileData:          testRecordInvalid,
			fileName:          "record_invalid.json",
			expectValid:       false,
			expectErrorOutput: true,
			expectWarnings:    false,
			expectEmptyOutput: false,
		},
		{
			name:              "record_valid should be valid and return empty string",
			fileData:          testRecordValid,
			fileName:          "record_valid.json",
			expectValid:       true,
			expectErrorOutput: false,
			expectWarnings:    false,
			expectEmptyOutput: true,
		},
		{
			name:              "record_valid_with_warnings should be valid but return warning messages",
			fileData:          testRecordValidWithWarnings,
			fileName:          "record_valid_with_warnings.json",
			expectValid:       true,
			expectErrorOutput: false,
			expectWarnings:    true,
			expectEmptyOutput: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary file with the embedded testdata content
			tmpDir := t.TempDir()
			filePath := filepath.Join(tmpDir, tt.fileName)

			err := os.WriteFile(filePath, tt.fileData, 0o600)
			require.NoError(t, err)

			// Reset opts to ensure clean state between tests
			// Note: cobra will populate opts.SchemaURL when parsing --url flag during Execute()
			opts.SchemaURL = ""

			cmd := Command
			cmd.SetArgs([]string{"--url", "https://schema.oasf.outshift.com", filePath})
			cmd.SetContext(context.Background())

			// Verify flag is set before execution (cobra will parse it during Execute)
			// We can't check opts.SchemaURL here because it's only populated during Execute()

			var stdout bytes.Buffer

			var stderr bytes.Buffer

			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)

			// Execute the command
			err = cmd.Execute()

			// Combine stdout and stderr for checking output
			output := stdout.String() + stderr.String()

			if tt.expectValid {
				// Expect validation to succeed (no error returned)
				require.NoError(t, err, "Validation should succeed for %s", tt.fileName)

				if tt.expectEmptyOutput {
					// Should have no validation error/warning messages, just success message
					assert.Contains(t, output, "Record is valid", "Should show validation success message")
					// Should not contain ERROR: or WARNING: prefixes
					assert.NotContains(t, output, "ERROR:", "Should not contain error messages for valid record")
					assert.NotContains(t, output, "WARNING:", "Should not contain warning messages for valid record without warnings")
					assert.NotContains(t, output, "warning(s):", "Should not show warnings section")
				}

				if tt.expectWarnings {
					// Should contain warning messages
					assert.Contains(t, output, "Record is valid", "Should still show validation success")
					assert.Contains(t, output, "WARNING:", "Should contain warning messages in output")
					assert.Contains(t, output, "warning(s):", "Should show warnings count")
				}
			} else {
				// Expect validation to fail (error returned)
				require.Error(t, err, "Expected validation to fail for %s", tt.fileName)

				if tt.expectErrorOutput {
					// Should contain error messages
					assert.Contains(t, output, "ERROR:", "Should contain error messages in output")
					assert.Contains(t, output, "record validation failed", "Should show validation failure message")
					assert.Contains(t, output, "message(s):", "Should show message count")
					// Should not show success message
					assert.NotContains(t, output, "Record is valid", "Should not show success message for invalid record")
				}
			}
		})
	}
}
