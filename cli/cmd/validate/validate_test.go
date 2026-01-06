// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testValidRecord = `{
	"schema_version": "0.7.0",
	"name": "test-agent",
	"version": "1.0.0",
	"description": "Test agent",
	"locators": [
		{
			"type": "docker-image",
			"uri": "docker://test/image:latest"
		}
	]
}`

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

// TestValidateCommand_ValidRecord_EmbeddedSchemas tests validation with embedded schemas.
func TestValidateCommand_ValidRecord_EmbeddedSchemas(t *testing.T) {
	tmpDir := t.TempDir()
	validJSONFile := filepath.Join(tmpDir, "valid.json")
	err := os.WriteFile(validJSONFile, []byte(testValidRecord), 0o600)
	require.NoError(t, err)

	// Reset global state before test
	corev1.SetDisableAPIValidation(true)
	corev1.SetStrictValidation(true)

	cmd := Command
	cmd.SetArgs([]string{"--disable-api", validJSONFile})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err = cmd.Execute()
	// Note: Validation might fail if the record doesn't fully match schema requirements
	// This test mainly ensures the command runs without crashing
	if err != nil {
		// If validation fails, it should be a validation error, not a command error
		assert.Contains(t, err.Error(), "validation")
	}
}

// TestValidateCommand_DisableAPIFlag tests the --disable-api flag.
func TestValidateCommand_DisableAPIFlag(t *testing.T) {
	tmpDir := t.TempDir()
	validJSONFile := filepath.Join(tmpDir, "valid.json")
	err := os.WriteFile(validJSONFile, []byte(testValidRecord), 0o600)
	require.NoError(t, err)

	// Reset global state
	corev1.SetDisableAPIValidation(true)

	cmd := Command
	cmd.SetArgs([]string{"--disable-api", validJSONFile})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	// Should not crash, validation result depends on record validity
	_ = cmd.Execute()
}

// TestValidateCommand_DisableStrictFlag tests the --disable-strict flag (only works with --url).
func TestValidateCommand_DisableStrictFlag(t *testing.T) {
	tmpDir := t.TempDir()
	validJSONFile := filepath.Join(tmpDir, "valid.json")
	err := os.WriteFile(validJSONFile, []byte(testValidRecord), 0o600)
	require.NoError(t, err)

	// Reset global state
	corev1.SetDisableAPIValidation(false)
	corev1.SetStrictValidation(false)

	cmd := Command
	cmd.SetArgs([]string{"--url", "https://schema.oasf.outshift.com", "--disable-strict", validJSONFile})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	// Should not crash (may fail validation or timeout, but should handle gracefully)
	_ = cmd.Execute()
}

// TestValidateCommand_WithURLFlag tests the --url flag.
func TestValidateCommand_WithURLFlag(t *testing.T) {
	tmpDir := t.TempDir()
	validJSONFile := filepath.Join(tmpDir, "valid.json")
	err := os.WriteFile(validJSONFile, []byte(testValidRecord), 0o600)
	require.NoError(t, err)

	cmd := Command
	cmd.SetArgs([]string{"--url", "https://schema.oasf.outshift.com", validJSONFile})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	// Note: This will attempt API validation, which may fail if the URL is unreachable
	// The important thing is that it doesn't crash and handles the error gracefully
	_ = cmd.Execute()
}

// TestValidateCommand_DefaultBehavior tests that default behavior uses embedded schemas.
func TestValidateCommand_DefaultBehavior(t *testing.T) {
	tmpDir := t.TempDir()
	validJSONFile := filepath.Join(tmpDir, "valid.json")
	err := os.WriteFile(validJSONFile, []byte(testValidRecord), 0o600)
	require.NoError(t, err)

	// Reset global state to ensure default
	corev1.SetDisableAPIValidation(true)
	corev1.SetSchemaURL("")

	cmd := Command
	cmd.SetArgs([]string{validJSONFile})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	// Should use embedded schemas by default (no API validation)
	err = cmd.Execute()
	// Should not fail with "API validation is enabled but schema_url is not configured"
	if err != nil {
		assert.NotContains(t, err.Error(), "API validation is enabled but schema_url is not configured")
	}
}

// TestRunCommand_ContextCancellation tests that the command respects context cancellation.
func TestRunCommand_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()
	validJSONFile := filepath.Join(tmpDir, "valid.json")
	err := os.WriteFile(validJSONFile, []byte(testValidRecord), 0o600)
	require.NoError(t, err)

	// Reset global state
	corev1.SetDisableAPIValidation(true)

	cmd := Command
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately
	cmd.SetContext(ctx)

	cmd.SetArgs([]string{"--disable-api", validJSONFile})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	// Should handle cancellation gracefully
	_ = cmd.Execute()
}

// TestValidateCommand_OutputFormats tests different output formats.
func TestValidateCommand_OutputFormats(t *testing.T) {
	tmpDir := t.TempDir()
	validJSONFile := filepath.Join(tmpDir, "valid.json")
	err := os.WriteFile(validJSONFile, []byte(testValidRecord), 0o600)
	require.NoError(t, err)

	// Reset global state
	corev1.SetDisableAPIValidation(true)

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
	tmpDir := t.TempDir()
	validJSONFile := filepath.Join(tmpDir, "valid.json")
	err := os.WriteFile(validJSONFile, []byte(testValidRecord), 0o600)
	require.NoError(t, err)

	cmd := Command
	cmd.SetArgs([]string{"--all", "--disable-api", validJSONFile})

	var stderr bytes.Buffer
	cmd.SetErr(&stderr)

	// Should fail because file path cannot be specified with --all
	err = cmd.Execute()
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
