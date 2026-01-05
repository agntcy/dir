// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package validate

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/cli/presenter"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "validate <file>",
	Short: "Validate an OASF record JSON file locally",
	Long: `Validate an OASF record JSON file against the OASF schema locally.

This command performs local validation of OASF records without requiring
a connection to a Directory server. You must specify either --url for
API-based validation or --disable-api for embedded schema validation.

Usage examples:

1. Validate using embedded schemas (no API calls):
   dirctl validate record.json --disable-api

2. Validate with API-based validation using a custom schema URL:
   dirctl validate record.json --url https://schema.oasf.outshift.com

3. Validate with non-strict mode (more permissive, only works with --url):
   dirctl validate record.json --url https://schema.oasf.outshift.com --disable-strict

Note: You must specify either --url (for API validation) or --disable-api
(for embedded schema validation). This command is intended for local
validation purposes.
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("file path is required\n\nUsage: dirctl validate <file>")
		}
		if len(args) > 1 {
			return errors.New("only one file path is allowed")
		}

		return runCommand(cmd, args[0])
	},
}

func runCommand(cmd *cobra.Command, filePath string) error {
	// Read the JSON file
	jsonData, err := os.ReadFile(filepath.Clean(filePath))
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Unmarshal the JSON into a Record
	record, err := corev1.UnmarshalRecord(jsonData)
	if err != nil {
		return fmt.Errorf("failed to parse record JSON: %w", err)
	}

	// Configure validation settings based on flags
	// Note: These are global settings, but they're thread-safe
	// Require either --url or --disable-api to be explicitly set
	switch {
	case opts.SchemaURL != "":
		// API validation enabled with provided URL
		corev1.SetDisableAPIValidation(false)
		corev1.SetSchemaURL(opts.SchemaURL)
	case opts.DisableAPI:
		// Explicitly disable API validation (use embedded schemas)
		corev1.SetDisableAPIValidation(true)
	default:
		// Neither --url nor --disable-api was provided
		return errors.New("either --url or --disable-api flag must be specified")
	}

	// Configure strict validation (only applies to API validation)
	// Note: --disable-strict only works with --url (API validation)
	if opts.SchemaURL != "" {
		if opts.DisableStrict {
			corev1.SetStrictValidation(false)
		} else {
			corev1.SetStrictValidation(true)
		}
	}
	// For embedded schemas (--disable-api), strict validation setting is ignored

	// Validate the record
	ctx := cmd.Context()

	valid, validationErrors, err := record.Validate(ctx)
	if err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	// Output results
	if !valid {
		return outputValidationErrors(cmd, validationErrors)
	}

	return outputValidationSuccess(cmd, record)
}

func outputValidationSuccess(cmd *cobra.Command, record *corev1.Record) error {
	schemaVersion := record.GetSchemaVersion()
	opts := presenter.GetOutputOptions(cmd)

	if opts.IsStructuredOutput() {
		// For structured output, use PrintMessage
		if schemaVersion != "" {
			return presenter.PrintMessage(cmd, "validation", "Record is valid", fmt.Sprintf("(schema version: %s)", schemaVersion))
		}

		return presenter.PrintMessage(cmd, "validation", "Record is valid", "")
	}

	// For human-readable output, print without colon
	if schemaVersion != "" {
		presenter.Printf(cmd, "Record is valid (schema version: %s)\n", schemaVersion)
	} else {
		presenter.Printf(cmd, "Record is valid\n")
	}

	return nil
}

func outputValidationErrors(cmd *cobra.Command, validationErrors []string) error {
	if len(validationErrors) > 0 {
		presenter.Printf(cmd, "Validation failed with %d error(s):\n", len(validationErrors))

		for i, errMsg := range validationErrors {
			presenter.Printf(cmd, "  %d. %s\n", i+1, errMsg)
		}

		return errors.New("record validation failed")
	}

	return errors.New("record validation failed (no error details available)")
}
