// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"

	objectsv1 "github.com/agntcy/dir/api/objects/v1"
	clicmd "github.com/agntcy/dir/cli/cmd"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func Ptr[T any](v T) *T {
	return &v
}

//nolint:govet
func compareJSONAgents(json1, json2 []byte) (bool, error) {
	var agent1, agent2 objectsv1.Agent

	// Convert to JSON
	if err := json.Unmarshal(json1, &agent1); err != nil {
		return false, fmt.Errorf("failed to unmarshal json: %w", err)
	}

	if err := json.Unmarshal(json2, &agent2); err != nil {
		return false, fmt.Errorf("failed to unmarshal json: %w", err)
	}

	return compareAgents(&agent1, &agent2)
}

//nolint:govet
func compareAgents(agent1, agent2 *objectsv1.Agent) (bool, error) {
	// Overwrite CreatedAt
	agent1.CreatedAt = agent2.GetCreatedAt()

	// Sort the authors slices
	sort.Strings(agent1.GetAuthors())
	sort.Strings(agent2.GetAuthors())

	// Sort the locators slices by type
	sort.Slice(agent1.GetLocators(), func(i, j int) bool {
		return agent1.GetLocators()[i].GetType() < agent1.GetLocators()[j].GetType()
	})
	sort.Slice(agent2.GetLocators(), func(i, j int) bool {
		return agent2.GetLocators()[i].GetType() < agent2.GetLocators()[j].GetType()
	})

	// Sort the extensions slices
	sort.Slice(agent1.GetExtensions(), func(i, j int) bool {
		return agent1.GetExtensions()[i].GetName() < agent1.GetExtensions()[j].GetName()
	})
	sort.Slice(agent2.GetExtensions(), func(i, j int) bool {
		return agent2.GetExtensions()[i].GetName() < agent2.GetExtensions()[j].GetName()
	})

	// Convert JSON
	json1, err := json.Marshal(agent1)
	if err != nil {
		return false, fmt.Errorf("failed to marshal json: %w", err)
	}

	json2, err := json.Marshal(agent2)
	if err != nil {
		return false, fmt.Errorf("failed to marshal json: %w", err)
	}

	return reflect.DeepEqual(json1, json2), nil //nolint:govet
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
