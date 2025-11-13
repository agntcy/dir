// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/agntcy/oasf-sdk/pkg/validator"
)

// schemaClass represents a generic schema class (domain or skill).
type schemaClass struct {
	Name    string
	Caption string
	ID      int
}

// validateVersion checks if the provided version is valid and returns available versions.
func validateVersion(version string) ([]string, error) {
	availableVersions, err := validator.GetAvailableSchemaVersions()
	if err != nil {
		return nil, fmt.Errorf("failed to get available schema versions: %w", err)
	}

	if version == "" {
		return availableVersions, fmt.Errorf("version parameter is required. Available versions: %s",
			strings.Join(availableVersions, ", "))
	}

	versionValid := false

	for _, v := range availableVersions {
		if version == v {
			versionValid = true

			break
		}
	}

	if !versionValid {
		return availableVersions, fmt.Errorf("invalid version '%s'. Available versions: %s",
			version, strings.Join(availableVersions, ", "))
	}

	return availableVersions, nil
}

// parseSchemaData parses JSON schema data into a list of schema items.
func parseSchemaData(data []byte, parseFunc func(map[string]interface{}) schemaClass) ([]schemaClass, error) {
	var schemaData map[string]interface{}
	if err := json.Unmarshal(data, &schemaData); err != nil {
		return nil, fmt.Errorf("failed to parse schema data: %w", err)
	}

	var items []schemaClass

	for _, itemDef := range schemaData {
		defMap, ok := itemDef.(map[string]interface{})
		if !ok {
			continue
		}

		item := parseFunc(defMap)
		if item.Name != "" {
			items = append(items, item)
		}
	}

	return items, nil
}

// filterChildItems returns child items that are direct descendants of the parent.
func filterChildItems(allItems []schemaClass, parent string) ([]schemaClass, error) {
	prefix := parent + "/"

	var children []schemaClass

	for _, item := range allItems {
		if !strings.HasPrefix(item.Name, prefix) {
			continue
		}

		remainder := strings.TrimPrefix(item.Name, prefix)
		if !strings.Contains(remainder, "/") {
			children = append(children, item)
		}
	}

	if len(children) == 0 {
		return nil, fmt.Errorf("parent '%s' not found or has no children", parent)
	}

	return children, nil
}

// extractTopLevelCategories extracts unique top-level parent categories from items.
func extractTopLevelCategories(allItems []schemaClass) []schemaClass {
	parentCategories := make(map[string]bool)
	topLevel := make([]schemaClass, 0, len(allItems))

	for _, item := range allItems {
		idx := strings.Index(item.Name, "/")
		if idx <= 0 {
			continue
		}

		parentCategory := item.Name[:idx]
		if parentCategories[parentCategory] {
			continue
		}

		parentCategories[parentCategory] = true
		topLevel = append(topLevel, schemaClass{Name: parentCategory})
	}

	return topLevel
}

// parseItemFromSchema extracts schema item information from the schema definition.
func parseItemFromSchema(defMap map[string]interface{}) schemaClass {
	item := schemaClass{}

	// Extract title for caption
	if title, ok := defMap["title"].(string); ok {
		item.Caption = title
	}

	// Extract properties
	props, ok := defMap["properties"].(map[string]interface{})
	if !ok {
		return item
	}

	// Extract name
	if nameField, ok := props["name"].(map[string]interface{}); ok {
		if constVal, ok := nameField["const"].(string); ok {
			item.Name = constVal
		}
	}

	// Extract ID
	if idField, ok := props["id"].(map[string]interface{}); ok {
		if constVal, ok := idField["const"].(float64); ok {
			item.ID = int(constVal)
		}
	}

	return item
}
