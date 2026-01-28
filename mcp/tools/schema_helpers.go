// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
	"sync"

	"github.com/agntcy/oasf-sdk/pkg/schema"
)

var (
	schemaInstance *schema.Schema
	schemaMu       sync.Mutex
	schemaURL      string
)

// getSchemaInstance returns a Schema instance initialized from environment variable.
// It checks the environment variable each time to support test scenarios.
func getSchemaInstance() (*schema.Schema, error) {
	schemaMu.Lock()
	defer schemaMu.Unlock()

	currentSchemaURL := os.Getenv("OASF_API_VALIDATION_SCHEMA_URL")
	if currentSchemaURL == "" {
		return nil, fmt.Errorf("OASF_API_VALIDATION_SCHEMA_URL environment variable is required. Set it to the OASF schema URL (e.g., https://schema.oasf.outshift.com)")
	}

	// If schema URL changed or instance is nil, create a new instance
	if schemaInstance == nil || schemaURL != currentSchemaURL {
		var err error

		schemaInstance, err = schema.New(currentSchemaURL)
		if err != nil {
			return nil, fmt.Errorf("failed to create schema instance: %w", err)
		}

		schemaURL = currentSchemaURL
	}

	return schemaInstance, nil
}

// schemaClass represents a generic schema class (domain or skill).
type schemaClass struct {
	Name    string
	Caption string
	ID      int
}

// validateVersion checks if the provided version is valid and returns available versions.
func validateVersion(ctx context.Context, version string) ([]string, error) {
	// Get schema instance to fetch available versions
	schemaInstance, err := getSchemaInstance()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize schema client: %w", err)
	}

	availableVersions, err := schemaInstance.GetAvailableSchemaVersions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get available schema versions: %w", err)
	}

	if version == "" {
		return availableVersions, fmt.Errorf("version parameter is required. Available versions: %s",
			strings.Join(availableVersions, ", "))
	}

	versionValid := slices.Contains(availableVersions, version)

	if !versionValid {
		return availableVersions, fmt.Errorf("invalid version '%s'. Available versions: %s",
			version, strings.Join(availableVersions, ", "))
	}

	return availableVersions, nil
}

// parseSchemaData parses JSON schema data into a list of schema items.
func parseSchemaData(data []byte, parseFunc func(map[string]any) schemaClass) ([]schemaClass, error) {
	var schemaData map[string]any
	if err := json.Unmarshal(data, &schemaData); err != nil {
		return nil, fmt.Errorf("failed to parse schema data: %w", err)
	}

	var items []schemaClass

	for _, itemDef := range schemaData {
		defMap, ok := itemDef.(map[string]any)
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
func parseItemFromSchema(defMap map[string]any) schemaClass {
	item := schemaClass{}

	// Extract title for caption
	if title, ok := defMap["title"].(string); ok {
		item.Caption = title
	}

	// Extract properties
	props, ok := defMap["properties"].(map[string]any)
	if !ok {
		return item
	}

	// Extract name
	if nameField, ok := props["name"].(map[string]any); ok {
		if constVal, ok := nameField["const"].(string); ok {
			item.Name = constVal
		}
	}

	// Extract ID
	if idField, ok := props["id"].(map[string]any); ok {
		if constVal, ok := idField["const"].(float64); ok {
			item.ID = int(constVal)
		}
	}

	return item
}
