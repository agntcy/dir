// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
)

func Ptr[T any](v T) *T {
	return &v
}

//nolint:govet
func compareJSONAgents(json1, json2 []byte) (bool, error) {
	var agent1, agent2 coretypes.Agent

	// Convert to JSON
	if err := json.Unmarshal(json1, &agent1); err != nil {
		return false, fmt.Errorf("failed to unmarshal json: %w", err)
	}

	if err := json.Unmarshal(json2, &agent2); err != nil {
		return false, fmt.Errorf("failed to unmarshal json: %w", err)
	}

	return compareAgents(agent1, agent2)
}

//nolint:govet
func compareAgents(agent1, agent2 coretypes.Agent) (bool, error) {
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
	json1, err := json.Marshal(&agent1)
	if err != nil {
		return false, fmt.Errorf("failed to marshal json: %w", err)
	}

	json2, err := json.Marshal(&agent2)
	if err != nil {
		return false, fmt.Errorf("failed to marshal json: %w", err)
	}

	return reflect.DeepEqual(json1, json2), nil //nolint:govet
}
