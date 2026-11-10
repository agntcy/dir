// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"testing"

	"github.com/Agent-Card/ai-catalog-go/catalog"
	"github.com/Agent-Card/ai-catalog-go/validate"
)

// wellKnownShape is the document GetWellKnownCatalog serves: four nested
// catalogs, no collections, urn:air identifiers.
const wellKnownShape = `{
  "specVersion": "1.0",
  "host": {
    "displayName": "AGNTCY Directory",
    "identifier": "https://ai-catalog.example"
  },
  "entries": [
    {
      "identifier": "urn:air:org.agntcy:catalog:a2a",
      "displayName": "A2A Agents",
      "type": "application/ai-catalog+json",
      "url": "https://ai-catalog.example/v1/catalog?filter=type%3Dapplication%2Fa2a-agent-card%2Bjson",
      "tags": ["a2a"]
    },
    {
      "identifier": "urn:air:org.agntcy:catalog:mcp",
      "displayName": "MCP Servers",
      "type": "application/ai-catalog+json",
      "url": "https://ai-catalog.example/v1/catalog?filter=type%3Dapplication%2Fmcp-server-card%2Bjson",
      "tags": ["mcp"]
    },
    {
      "identifier": "urn:air:org.agntcy:catalog:skill",
      "displayName": "Agent Skills",
      "type": "application/ai-catalog+json",
      "url": "https://ai-catalog.example/v1/catalog?filter=type%3Dapplication%2Fagent-skills%2Bmd",
      "tags": ["skills"]
    },
    {
      "identifier": "urn:air:org.agntcy:catalog:skill-bundle",
      "displayName": "Agent Skill Bundles",
      "type": "application/ai-catalog+json",
      "url": "https://ai-catalog.example/v1/catalog?filter=type%3Dapplication%2Fagent-skills%2Bgzip",
      "tags": ["skills"]
    }
  ]
}`

const nestedCatalogShape = `{
  "specVersion": "1.0",
  "host": {
    "displayName": "AGNTCY Directory",
    "identifier": "https://ai-catalog.example"
  },
  "entries": []
}`

func TestSDKValidatesWellKnownShape(t *testing.T) {
	t.Parallel()

	doc, err := catalog.Parse([]byte(wellKnownShape))
	if err != nil {
		t.Fatalf("parse well-known catalog: %v", err)
	}

	if got := doc.GetByType(catalog.MediaTypeCatalog); len(got) != 4 {
		t.Fatalf("nested catalogs: got %d, want 4", len(got))
	}

	result := validate.Validate(doc)
	if !result.IsValid {
		t.Fatalf("well-known catalog invalid: %+v", result.Errors)
	}
}

func TestSDKValidatesNestedCatalogShape(t *testing.T) {
	t.Parallel()

	doc, err := catalog.Parse([]byte(nestedCatalogShape))
	if err != nil {
		t.Fatalf("parse nested catalog: %v", err)
	}

	result := validate.Validate(doc)
	if !result.IsValid {
		t.Fatalf("nested catalog invalid: %+v", result.Errors)
	}
}
