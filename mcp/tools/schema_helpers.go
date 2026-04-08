// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"context"
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

		schemaInstance, err = schema.New(currentSchemaURL, schema.WithCache(true))
		if err != nil {
			return nil, fmt.Errorf("failed to create schema instance: %w", err)
		}

		schemaURL = currentSchemaURL
	}

	return schemaInstance, nil
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

// findTaxonomyItemByName returns the taxonomy node matching a parent name.
// It supports matching either the map key or the item's name field.
func findTaxonomyItemByName(taxonomy schema.Taxonomy, parent string) (schema.TaxonomyItem, bool) {
	for key, item := range taxonomy {
		if key == parent || item.Name == parent {
			return item, true
		}

		if len(item.Classes) > 0 {
			if found, ok := findTaxonomyItemByName(schema.Taxonomy(item.Classes), parent); ok {
				return found, true
			}
		}
	}

	return schema.TaxonomyItem{}, false
}
