// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	clicmd "github.com/agntcy/dir/cli/cmd"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// ResetCobraFlags resets all CLI command flags to their default values.
// This ensures clean state between test executions.
func ResetCobraFlags() {
	// Reset root command flags
	resetCommandFlags(clicmd.RootCmd)

	// Walk through all subcommands and reset their flags
	for _, cmd := range clicmd.RootCmd.Commands() {
		resetCommandFlags(cmd)

		// Also reset any nested subcommands
		resetNestedCommandFlags(cmd)
	}
}

// resetCommandFlags resets flags for a specific command.
//
//nolint:errcheck
func resetCommandFlags(cmd *cobra.Command) {
	if cmd.Flags() != nil {
		// Reset local flags
		cmd.Flags().VisitAll(func(flag *pflag.Flag) {
			if flag.Value != nil {
				// Reset to default value based on flag type
				switch flag.Value.Type() {
				case "string":
					flag.Value.Set(flag.DefValue)
				case "bool":
					flag.Value.Set(flag.DefValue)
				case "int", "int32", "int64":
					flag.Value.Set(flag.DefValue)
				case "uint", "uint32", "uint64":
					flag.Value.Set(flag.DefValue)
				case "float32", "float64":
					flag.Value.Set(flag.DefValue)
				default:
					// For custom types, try to set to default value
					flag.Value.Set(flag.DefValue)
				}
				// Mark as not changed
				flag.Changed = false
			}
		})
	}

	if cmd.PersistentFlags() != nil {
		// Reset persistent flags
		cmd.PersistentFlags().VisitAll(func(flag *pflag.Flag) {
			if flag.Value != nil {
				flag.Value.Set(flag.DefValue)
				flag.Changed = false
			}
		})
	}
}

// resetNestedCommandFlags recursively resets flags for nested commands.
func resetNestedCommandFlags(cmd *cobra.Command) {
	for _, subCmd := range cmd.Commands() {
		resetCommandFlags(subCmd)
		resetNestedCommandFlags(subCmd)
	}
}

// ResetCLIState provides a comprehensive reset of CLI state.
// This combines flag reset with any other state that needs to be cleared.
func ResetCLIState() {
	ResetCobraFlags()

	// Reset command args
	clicmd.RootCmd.SetArgs(nil)

	// Clear any output buffers by setting output to default
	clicmd.RootCmd.SetOut(nil)
	clicmd.RootCmd.SetErr(nil)
}
