// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:dupl // Intentional duplication with skills file for separate domain/skill handling
package tools

import (
	"context"
	"fmt"

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
	availableVersions, err := validateVersion(input.Version)
	if err != nil {
		//nolint:nilerr // MCP tools communicate errors through output, not error return
		return nil, GetSchemaDomainsOutput{
			ErrorMessage:      err.Error(),
			AvailableVersions: availableVersions,
		}, nil
	}

	domainsJSON, err := validator.GetSchemaDomains(input.Version)
	if err != nil {
		//nolint:nilerr // MCP tools communicate errors through output, not error return
		return nil, GetSchemaDomainsOutput{
			Version:           input.Version,
			ErrorMessage:      fmt.Sprintf("Failed to get domains from OASF %s schema: %v", input.Version, err),
			AvailableVersions: availableVersions,
		}, nil
	}

	allDomains, err := parseSchemaData(domainsJSON, parseItemFromSchema)
	if err != nil {
		//nolint:nilerr // MCP tools communicate errors through output, not error return
		return nil, GetSchemaDomainsOutput{
			Version:           input.Version,
			ErrorMessage:      err.Error(),
			AvailableVersions: availableVersions,
		}, nil
	}

	resultDomains, err := filterDomains(allDomains, input.ParentDomain)
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

// filterDomains filters domains based on parent parameter.
func filterDomains(allDomains []schemaClass, parent string) ([]schemaClass, error) {
	if parent != "" {
		return filterChildItems(allDomains, parent)
	}

	return extractTopLevelCategories(allDomains), nil
}

// convertToDomainItems converts generic schema items to DomainItem type.
func convertToDomainItems(items []schemaClass) []DomainItem {
	domains := make([]DomainItem, len(items))

	for i, item := range items {
		domains[i] = DomainItem(item)
	}

	return domains
}
