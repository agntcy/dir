// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package presenter

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Print writes to stdout.
func Print(cmd *cobra.Command, args ...interface{}) {
	_, _ = fmt.Fprint(cmd.OutOrStdout(), args...)
}

// Println writes to stdout with a newline.
func Println(cmd *cobra.Command, args ...interface{}) {
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), args...)
}

// Printf writes formatted output to stdout.
func Printf(cmd *cobra.Command, format string, args ...interface{}) {
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), format, args...)
}

// PrintSmartf writes formatted output to stdout for human format,
// or to stderr for structured formats (json, jsonl, raw).
// Use this for metadata messages that should not pollute structured output.
func PrintSmartf(cmd *cobra.Command, format string, args ...interface{}) {
	opts := GetOutputOptions(cmd)
	if opts.IsStructuredOutput() {
		Errorf(cmd, format, args...)
	} else {
		Printf(cmd, format, args...)
	}
}

// Error writes to stderr.
func Error(cmd *cobra.Command, args ...interface{}) {
	_, _ = fmt.Fprint(cmd.ErrOrStderr(), args...)
}

// Errorf writes formatted output to stderr.
func Errorf(cmd *cobra.Command, format string, args ...interface{}) {
	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), format, args...)
}
