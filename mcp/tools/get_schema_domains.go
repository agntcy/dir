// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/agntcy/oasf-sdk/pkg/validator"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// GetSchemaDomainsInput represents the input for getting OASF schema domains.
type GetSchemaDomainsInput struct {
	Version      string `json:"version"                 jsonschema:"OASF schema version to retrieve domains from (e.g., 0.7.0, 0.8.0)"`
	ParentDomain string `json:"parent_domain,omitempty" jsonschema:"Optional parent domain name to filter sub-domains (e.g., 'artificial_intelligence')"`
}

// DomainItem represents a domain in the OASF schema.
type DomainItem struct {
	Name    string `json:"name"`
	Caption string `json:"caption,omitempty"`
	ID      int    `json:"id,omitempty"`
}

// GetSchemaDomainsOutput represents the output after getting OASF schema domains.
type GetSchemaDomainsOutput struct {
	Version           string       `json:"version"                      jsonschema:"The requested OASF schema version"`
	Domains           []DomainItem `json:"domains"                      jsonschema:"List of domains (top-level or filtered by parent)"`
	ParentDomain      string       `json:"parent_domain,omitempty"      jsonschema:"The parent domain filter if specified"`
	ErrorMessage      string       `json:"error_message,omitempty"      jsonschema:"Error message if domain retrieval failed"`
	AvailableVersions []string     `json:"available_versions,omitempty" jsonschema:"List of available OASF schema versions"`
}

// GetSchemaDomains retrieves domains from the OASF schema for the specified version.
// If parent_domain is provided, returns only sub-domains under that parent.
// Otherwise, returns all top-level domains.
func GetSchemaDomains(_ context.Context, _ *mcp.CallToolRequest, input GetSchemaDomainsInput) (
	*mcp.CallToolResult,
	GetSchemaDomainsOutput,
	error,
) {
	// Get available schema versions from the OASF SDK
	availableVersions, err := validator.GetAvailableSchemaVersions()
	if err != nil {
		return nil, GetSchemaDomainsOutput{
			ErrorMessage: fmt.Sprintf("Failed to get available schema versions: %v", err),
		}, nil
	}

	// Validate the version parameter
	if input.Version == "" {
		return nil, GetSchemaDomainsOutput{
			ErrorMessage:      "Version parameter is required. Available versions: " + strings.Join(availableVersions, ", "),
			AvailableVersions: availableVersions,
		}, nil
	}

	// Check if the requested version is available
	versionValid := false
	for _, version := range availableVersions {
		if input.Version == version {
			versionValid = true
			break
		}
	}

	if !versionValid {
		return nil, GetSchemaDomainsOutput{
			ErrorMessage:      fmt.Sprintf("Invalid version '%s'. Available versions: %s", input.Version, strings.Join(availableVersions, ", ")),
			AvailableVersions: availableVersions,
		}, nil
	}

	// Get domains content using the OASF SDK
	domainsJSON, err := validator.GetSchemaDomains(input.Version)
	if err != nil {
		return nil, GetSchemaDomainsOutput{
			Version:           input.Version,
			ErrorMessage:      fmt.Sprintf("Failed to get domains from OASF %s schema: %v", input.Version, err),
			AvailableVersions: availableVersions,
		}, nil
	}

	// Parse domains JSON
	var domainsData map[string]interface{}
	if err := json.Unmarshal(domainsJSON, &domainsData); err != nil {
		return nil, GetSchemaDomainsOutput{
			Version:           input.Version,
			ErrorMessage:      fmt.Sprintf("Failed to parse domains data: %v", err),
			AvailableVersions: availableVersions,
		}, nil
	}

	// Parse all domains from the flat map structure
	// Each key is a domain short name, value contains the full name with hierarchy
	var allDomains []DomainItem
	for _, domainDef := range domainsData {
		defMap, ok := domainDef.(map[string]interface{})
		if !ok {
			continue
		}

		domain := parseDomainFromSchema(defMap)
		if domain.Name != "" {
			allDomains = append(allDomains, domain)
		}
	}

	// Filter based on parent_domain parameter
	var resultDomains []DomainItem
	if input.ParentDomain != "" {
		// Return domains that are children of the parent_domain
		// Children have names like "parent_domain/child_domain"
		prefix := input.ParentDomain + "/"
		for _, domain := range allDomains {
			if strings.HasPrefix(domain.Name, prefix) {
				// Check if this is a direct child (no further slashes after prefix)
				remainder := strings.TrimPrefix(domain.Name, prefix)
				if !strings.Contains(remainder, "/") {
					resultDomains = append(resultDomains, domain)
				}
			}
		}

		if len(resultDomains) == 0 {
			return nil, GetSchemaDomainsOutput{
				Version:           input.Version,
				ParentDomain:      input.ParentDomain,
				ErrorMessage:      fmt.Sprintf("Parent domain '%s' not found or has no children", input.ParentDomain),
				AvailableVersions: availableVersions,
			}, nil
		}
	} else {
		// Return only top-level parent categories
		// Extract unique parent categories from domain names (part before first "/")
		parentCategories := make(map[string]bool)
		for _, domain := range allDomains {
			if idx := strings.Index(domain.Name, "/"); idx > 0 {
				parentCategory := domain.Name[:idx]
				if !parentCategories[parentCategory] {
					parentCategories[parentCategory] = true
					// Create a domain item for the parent category
					resultDomains = append(resultDomains, DomainItem{
						Name: parentCategory,
					})
				}
			}
		}
	}

	// Return the domains
	return nil, GetSchemaDomainsOutput{
		Version:           input.Version,
		Domains:           resultDomains,
		ParentDomain:      input.ParentDomain,
		AvailableVersions: availableVersions,
	}, nil
}

// parseDomainFromSchema extracts domain information from the schema definition.
func parseDomainFromSchema(defMap map[string]interface{}) DomainItem {
	domain := DomainItem{}

	// Extract title for caption
	if title, ok := defMap["title"].(string); ok {
		domain.Caption = title
	}

	// Extract properties
	props, ok := defMap["properties"].(map[string]interface{})
	if !ok {
		return domain
	}

	// Extract name
	if nameField, ok := props["name"].(map[string]interface{}); ok {
		if constVal, ok := nameField["const"].(string); ok {
			domain.Name = constVal
		}
	}

	// Extract ID
	if idField, ok := props["id"].(map[string]interface{}); ok {
		if constVal, ok := idField["const"].(float64); ok {
			domain.ID = int(constVal)
		}
	}

	return domain
}
