// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"

	objectsv1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/objects/v1"
	objectsv2 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/objects/v2"
	objectsv3 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/objects/v3"
	routingv1 "github.com/agntcy/dir/api/routing/v1"
	clicmd "github.com/agntcy/dir/cli/cmd"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Ptr creates a pointer to the given value.
func Ptr[T any](v T) *T {
	return &v
}

// CollectChannelItems collects all items from a channel into a slice.
// This utility eliminates the repetitive pattern of iterating over channels
// in list operations throughout the test suite.
func CollectChannelItems(itemsChan <-chan *routingv1.ListResponse) []*routingv1.ListResponse {
	//nolint:prealloc // Cannot pre-allocate when reading from channel - count is unknown
	var items []*routingv1.ListResponse
	for item := range itemsChan {
		items = append(items, item)
	}

	return items
}

// OASFVersion represents the detected OASF version from JSON.
type OASFVersion string

const (
	OASFVersionV1 OASFVersion = "v0.3.1"
	OASFVersionV2 OASFVersion = "v0.4.0"
	OASFVersionV3 OASFVersion = "v0.5.0"
)

// schemaVersionDetector is used to extract schema_version from JSON.
type schemaVersionDetector struct {
	SchemaVersion string `json:"schema_version"`
}

// detectOASFVersion extracts the OASF version from JSON by reading schema_version field.
func detectOASFVersion(jsonData []byte) (OASFVersion, error) {
	var detector schemaVersionDetector
	if err := json.Unmarshal(jsonData, &detector); err != nil {
		return "", fmt.Errorf("failed to detect OASF version: %w", err)
	}

	switch detector.SchemaVersion {
	case "v0.3.1":
		return OASFVersionV1, nil
	case "v0.4.0":
		return OASFVersionV2, nil
	case "v0.5.0":
		return OASFVersionV3, nil
	default:
		return "", fmt.Errorf("unsupported OASF version: %s", detector.SchemaVersion)
	}
}

// CompareOASFRecords compares two OASF JSON records with version-aware logic.
// This function automatically detects OASF versions and uses appropriate comparison logic.
func CompareOASFRecords(json1, json2 []byte) (bool, error) {
	// Detect versions from both JSON records
	version1, err := detectOASFVersion(json1)
	if err != nil {
		return false, fmt.Errorf("failed to detect version for first record: %w", err)
	}

	version2, err := detectOASFVersion(json2)
	if err != nil {
		return false, fmt.Errorf("failed to detect version for second record: %w", err)
	}

	// Both records must be the same version to compare
	if version1 != version2 {
		return false, fmt.Errorf("version mismatch: %s vs %s", version1, version2)
	}

	// Use version-specific comparison
	switch version1 {
	case OASFVersionV1:
		return compareV1Records(json1, json2)
	case OASFVersionV2:
		return compareV2Records(json1, json2)
	case OASFVersionV3:
		return compareV3Records(json1, json2)
	default:
		return false, fmt.Errorf("unsupported OASF version: %s", version1)
	}
}

// compareV1Records compares two V1 Agent records.
func compareV1Records(json1, json2 []byte) (bool, error) {
	var agent1, agent2 objectsv1.Agent

	if err := json.Unmarshal(json1, &agent1); err != nil {
		return false, fmt.Errorf("failed to unmarshal V1 agent 1: %w", err)
	}

	if err := json.Unmarshal(json2, &agent2); err != nil {
		return false, fmt.Errorf("failed to unmarshal V1 agent 2: %w", err)
	}

	return compareV1Agents(&agent1, &agent2)
}

// compareV2Records compares two V2 AgentRecord records.
func compareV2Records(json1, json2 []byte) (bool, error) {
	var record1, record2 objectsv2.AgentRecord

	if err := json.Unmarshal(json1, &record1); err != nil {
		return false, fmt.Errorf("failed to unmarshal V2 record 1: %w", err)
	}

	if err := json.Unmarshal(json2, &record2); err != nil {
		return false, fmt.Errorf("failed to unmarshal V2 record 2: %w", err)
	}

	return compareV2AgentRecords(&record1, &record2)
}

// compareV3Records compares two V3 Record records.
func compareV3Records(json1, json2 []byte) (bool, error) {
	var record1, record2 objectsv3.Record

	if err := json.Unmarshal(json1, &record1); err != nil {
		return false, fmt.Errorf("failed to unmarshal V3 record 1: %w", err)
	}

	if err := json.Unmarshal(json2, &record2); err != nil {
		return false, fmt.Errorf("failed to unmarshal V3 record 2: %w", err)
	}

	return compareV3RecordStructs(&record1, &record2)
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

	return compareV1Agents(&agent1, &agent2)
}

