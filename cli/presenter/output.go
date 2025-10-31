// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package presenter

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/spf13/cobra"
)

// OutputFormat represents the different output formats available.
type OutputFormat string

const (
	// FormatHuman is the default human-readable output format.
	FormatHuman OutputFormat = "human"
	// FormatJSON is pretty-printed JSON format with indentation.
	FormatJSON OutputFormat = "json"
	// FormatJSONL is newline-delimited JSON format (one object per line, no indentation).
	FormatJSONL OutputFormat = "jsonl"
	// FormatRaw outputs only raw values (CIDs, IDs, etc.) without formatting.
	FormatRaw OutputFormat = "raw"
)

// OutputOptions holds the output formatting options.
type OutputOptions struct {
	Format OutputFormat
}

// IsStructuredOutput returns true if the output format is structured (json, jsonl, or raw).
// Structured outputs route metadata to stderr instead of stdout.
func (o OutputOptions) IsStructuredOutput() bool {
	return o.Format == FormatJSON || o.Format == FormatJSONL || o.Format == FormatRaw
}

// GetOutputOptions extracts output format options from command flags.
func GetOutputOptions(cmd *cobra.Command) OutputOptions {
	opts := OutputOptions{
		Format: FormatHuman, // Default to human-readable
	}

	// Check for --output flag
	if outputFlag, err := cmd.Flags().GetString("output"); err == nil && outputFlag != "" {
		opts.Format = OutputFormat(outputFlag)
	}

	return opts
}

// AddOutputFlags adds the standard --output flag to a command.
func AddOutputFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("output", "o", "human", "Output format: human|json|jsonl|raw")
}

// PrintMessage outputs data in the appropriate format based on command flags.
func PrintMessage(cmd *cobra.Command, title, message string, value any) error {
	opts := GetOutputOptions(cmd)

	// Handle empty case for multiple values
	if value == nil || isEmptySlice(value) {
		if opts.IsStructuredOutput() {
			// For structured output, print empty array to stdout
			Print(cmd, "[]\n")
		} else {
			// For human format, print descriptive message
			Println(cmd, fmt.Sprintf("No %s found", title))
		}

		return nil
	}

	switch opts.Format {
	case FormatRaw:
		// For raw format, output just the value
		Print(cmd, fmt.Sprintf("%v", value))

		return nil

	case FormatJSON:
		// For JSON format, output the value as JSON with indentation
		output, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}

		Print(cmd, string(output))
		Print(cmd, "\n")

		return nil

	case FormatJSONL:
		// For JSONL format, output newline-delimited JSON
		return printJSONL(cmd, value)

	case FormatHuman:
		// For human-readable format, output with descriptive message
		Println(cmd, fmt.Sprintf("%s: %s", message, fmt.Sprintf("%v", value)))

		return nil
	}

	return nil
}

// printJSONL outputs data in newline-delimited JSON format (one object per line, no indentation).
func printJSONL(cmd *cobra.Command, value any) error {
	// Handle single object
	if !isSliceOrArray(value) {
		output, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}

		Printf(cmd, "%s\n", string(output))

		return nil
	}

	// Handle array/slice - print each element on a separate line
	v := reflect.ValueOf(value)
	for i := range v.Len() {
		output, err := json.Marshal(v.Index(i).Interface())
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}

		Printf(cmd, "%s\n", string(output))
	}

	return nil
}

// isSliceOrArray returns true if the value is a slice or array.
func isSliceOrArray(value any) bool {
	if value == nil {
		return false
	}

	v := reflect.ValueOf(value)

	return v.Kind() == reflect.Slice || v.Kind() == reflect.Array
}

// isEmptySlice returns true if the value is an empty slice.
func isEmptySlice(value any) bool {
	if value == nil {
		return false
	}

	if slice, ok := value.([]interface{}); ok && len(slice) == 0 {
		return true
	}

	// Check using reflection for other slice types
	v := reflect.ValueOf(value)
	if v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
		return v.Len() == 0
	}

	return false
}
