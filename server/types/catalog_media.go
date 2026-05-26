// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"strings"

	"github.com/agntcy/oasf-sdk/pkg/translator"
)

// AI Catalog media-type registry.
//
// This file is the single source of truth for the OASF integration
// module ↔ AI Catalog media-type mapping defined in §3.3 / §4.1 of the
// Agent Finder Specification.

// CatalogContainerMediaType is the AI Catalog media type used for
// container entries (records that bundle multiple modules) and for the
// WellKnownCatalog document itself.
const CatalogContainerMediaType = "application/ai-catalog+json"

// catalogMediaTypes maps an OASF integration module name onto the
// canonical AI Catalog media type that records carrying that module
// advertise.
var catalogMediaTypes = map[string]string{
	translator.A2AModuleName:         "application/a2a-agent-card+json",
	translator.MCPModuleName:         "application/mcp-server+json",
	translator.AgentSkillsModuleName: "application/ai-skill+md",
}

// CatalogMediaTypeForModule returns the AI Catalog media type advertised
// by an OASF integration module, or ("", false) for unknown modules.
// Callers MAY fall back to a generic media type or skip the entry.
func CatalogMediaTypeForModule(oasfModule string) (string, bool) {
	mt, ok := catalogMediaTypes[oasfModule]

	return mt, ok
}

// OASFModuleForMediaType returns the OASF module name behind an
// AI Catalog media type.
func OASFModuleForMediaType(mediaType string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(mediaType)) {
	case "application/a2a-agent-card+json":
		return translator.A2AModuleName, true
	case "application/mcp-server+json":
		return translator.MCPModuleName, true
	case "application/ai-skill", "application/ai-skill+md":
		return translator.AgentSkillsModuleName, true
	default:
		return "", false
	}
}
