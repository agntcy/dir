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
	"strings"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/cli/presenter"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "validate [<file>]",
	Short: "Validate OASF record JSON from a file or stdin",
	Long: `Validate OASF record JSON against the OASF schema. The JSON can be provided
as a file path or piped from stdin (e.g., from dirctl pull).

A schema URL must be provided via --url for API-based validation.

Usage examples:

1. Validate a file with API-based validation:
   dirctl validate record.json --url https://schema.oasf.outshift.com

2. Validate JSON piped from stdin:
   cat record.json | dirctl validate --url https://schema.oasf.outshift.com

3. Validate a record pulled from directory:
   dirctl pull <cid> --output json | dirctl validate --url https://schema.oasf.outshift.com
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			jsonData []byte
			err      error
		)

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

	// Initialize validator with schema URL
	// Note: opts.SchemaURL is populated by cobra during flag parsing in Execute()
	if opts.SchemaURL == "" {
		return fmt.Errorf("schema URL is required, use --url flag to provide it")
	}

	if err := corev1.InitializeValidator(opts.SchemaURL); err != nil {
		return fmt.Errorf("failed to initialize validator: %w", err)
	}

	// Validate the record
	ctx := cmd.Context()

	valid, validationErrors, err := record.Validate(ctx)
	if err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	// Output results
	if !valid {
		return outputValidationErrors(cmd, record, validationErrors)
	}

	return outputValidationSuccess(cmd, record, validationErrors)
}

func outputValidationSuccess(cmd *cobra.Command, record *corev1.Record, warnings []string) error {
	schemaVersion := record.GetSchemaVersion()

	// Build the complete message with all validation messages
	var msg strings.Builder
	if schemaVersion != "" {
		fmt.Fprintf(&msg, "Record is valid (schema version: %s)", schemaVersion)
	} else {
		msg.WriteString("Record is valid")
	}

	if len(warnings) > 0 {
		fmt.Fprintf(&msg, " with %d warning(s):\n", len(warnings))

		for i, warning := range warnings {
			fmt.Fprintf(&msg, "  %d. %s\n", i+1, warning)
		}
	}

	presenter.Print(cmd, msg.String())

	return nil
}

func outputValidationErrors(_ *cobra.Command, record *corev1.Record, validationErrors []string) error {
	if len(validationErrors) > 0 {
		schemaVersion := record.GetSchemaVersion()

		// Build the complete error message with all validation messages
		var errorMsg strings.Builder
		if schemaVersion != "" {
			fmt.Fprintf(&errorMsg, "record validation failed (schema version: %s) with %d message(s):\n", schemaVersion, len(validationErrors))
		} else {
			fmt.Fprintf(&errorMsg, "record validation failed with %d message(s):\n", len(validationErrors))
		}

		for i, msg := range validationErrors {
			fmt.Fprintf(&errorMsg, "  %d. %s\n", i+1, msg)
		}

		return errors.New(errorMsg.String())
	}

	// Fallback if no error details available
	return errors.New("record validation failed (no error details available)")
}
