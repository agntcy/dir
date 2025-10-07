// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package presenter

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// OutputFormat represents the different output formats available.
type OutputFormat string

const (
	FormatHuman OutputFormat = "human"
	FormatJSON  OutputFormat = "json"
	FormatRaw   OutputFormat = "raw"
)

// OutputOptions holds the output formatting options.
type OutputOptions struct {
	Format OutputFormat
}

// GetOutputOptions extracts output format options from command flags.
func GetOutputOptions(cmd *cobra.Command) OutputOptions {
	opts := OutputOptions{
		Format: FormatHuman, // Default to human-readable
	}

	// Check for --json flag
	if jsonFlag, _ := cmd.Flags().GetBool("json"); jsonFlag {
		opts.Format = FormatJSON
	}

	// Check for --raw flag. This takes precedence over --json.
	if rawFlag, _ := cmd.Flags().GetBool("raw"); rawFlag {
		opts.Format = FormatRaw
	}

	return opts
}

// AddOutputFlags adds standard --json and --raw flags to a command.
func AddOutputFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("json", false, "Output results in JSON format")
	cmd.Flags().Bool("raw", false, "Output raw values without formatting")
}

// OutputSingleValue outputs a single value in the appropriate format.
func OutputSingleValue(cmd *cobra.Command, opts OutputOptions, label string, message string, value interface{}) error {
	switch opts.Format {
	case FormatJSON:
		// For single values, output as JSON object
		result := map[string]interface{}{
			label: value,
		}

		output, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}

		Print(cmd, string(output))

		return nil

	case FormatRaw:
		// For raw format, output just the value
		Print(cmd, fmt.Sprintf("%v", value))

		return nil

	case FormatHuman:
		// For human-readable format, output with descriptive label
		Println(cmd, fmt.Sprintf("%s: %v", message, value))

		return nil
	}

	return nil
}

// OutputMultipleValues outputs multiple values in the appropriate format.
func OutputMultipleValues(cmd *cobra.Command, opts OutputOptions, label string, values []interface{}) error {
	if len(values) == 0 {
		Println(cmd, fmt.Sprintf("No %s to output", label))

		return nil
	}

	return OutputStructuredData(cmd, opts, label, values)
}

// OutputStructuredData outputs structured data in the appropriate format.
func OutputStructuredData(cmd *cobra.Command, opts OutputOptions, label string, data interface{}) error {
	switch opts.Format {
	case FormatRaw:
		// For raw format, try to extract raw values
		if rawData, ok := data.([]byte); ok {
			Print(cmd, string(rawData))
		} else {
			// Fallback to string representation
			Print(cmd, fmt.Sprintf("%v", data))
		}

		return nil

	case FormatJSON:
		// For JSON format, output the data as-is
		output, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}

		Print(cmd, string(output))

		return nil

	case FormatHuman:
		Println(cmd, cases.Title(language.English).String(label)+":")

		// For human format, output the data as a JSON object
		output, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}

		Print(cmd, string(output))

		return nil
	}

	return nil
}
