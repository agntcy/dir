// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"fmt"
	"strings"

	mcpapiv0 "github.com/modelcontextprotocol/registry/pkg/api/v0"
	"google.golang.org/protobuf/types/known/structpb"
)

// SourceKind identifies which payload is set on SourceItem.
type SourceKind int

const (
	// SourceKindMCP is an MCP registry/listing payload ([mcpapiv0.ServerResponse]).
	SourceKindMCP SourceKind = iota
	// SourceKindA2A is an A2A AgentCard as [structpb.Struct] (JSON object).
	SourceKindA2A
)

// SourceItem is one record from fetch through dedup before OASF transformation.
// SourceItem.Kind selects which field is valid.
type SourceItem struct {
	Kind SourceKind
	MCP  mcpapiv0.ServerResponse
	A2A  *structpb.Struct
}

// MCPSourceItem wraps an MCP server response for the pipeline.
func MCPSourceItem(s mcpapiv0.ServerResponse) SourceItem {
	return SourceItem{Kind: SourceKindMCP, MCP: s}
}

// A2ASourceItem wraps an AgentCard as structpb.Struct for the pipeline.
func A2ASourceItem(card *structpb.Struct) SourceItem {
	return SourceItem{Kind: SourceKindA2A, A2A: card}
}

// NameVersion returns "name@version" for deduplication, or "" if it cannot be derived.
func (s SourceItem) NameVersion() string {
	switch s.Kind {
	case SourceKindMCP:
		if s.MCP.Server.Name != "" && s.MCP.Server.Version != "" {
			return fmt.Sprintf("%s@%s", s.MCP.Server.Name, s.MCP.Server.Version)
		}
	case SourceKindA2A:
		if s.A2A == nil {
			return ""
		}

		name, version := a2aNameVersionFields(s.A2A)
		if name == "" {
			return ""
		}

		if version == "" {
			version = "v1.0.0"
		}

		return fmt.Sprintf("%s@%s", name, version)
	}

	return ""
}

func a2aNameVersionFields(card *structpb.Struct) (string, string) {
	if card == nil {
		return "", ""
	}

	var name, version string

	fields := card.GetFields()
	if v, ok := fields["name"]; ok {
		name = strings.TrimSpace(v.GetStringValue())
	}

	if v, ok := fields["version"]; ok {
		version = strings.TrimSpace(v.GetStringValue())
	}

	return name, version
}