// CompareJSONAgents compares two agent JSON byte arrays for equality.
func CompareJSONAgents(json1, json2 []byte) (bool, error) {
	return compareJSONAgents(json1, json2)
}

//nolint:govet
func compareV1Agents(agent1, agent2 *objectsv1.Agent) (bool, error) {
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

// CompareAgents compares two agent objects for equality.
func CompareAgents(agent1, agent2 *objectsv1.Agent) (bool, error) {
	return compareV1Agents(agent1, agent2)
}

//nolint:govet
func compareV2AgentRecords(record1, record2 *objectsv2.AgentRecord) (bool, error) {
	// Normalize CreatedAt
	record1.CreatedAt = record2.GetCreatedAt()

	// Sort authors
	sort.Strings(record1.GetAuthors())
	sort.Strings(record2.GetAuthors())

	// Sort locators by type
	sort.Slice(record1.GetLocators(), func(i, j int) bool {
		return record1.GetLocators()[i].GetType() < record1.GetLocators()[j].GetType()
	})
	sort.Slice(record2.GetLocators(), func(i, j int) bool {
		return record2.GetLocators()[i].GetType() < record2.GetLocators()[j].GetType()
	})

	// Sort extensions by name
	sort.Slice(record1.GetExtensions(), func(i, j int) bool {
		return record1.GetExtensions()[i].GetName() < record1.GetExtensions()[j].GetName()
	})
	sort.Slice(record2.GetExtensions(), func(i, j int) bool {
		return record2.GetExtensions()[i].GetName() < record2.GetExtensions()[j].GetName()
	})

	// Sort skills by name (V2 uses simple name/id format)
	sort.Slice(record1.GetSkills(), func(i, j int) bool {
		return record1.GetSkills()[i].GetName() < record1.GetSkills()[j].GetName()
	})
	sort.Slice(record2.GetSkills(), func(i, j int) bool {
		return record2.GetSkills()[i].GetName() < record2.GetSkills()[j].GetName()
	})

	// Marshal and compare
	json1, err := json.Marshal(record1)
	if err != nil {
		return false, fmt.Errorf("failed to marshal V2 record 1: %w", err)
	}

	json2, err := json.Marshal(record2)
	if err != nil {
		return false, fmt.Errorf("failed to marshal V2 record 2: %w", err)
	}

	return reflect.DeepEqual(json1, json2), nil
}

//nolint:govet
func compareV3RecordStructs(record1, record2 *objectsv3.Record) (bool, error) {
	// Normalize CreatedAt
	record1.CreatedAt = record2.GetCreatedAt()

	// Sort authors
	sort.Strings(record1.GetAuthors())
	sort.Strings(record2.GetAuthors())

	// Sort locators by type
	sort.Slice(record1.GetLocators(), func(i, j int) bool {
		return record1.GetLocators()[i].GetType() < record1.GetLocators()[j].GetType()
	})
	sort.Slice(record2.GetLocators(), func(i, j int) bool {
		return record2.GetLocators()[i].GetType() < record2.GetLocators()[j].GetType()
	})

	// Sort extensions by name
	sort.Slice(record1.GetExtensions(), func(i, j int) bool {
		return record1.GetExtensions()[i].GetName() < record1.GetExtensions()[j].GetName()
	})
	sort.Slice(record2.GetExtensions(), func(i, j int) bool {
		return record2.GetExtensions()[i].GetName() < record2.GetExtensions()[j].GetName()
	})

	// Sort skills by name (V3 uses simple name/id format like V2)
	sort.Slice(record1.GetSkills(), func(i, j int) bool {
		return record1.GetSkills()[i].GetName() < record1.GetSkills()[j].GetName()
	})
	sort.Slice(record2.GetSkills(), func(i, j int) bool {
		return record2.GetSkills()[i].GetName() < record2.GetSkills()[j].GetName()
	})

	// Sort domains by name (V3-specific field)
	sort.Slice(record1.GetDomains(), func(i, j int) bool {
		return record1.GetDomains()[i].GetName() < record1.GetDomains()[j].GetName()
	})
	sort.Slice(record2.GetDomains(), func(i, j int) bool {
		return record2.GetDomains()[i].GetName() < record2.GetDomains()[j].GetName()
	})

	// Marshal and compare
	json1, err := json.Marshal(record1)
	if err != nil {
		return false, fmt.Errorf("failed to marshal V3 record 1: %w", err)
	}

	json2, err := json.Marshal(record2)
	if err != nil {
		return false, fmt.Errorf("failed to marshal V3 record 2: %w", err)
	}

	return reflect.DeepEqual(json1, json2), nil
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
