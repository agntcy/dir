// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"bytes"
	_ "embed"
	"os"
	"path/filepath"
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Embedded test data files
//
//go:embed testdata/record_invalid.json
var testRecordInvalid []byte

//go:embed testdata/record_valid.json
var testRecordValid []byte

//go:embed testdata/record_valid_for_schema.json
var testRecordValidForSchema []byte

//go:embed testdata/record_valid_for_non_strict.json
var testRecordValidForNonStrict []byte

// TestValidateCommand_NoArgs tests that the command requires a file path.
func TestValidateCommand_NoArgs(t *testing.T) {
	cmd := Command
	cmd.SetArgs([]string{})

	var stderr bytes.Buffer
	cmd.SetErr(&stderr)

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "file path is required")
	assert.Contains(t, err.Error(), "Usage: dirctl validate <file>")
}

// TestValidateCommand_TooManyArgs tests that the command only accepts one file path.
func TestValidateCommand_TooManyArgs(t *testing.T) {
	cmd := Command
	cmd.SetArgs([]string{"file1.json", "file2.json"})

	var stderr bytes.Buffer
	cmd.SetErr(&stderr)

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "only one file path is allowed")
}

// TestValidateCommand_FileNotFound tests error handling for non-existent files.
func TestValidateCommand_FileNotFound(t *testing.T) {
	cmd := Command
	cmd.SetArgs([]string{"nonexistent.json"})

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

	cmd := Command
	cmd.SetArgs([]string{invalidJSONFile})

	var stderr bytes.Buffer
	cmd.SetErr(&stderr)

	err = cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse record JSON")
}

// getTestRecordFile creates a temporary file with a real test record for testing.
func getTestRecordFile(t *testing.T) string {
	t.Helper()

	// Use embedded record_valid.json as a valid test record
	tmpDir := t.TempDir()
	validJSONFile := filepath.Join(tmpDir, "record.json")
	err := os.WriteFile(validJSONFile, testRecordValid, 0o600)
	require.NoError(t, err)

	return validJSONFile
}

// TestValidateCommand_OutputFormats tests different output formats.
func TestValidateCommand_OutputFormats(t *testing.T) {
	validJSONFile := getTestRecordFile(t)
	if validJSONFile == "" {
		return
	}

	// Reset opts to ensure clean state
	opts.ValidateAll = false
	opts.SchemaURL = ""
	opts.DisableAPI = false
	opts.DisableStrict = false

	tests := []struct {
		name   string
		args   []string
		output string
	}{
		{
			name:   "human format (default)",
			args:   []string{"--disable-api", validJSONFile},
			output: "human",
		},
		{
			name:   "json format",
			args:   []string{"--disable-api", "--output", "json", validJSONFile},
			output: "json",
		},
		{
			name:   "raw format",
			args:   []string{"--disable-api", "--output", "raw", validJSONFile},
			output: "raw",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := Command
			cmd.SetArgs(tt.args)

			var stdout bytes.Buffer
			cmd.SetOut(&stdout)

			// Should not crash regardless of output format
			_ = cmd.Execute()
		})
	}
}

// TestValidateCommand_CommandInitialization tests that the command is properly initialized.
func TestValidateCommand_CommandInitialization(t *testing.T) {
	assert.NotNil(t, Command)
	assert.Contains(t, Command.Use, "validate")
	assert.NotEmpty(t, Command.Short)
	assert.Contains(t, Command.Short, "local")
	assert.NotEmpty(t, Command.Long)
	assert.Contains(t, Command.Long, "local validation")
	assert.NotNil(t, Command.RunE)

	// Check that flags are registered
	flags := Command.Flags()
	assert.NotNil(t, flags.Lookup("disable-api"))
	assert.NotNil(t, flags.Lookup("disable-strict"))
	assert.NotNil(t, flags.Lookup("url"))
	assert.NotNil(t, flags.Lookup("output"))
}

// TestValidateCommand_AllRequiresFlag tests that --all also requires a flag.
func TestValidateCommand_AllRequiresFlag(t *testing.T) {
	// Reset opts to ensure clean state
	opts.ValidateAll = true
	opts.SchemaURL = ""
	opts.DisableAPI = false

	cmd := Command
	cmd.SetArgs([]string{"--all"})

	var stderr bytes.Buffer
	cmd.SetErr(&stderr)

	// Should fail because neither --url nor --disable-api was provided
	// The flag check happens first in runValidateAllCommand
	err := cmd.Execute()
	require.Error(t, err)
	// The error should be about missing flags (checked before client connection)
	assert.Contains(t, err.Error(), "either --url or --disable-api flag must be specified when using --all")
}

// TestValidateCommand_AllWithFile tests that --all cannot be used with a file path.
func TestValidateCommand_AllWithFile(t *testing.T) {
	validJSONFile := getTestRecordFile(t)
	if validJSONFile == "" {
		return
	}

	// Reset opts to ensure clean state
	opts.ValidateAll = false
	opts.SchemaURL = ""
	opts.DisableAPI = false
	opts.DisableStrict = false

	cmd := Command
	cmd.SetArgs([]string{"--all", "--disable-api", validJSONFile})

	var stderr bytes.Buffer
	cmd.SetErr(&stderr)

	// Should fail because file path cannot be specified with --all
	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "file path cannot be specified when using --all flag")
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

	// Reset global state and opts
	corev1.SetDisableAPIValidation(true)

	opts.ValidateAll = false // Ensure --all is not set

	cmd := Command
	cmd.SetArgs([]string{"--disable-api", invalidJSONFile})

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

