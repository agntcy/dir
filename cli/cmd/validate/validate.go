// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package validate

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/cli/presenter"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "validate [<file>]",
	Short: "Validate OASF record JSON from a file or stdin",
	Long: `Validate OASF record JSON against the OASF schema. The JSON can be provided
as a file path or piped from stdin (e.g., from dirctl pull).

You must specify either --url for API-based validation or --disable-api for
embedded schema validation.

Usage examples:

1. Validate a file using embedded schemas (no API calls):
   dirctl validate record.json --disable-api

2. Validate a file with API-based validation:
   dirctl validate record.json --url https://schema.oasf.outshift.com

3. Validate a file with non-strict mode (more permissive, only works with --url):
   dirctl validate record.json --url https://schema.oasf.outshift.com --disable-strict

4. Validate JSON piped from stdin:
   cat record.json | dirctl validate --disable-api

5. Validate a record pulled from directory:
   dirctl pull <cid> --output json | dirctl validate --disable-api

Note: You must specify either --url (for API validation) or --disable-api
(for embedded schema validation).
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var jsonData []byte
		var err error

		if len(args) > 1 {
			return errors.New("only one file path is allowed")
		}

		if len(args) == 0 {
			// Read from stdin
			jsonData, err = io.ReadAll(cmd.InOrStdin())
			if err != nil {
				return fmt.Errorf("failed to read from stdin: %w", err)
			}
		} else {
			// Read from file
			jsonData, err = os.ReadFile(filepath.Clean(args[0]))
			if err != nil {
				return fmt.Errorf("failed to read file: %w", err)
			}
		}

		return runCommand(cmd, jsonData)
	},
}

func runCommand(cmd *cobra.Command, jsonData []byte) error {
	// Unmarshal the JSON into a Record
	record, err := corev1.UnmarshalRecord(jsonData)
	if err != nil {
		return fmt.Errorf("failed to parse record JSON: %w", err)
	}

	// Configure validation settings
	configureValidationSettings()

	// Check if flags are provided
	if opts.SchemaURL == "" && !opts.DisableAPI {
		return errors.New("either --url or --disable-api flag must be specified")
	}

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

	// Print validation success message
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

// configureValidationSettings configures validation settings based on flags.
func configureValidationSettings() {
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
		// This will be checked in runCommand
		corev1.SetDisableAPIValidation(true)
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
}
