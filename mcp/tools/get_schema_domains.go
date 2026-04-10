// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:dupl // Intentional duplication with skills file for separate domain/skill handling
package tools

import (
	"context"
	"fmt"
	"slices"

	"github.com/agntcy/oasf-sdk/pkg/schema"
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
func (t *Tools) GetSchemaDomains(ctx context.Context, _ *mcp.CallToolRequest, input GetSchemaDomainsInput) (
	*mcp.CallToolResult,
	GetSchemaDomainsOutput,
	error,
) {
	availableVersions, err := validateVersion(ctx, input.Version)
	if err != nil {
		//nolint:nilerr // MCP tools communicate errors through output, not error return
		return nil, GetSchemaDomainsOutput{
			ErrorMessage:      err.Error(),
			AvailableVersions: availableVersions,
		}, nil
	}

	// Get schema instance
	schemaInstance, err := getSchemaInstance()
	if err != nil {
		//nolint:nilerr // MCP tools communicate errors through output, not error return
		return nil, GetSchemaDomainsOutput{
			Version:           input.Version,
			ErrorMessage:      fmt.Sprintf("Failed to initialize schema client: %v", err),
			AvailableVersions: availableVersions,
		}, nil
	}

	// Get domains taxonomy using the schema package with explicit schema version option.
	domainTaxonomy, err := schemaInstance.GetSchemaDomains(ctx, schema.WithSchemaVersion(input.Version))
	if err != nil {
		//nolint:nilerr // MCP tools communicate errors through output, not error return
		return nil, GetSchemaDomainsOutput{
			Version:           input.Version,
			ErrorMessage:      fmt.Sprintf("Failed to get domains from OASF %s schema: %v", input.Version, err),
			AvailableVersions: availableVersions,
		}, nil
	}

	resultDomains, err := filterDomains(domainTaxonomy, input.ParentDomain)
	if err != nil {
		//nolint:nilerr // MCP tools communicate errors through output, not error return
		return nil, GetSchemaDomainsOutput{
			Version:           input.Version,
			ParentDomain:      input.ParentDomain,
			ErrorMessage:      err.Error(),
			AvailableVersions: availableVersions,
		}, nil
	}

	return nil, GetSchemaDomainsOutput{
		Version:           input.Version,
		Domains:           convertToDomainItems(resultDomains),
		ParentDomain:      input.ParentDomain,
		AvailableVersions: availableVersions,
	}, nil
}

// filterDomains filters domain taxonomy based on parent parameter.
func filterDomains(domainTaxonomy schema.Taxonomy, parent string) (schema.Taxonomy, error) {
	if parent == "" {
		return domainTaxonomy, nil
	}

	parentDomain, found := findTaxonomyItemByName(domainTaxonomy, parent)
	if !found || len(parentDomain.Classes) == 0 {
		return nil, fmt.Errorf("parent '%s' not found or has no children", parent)
	}

	return schema.Taxonomy(parentDomain.Classes), nil
}

// convertToDomainItems converts taxonomy map items to DomainItem type.
func convertToDomainItems(taxonomy schema.Taxonomy) []DomainItem {
	domains := make([]DomainItem, 0, len(taxonomy))

	for _, item := range taxonomy {
		domains = append(domains, DomainItem{
			Name:    item.Name,
			Caption: item.Caption,
			ID:      item.ID,
		})
	}

	slices.SortFunc(domains, func(a, b DomainItem) int {
		if a.Name < b.Name {
			return -1
		}

		if a.Name > b.Name {
			return 1
		}

		return 0
	})

	return domains
}