// TestValidateCommand_RealFiles tests validation of actual testdata files with different validation settings.
func TestValidateCommand_RealFiles(t *testing.T) {
	tests := []struct {
		name           string
		fileData       []byte
		fileName       string
		args           []string
		expectValid    bool
		expectErrorMsg string // If empty, no specific error message expected
	}{
		// record_invalid.json - should fail with all 3 validation settings
		{
			name:        "record_invalid_API_strict",
			fileData:    testRecordInvalid,
			fileName:    "record_invalid.json",
			args:        []string{"--url", "https://schema.oasf.outshift.com/"},
			expectValid: false,
		},
		{
			name:        "record_invalid_API_non_strict",
			fileData:    testRecordInvalid,
			fileName:    "record_invalid.json",
			args:        []string{"--url", "https://schema.oasf.outshift.com/", "--disable-strict"},
			expectValid: false,
		},
		{
			name:        "record_invalid_embedded",
			fileData:    testRecordInvalid,
			fileName:    "record_invalid.json",
			args:        []string{"--disable-api"},
			expectValid: false,
		},
		// record_valid.json - should be valid with all 3 validation settings
		{
			name:        "record_valid_API_strict",
			fileData:    testRecordValid,
			fileName:    "record_valid.json",
			args:        []string{"--url", "https://schema.oasf.outshift.com/"},
			expectValid: true,
		},
		{
			name:        "record_valid_API_non_strict",
			fileData:    testRecordValid,
			fileName:    "record_valid.json",
			args:        []string{"--url", "https://schema.oasf.outshift.com/", "--disable-strict"},
			expectValid: true,
		},
		{
			name:        "record_valid_embedded",
			fileData:    testRecordValid,
			fileName:    "record_valid.json",
			args:        []string{"--disable-api"},
			expectValid: true,
		},
		// record_valid_for_schema.json - should only be valid if --disable-api
		{
			name:        "record_valid_for_schema_API_strict",
			fileData:    testRecordValidForSchema,
			fileName:    "record_valid_for_schema.json",
			args:        []string{"--url", "https://schema.oasf.outshift.com/"},
			expectValid: false,
		},
		{
			name:        "record_valid_for_schema_API_non_strict",
			fileData:    testRecordValidForSchema,
			fileName:    "record_valid_for_schema.json",
			args:        []string{"--url", "https://schema.oasf.outshift.com/", "--disable-strict"},
			expectValid: true,
		},
		{
			name:        "record_valid_for_schema_embedded",
			fileData:    testRecordValidForSchema,
			fileName:    "record_valid_for_schema.json",
			args:        []string{"--disable-api"},
			expectValid: true,
		},
		// record_valid_for_non_strict.json - should be valid for both --disable-api and --disable-strict
		{
			name:        "record_valid_for_non_strict_API_strict",
			fileData:    testRecordValidForNonStrict,
			fileName:    "record_valid_for_non_strict.json",
			args:        []string{"--url", "https://schema.oasf.outshift.com/"},
			expectValid: false,
		},
		{
			name:        "record_valid_for_non_strict_API_non_strict",
			fileData:    testRecordValidForNonStrict,
			fileName:    "record_valid_for_non_strict.json",
			args:        []string{"--url", "https://schema.oasf.outshift.com/", "--disable-strict"},
			expectValid: true,
		},
		{
			name:        "record_valid_for_non_strict_embedded",
			fileData:    testRecordValidForNonStrict,
			fileName:    "record_valid_for_non_strict.json",
			args:        []string{"--disable-api"},
			expectValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset opts to ensure clean state between tests
			opts.ValidateAll = false
			opts.SchemaURL = ""
			opts.DisableAPI = false
			opts.DisableStrict = false

			// Create a temporary file with the embedded testdata content
			tmpDir := t.TempDir()
			filePath := filepath.Join(tmpDir, tt.fileName)

			err := os.WriteFile(filePath, tt.fileData, 0o600)
			require.NoError(t, err)

			// Build command args (create new slice to avoid modifying original)
			args := make([]string, len(tt.args), len(tt.args)+1)
			copy(args, tt.args)
			args = append(args, filePath)

			cmd := Command
			cmd.SetArgs(args)

			var stdout bytes.Buffer

			var stderr bytes.Buffer

			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)

			// Execute the command
			err = cmd.Execute()

			if tt.expectValid {
				// Expect validation to succeed
				assert.NoError(t, err)
			} else {
				// Expect validation to fail
				require.Error(t, err, "Expected validation to fail")

				if tt.expectErrorMsg != "" {
					assert.Contains(t, err.Error(), tt.expectErrorMsg)
				}
			}
		})
	}
}
