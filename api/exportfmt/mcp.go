// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package exportfmt

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	corev1 "github.com/agntcy/dir/api/core/v1"
	recordutil "github.com/agntcy/oasf-sdk/pkg/record"
	"github.com/agntcy/oasf-sdk/pkg/translator"
	"google.golang.org/protobuf/types/known/structpb"
)

func init() {
	RegisterFormatter(FormatMCPGHCopiot, &mcpGHCopilotFormatter{})
}

type mcpGHCopilotFormatter struct{}

func (f *mcpGHCopilotFormatter) Format(record *corev1.Record) ([]byte, error) {
	data := record.GetData()
	if data == nil {
		return nil, fmt.Errorf("record contains no data")
	}

	ghCopilotConfig, err := translator.RecordToGHCopilot(data)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to translate record to GitHub Copilot MCP config: %w", ErrUnsupportedRecord, err)
	}

	raw, err := json.MarshalIndent(ghCopilotConfig, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal GitHub Copilot MCP config to JSON: %w", err)
	}

	raw = append(raw, '\n')

	return raw, nil
}

func (f *mcpGHCopilotFormatter) FileExtension() string {
	return ExtJSON
}

// mcpNormalizeServerName strips common MCP server-name suffixes
// (mirroring translator.RecordToGHCopilot's normalization) so that, e.g.,
// "github-mcp-server" becomes "github" across every MCP export format in
// this repo.
func mcpNormalizeServerName(name string) string {
	name = strings.TrimSuffix(name, "-mcp-server")
	name = strings.TrimSuffix(name, "-server")
	name = strings.TrimSuffix(name, "-mcp")

	return name
}

// mcpStringList reads a structpb list-of-strings value. Returns nil
// (omitted from JSON output) if val is not a non-empty list.
func mcpStringList(val *structpb.Value) []string {
	list := val.GetListValue()
	if list == nil || len(list.GetValues()) == 0 {
		return nil
	}

	out := make([]string, 0, len(list.GetValues()))
	for _, v := range list.GetValues() {
		out = append(out, v.GetStringValue())
	}

	return out
}

// mcpStringMap reads a structpb struct value as a flat string map.
// Returns nil (omitted from JSON output) if val is not a non-empty struct.
func mcpStringMap(val *structpb.Value) map[string]string {
	s := val.GetStructValue()
	if s == nil || len(s.GetFields()) == 0 {
		return nil
	}

	out := make(map[string]string, len(s.GetFields()))
	for k, v := range s.GetFields() {
		out[k] = v.GetStringValue()
	}

	return out
}

// recordToMCPServer translates a record's "integration/mcp" module (OASF
// 1.0.0 "connections" format) into a single MCP server entry. The
// serverFromConnection callback builds the format-specific server struct
// from a single OASF connection.
func recordToMCPServer[S any](
	record *structpb.Struct,
	serverFromConnection func(connType string, connMap *structpb.Struct) (S, bool),
) (string, S, error) {
	var zero S

	found, moduleStruct := recordutil.GetModule(record, translator.MCPModuleName)
	if !found {
		return "", zero, errors.New("MCP module not found in record")
	}

	moduleData := moduleStruct.GetFields()["data"].GetStructValue()
	if moduleData == nil {
		return "", zero, errors.New("MCP module has no data")
	}

	nameVal, ok := moduleData.GetFields()["name"]
	if !ok {
		return "", zero, errors.New("missing 'name' in MCP module data")
	}

	serverName := mcpNormalizeServerName(nameVal.GetStringValue())

	connectionsVal, ok := moduleData.GetFields()["connections"]
	if !ok {
		return "", zero, errors.New("missing 'connections' in MCP module data")
	}

	connectionsList := connectionsVal.GetListValue()
	if connectionsList == nil {
		return "", zero, errors.New("'connections' must be an array")
	}

	// Connections are alternative ways to reach the same server. Take the
	// first one we know how to represent (stdio, streamable-http, or sse).
	for _, connVal := range connectionsList.GetValues() {
		connMap := connVal.GetStructValue()
		if connMap == nil {
			continue
		}

		connType := connMap.GetFields()["type"].GetStringValue()

		server, ok := serverFromConnection(connType, connMap)
		if !ok {
			continue
		}

		return serverName, server, nil
	}

	return "", zero, errors.New("no supported MCP connection found (stdio, streamable-http, sse)")
}
