// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"reflect"

	corev1 "github.com/agntcy/dir/api/core/v1"
	routingv1 "github.com/agntcy/dir/api/routing/v1"
	clicmd "github.com/agntcy/dir/cli/cmd"
	searchcmd "github.com/agntcy/dir/cli/cmd/search"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Ptr creates a pointer to the given value.
func Ptr[T any](v T) *T {
	return &v
}

// CollectItems collects all items from a channel into a slice.
// This generic utility eliminates the repetitive pattern of iterating over channels
// and works with any channel type.
func CollectItems[T any](itemsChan <-chan T) []T {
	//nolint:prealloc // Cannot pre-allocate when reading from channel - count is unknown
	var items []T
	for item := range itemsChan {
		items = append(items, item)
	}

	return items
}

// CollectListItems collects all list items from a channel into a slice.
// Wrapper around generic CollectItems for routing list operations.
func CollectListItems(itemsChan <-chan *routingv1.ListResponse) []*routingv1.ListResponse {
	return CollectItems(itemsChan)
}

// CollectSearchItems collects all search items from a channel into a slice.
// Wrapper around generic CollectItems for routing search operations.
func CollectSearchItems(searchChan <-chan *routingv1.SearchResponse) []*routingv1.SearchResponse {
	return CollectItems(searchChan)
}

// CompareOASFRecords compares two OASF JSON records with version-aware logic.
// This function automatically detects OASF versions and uses appropriate comparison logic.
//
//nolint:wrapcheck
func CompareOASFRecords(json1, json2 []byte) (bool, error) {
	record1, err := corev1.UnmarshalRecord(json1)
	if err != nil {
		return false, err
	}

	record2, err := corev1.UnmarshalRecord(json2)
	if err != nil {
		return false, err
	}

	return reflect.DeepEqual(record1, record2), nil
}

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
				case "stringArray", "stringSlice":
					// For string arrays/slices, completely clear them
					// Setting to empty string should clear the underlying slice
					flag.Value.Set("")
					// Also reset the default value to ensure clean state
					flag.DefValue = ""
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
				// Handle string arrays specially for persistent flags too
				if flag.Value.Type() == "stringArray" || flag.Value.Type() == "stringSlice" {
					flag.Value.Set("")
					flag.DefValue = ""
				} else {
					flag.Value.Set(flag.DefValue)
				}

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

	// Reset search command global state
	ResetSearchCommandState()

	// Force complete re-initialization of routing command flags to clear accumulated state
	resetRoutingCommandFlags()
}

// ResetSearchCommandState resets the global state in search command.
//
//nolint:errcheck
func ResetSearchCommandState() {
	if cmd := searchcmd.Command; cmd != nil {
		// Reset flags to default values
		cmd.Flags().Set("limit", "100")
		cmd.Flags().Set("offset", "0")

		// Reset all string array flags
		resetStringArrayFlag(cmd, "name")
		resetStringArrayFlag(cmd, "version")
		resetStringArrayFlag(cmd, "skill-id")
		resetStringArrayFlag(cmd, "skill")
		resetStringArrayFlag(cmd, "locator")
		resetStringArrayFlag(cmd, "module")
		resetStringArrayFlag(cmd, "domain-id")
		resetStringArrayFlag(cmd, "domain")
	}
}

// resetRoutingCommandFlags aggressively resets routing command flags and their underlying variables.
// The key insight is that Cobra StringArrayVar flags are bound to Go slice variables that persist
// across command executions. We need to reset both the flag state AND the underlying variables.
func resetRoutingCommandFlags() {
	// Import the routing package to access the global option variables
	// Since we can't import the routing package directly (circular dependency),
	// we need to reset the flags in a way that also clears the underlying slices
	// Find the routing command
	for _, cmd := range clicmd.RootCmd.Commands() {
		if cmd.Name() == "routing" {
			// Reset all routing subcommands
			for _, subCmd := range cmd.Commands() {
				switch subCmd.Name() {
				case "list":
					// Reset list command flags and underlying variables
					resetStringArrayFlag(subCmd, "skill")
					resetStringArrayFlag(subCmd, "locator")
					resetStringArrayFlag(subCmd, "domain")
					resetStringArrayFlag(subCmd, "module")
				case "search":
					// Reset search command flags and underlying variables
					resetStringArrayFlag(subCmd, "skill")
					resetStringArrayFlag(subCmd, "locator")
					resetStringArrayFlag(subCmd, "domain")
					resetStringArrayFlag(subCmd, "module")
				}
			}
		}
	}
}

// resetStringArrayFlag completely resets a StringArrayVar flag by clearing its underlying slice.
func resetStringArrayFlag(cmd *cobra.Command, flagName string) {
	if flag := cmd.Flags().Lookup(flagName); flag != nil {
		// For StringArrayVar flags, we need to clear the underlying slice completely
		// The flag.Value is a pointer to a stringArrayValue that wraps the actual slice
		// Method 1: Set to empty string (should clear the slice)
		_ = flag.Value.Set("") // Ignore error - flag reset is best effort

		// Method 2: Reset all flag metadata
		flag.DefValue = ""
		flag.Changed = false

		// Method 3: If the flag has a slice interface, try to clear it directly
		if sliceValue, ok := flag.Value.(interface{ Replace([]string) error }); ok {
			sliceValue.Replace([]string{}) //nolint:errcheck
		}
	}
}
