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
	searchv1 "github.com/agntcy/dir/api/search/v1"
	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/agntcy/dir/client"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "validate [<file>|--all]",
	Short: "Validate OASF record JSON files locally or all records in a directory instance",
	Long: `Validate OASF record JSON files against the OASF schema locally, or validate
all records in a directory instance.

For single file validation, this command performs local validation without requiring
a connection to a Directory server. You must specify either --url for API-based
validation or --disable-api for embedded schema validation.

For validating all records in a directory instance, use the --all flag. This
requires a connection to a Directory server and will validate all records stored
in that instance.

Usage examples:

1. Validate a single file using embedded schemas (no API calls):
   dirctl validate record.json --disable-api

2. Validate a single file with API-based validation:
   dirctl validate record.json --url https://schema.oasf.outshift.com

3. Validate a single file with non-strict mode (more permissive, only works with --url):
   dirctl validate record.json --url https://schema.oasf.outshift.com --disable-strict

4. Validate all records in a directory instance using embedded schemas:
   dirctl validate --all --disable-api

5. Validate all records in a directory instance with API-based validation:
   dirctl validate --all --url https://schema.oasf.outshift.com

Note: You must specify either --url (for API validation) or --disable-api
(for embedded schema validation). For single file validation, this command is
intended for local validation purposes. Use --all to validate all records in
a directory instance.
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if opts.ValidateAll {
			// Validate all records in directory instance
			if len(args) > 0 {
				return errors.New("file path cannot be specified when using --all flag")
			}

			return runValidateAllCommand(cmd)
		}

		// Validate a single file
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

	// Configure validation settings
	configureValidationSettings()

	// Check if flags are provided (for single file validation)
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

// runValidateAllCommand validates all records in the directory instance.
func runValidateAllCommand(cmd *cobra.Command) error {
	// Check if flags are provided (for --all validation) before connecting
	// This check must happen before configureValidationSettings() which sets defaults
	if opts.SchemaURL == "" && !opts.DisableAPI {
		return errors.New("either --url or --disable-api flag must be specified when using --all")
	}

	// Get the client from the context
	var c *client.Client

	var ok bool

	c, ok = ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	// Configure validation settings (same as single file validation)
	// Note: configureValidationSettings() may set defaults, but we've already checked flags above
	configureValidationSettings()

	// Search for all records (empty query returns all)
	cids, err := collectAllCIDs(cmd, c)
	if err != nil {
		return err
	}

	// Validate each record and collect results
	return validateAllRecords(cmd, c, cids)
}

// collectAllCIDs collects all CIDs from the directory instance.
func collectAllCIDs(cmd *cobra.Command, c *client.Client) ([]string, error) {
	limit := uint32(0) // 0 = no limit

	result, err := c.SearchCIDs(cmd.Context(), &searchv1.SearchCIDsRequest{
		Limit:   &limit,
		Queries: []*searchv1.RecordQuery{}, // Empty = all records
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search records: %w", err)
	}

	// Collect all CIDs
	var cids []string

	for {
		select {
		case resp := <-result.ResCh():
			cid := resp.GetRecordCid()
			if cid != "" {
				cids = append(cids, cid)
			}
		case err := <-result.ErrCh():
			return nil, fmt.Errorf("error receiving CID: %w", err)
		case <-result.DoneCh():
			return cids, nil
		case <-cmd.Context().Done():
			return nil, cmd.Context().Err()
		}
	}
}

// validateAllRecords validates all records and prints a summary report.
func validateAllRecords(cmd *cobra.Command, c *client.Client, cids []string) error {
	var totalValid, totalInvalid int

	var invalidCIDs []string

	presenter.Printf(cmd, "Validating %d records...\n", len(cids))

	for i, cid := range cids {
		valid, err := validateSingleRecord(cmd, c, cid, i+1, len(cids))
		if err != nil {
			totalInvalid++

			invalidCIDs = append(invalidCIDs, cid)

			continue
		}

		if valid {
			totalValid++
		} else {
			totalInvalid++

			invalidCIDs = append(invalidCIDs, cid)
		}
	}

	// Print summary report
	return printValidationSummary(cmd, len(cids), totalValid, totalInvalid, invalidCIDs)
}

// validateSingleRecord validates a single record by CID.
func validateSingleRecord(cmd *cobra.Command, c *client.Client, cid string, current, total int) (bool, error) {
	// Pull the record
	record, err := c.Pull(cmd.Context(), &corev1.RecordRef{Cid: cid})
	if err != nil {
		presenter.Printf(cmd, "  [%d/%d] Failed to pull record %s: %v\n", current, total, cid, err)

		return false, err
	}

	// Validate the record
	ctx := cmd.Context()

	valid, validationErrors, err := record.Validate(ctx)
	if err != nil {
		presenter.Printf(cmd, "  [%d/%d] Validation error for %s: %v\n", current, total, cid, err)

		return false, err
	}

	if !valid && len(validationErrors) > 0 {
		presenter.Printf(cmd, "  [%d/%d] Invalid: %s (%d error(s))\n", current, total, cid, len(validationErrors))
	}

	return valid, nil
}

// printValidationSummary prints the validation summary report.
func printValidationSummary(cmd *cobra.Command, total, valid, invalid int, invalidCIDs []string) error {
	presenter.Printf(cmd, "\n=== Validation Summary ===\n")
	presenter.Printf(cmd, "Total records validated: %d\n", total)
	presenter.Printf(cmd, "Valid:   %d\n", valid)
	presenter.Printf(cmd, "Invalid: %d\n", invalid)

	if len(invalidCIDs) > 0 {
		presenter.Printf(cmd, "\nInvalid record CIDs:\n")

		for _, cid := range invalidCIDs {
			presenter.Printf(cmd, "  - %s\n", cid)
		}
	}

	if invalid > 0 {
		return fmt.Errorf("validation completed with %d invalid record(s)", invalid)
	}

	return nil
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
		// This will be checked in runCommand, but for --all we need to set a default
		// Default to embedded schemas for --all if no flag specified
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
