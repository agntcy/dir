// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import (
	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
)

type QueryFilters struct {
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`

	Name        string   `json:"name,omitempty"`
	Version     string   `json:"version,omitempty"`
	Description string   `json:"description,omitempty"`
	Authors     []string `json:"authors,omitempty"`

	SkillNames      []string `json:"skill_names,omitempty"`
	SkillCategories []string `json:"skill_categories,omitempty"`

	LocatorTypes []string `json:"locator_types,omitempty"`

	ExtensionNames    []string `json:"extension_names,omitempty"`
	ExtensionVersions []string `json:"extension_versions,omitempty"`
}

type SearchAPI interface {
	// AddAgent adds an object to the search database.
	AddAgent(agent *coretypes.Agent) error

	// GetManyAgentsByFilters queries agents from the search database.
	GetManyAgentsByFilters(filters QueryFilters) ([]*coretypes.Agent, error)
}
