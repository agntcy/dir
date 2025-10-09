// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package presenter

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
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

// PrintMessage outputs data in the appropriate format based on command flags.
func PrintMessage(cmd *cobra.Command, title, message string, value any) error {
	opts := GetOutputOptions(cmd)

	// Handle empty case for multiple values
	if value == nil {
		Println(cmd, fmt.Sprintf("No %s found", title))

		return nil
	}

	// Handle slice/array with zero length
	if slice, ok := value.([]interface{}); ok && len(slice) == 0 {
		Println(cmd, fmt.Sprintf("No %s found", title))

		return nil
	}

	switch opts.Format {
	case FormatRaw:
		// For raw format, output just the value
		Print(cmd, fmt.Sprintf("%v", value))

		return nil

	case FormatJSON:
		// For JSON format, output the value as JSON
		output, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}

		Print(cmd, string(output))

		return nil

	case FormatHuman:
		// For human-readable format, output with descriptive message
		Println(cmd, fmt.Sprintf("%s: %s", message, fmt.Sprintf("%v", value)))

		return nil
	}

	return nil
}
