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
	RegisterFormatter(FormatMCPCursor, &mcpCursorFormatter{})
}

// CursorMCPServer models a single entry under "mcpServers" in Cursor's
// mcp.json configuration file (.cursor/mcp.json project-scoped, or
// ~/.cursor/mcp.json global).
//
// Command/Args/Env describe a local, stdio-launched server. URL/Headers
// describe a remote (SSE or streamable HTTP) server -- Cursor has no "type"
// field and infers the transport from the presence of URL.
// See https://cursor.com/docs/context/mcp.
type CursorMCPServer struct {
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

// CursorMCPConfig models Cursor's mcp.json file.
type CursorMCPConfig struct {
	MCPServers map[string]CursorMCPServer `json:"mcpServers"`
}

type mcpCursorFormatter struct{}

func (f *mcpCursorFormatter) Format(record *corev1.Record) ([]byte, error) {
	data := record.GetData()
	if data == nil {
		return nil, fmt.Errorf("record contains no data")
	}

	cfg, err := RecordToCursor(data)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to translate record to Cursor MCP config: %w", ErrUnsupportedRecord, err)
	}

	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Cursor MCP config to JSON: %w", err)
	}

	raw = append(raw, '\n')

	return raw, nil
}

func (f *mcpCursorFormatter) FileExtension() string {
	return ExtJSON
}

// RecordToCursor translates a record's "integration/mcp" module (OASF 1.0.0
// "connections" format) into a CursorMCPConfig. Exported (like
// translator.RecordToGHCopilot) so cli/cmd/export's batch exporter can merge
// several records' configs into one mcp.json without round-tripping through
// JSON.
//
// This is a local implementation rather than a call to an oasf-sdk
// translator function (issue #1385 proposes a RecordToCursor in oasf-sdk's
// translator package). Keeping it here lets the format ship without
// blocking on an oasf-sdk release; it can move upstream later without
// changing this package's public surface.
//
// Only the OASF 1.0.0 "connections" array format is supported. The legacy
// 0.7.0/0.8.0 "servers" array format (still handled by
// translator.RecordToGHCopilot for GitHub Copilot) is not.
func RecordToCursor(record *structpb.Struct) (*CursorMCPConfig, error) {
	found, moduleStruct := recordutil.GetModule(record, translator.MCPModuleName)
	if !found {
		return nil, errors.New("MCP module not found in record")
	}

	moduleData := moduleStruct.GetFields()["data"].GetStructValue()
	if moduleData == nil {
		return nil, errors.New("MCP module has no data")
	}

	nameVal, ok := moduleData.GetFields()["name"]
	if !ok {
		return nil, errors.New("missing 'name' in MCP module data")
	}

	serverName := cursorNormalizeServerName(nameVal.GetStringValue())

	connectionsVal, ok := moduleData.GetFields()["connections"]
	if !ok {
		return nil, errors.New("missing 'connections' in MCP module data")
	}

	connectionsList := connectionsVal.GetListValue()
	if connectionsList == nil {
		return nil, errors.New("'connections' must be an array")
	}

	// Connections are alternative ways to reach the same server. Take the
	// first one we know how to represent (stdio, streamable-http, or sse).
	for _, connVal := range connectionsList.GetValues() {
		connMap := connVal.GetStructValue()
		if connMap == nil {
			continue
		}

		connType := connMap.GetFields()["type"].GetStringValue()

		server, ok := cursorServerFromConnection(connType, connMap)
		if !ok {
			continue
		}

		return &CursorMCPConfig{
			MCPServers: map[string]CursorMCPServer{serverName: server},
		}, nil
	}

	return nil, errors.New("no supported MCP connection found (stdio, streamable-http, sse)")
}

// cursorServerFromConnection builds a CursorMCPServer from a single OASF
// connection struct, given its "type" field. ok is false if connType is not
// one Cursor can represent. Cursor infers the transport from the config
// shape (command vs url), so remote connections carry no explicit type.
func cursorServerFromConnection(connType string, connMap *structpb.Struct) (CursorMCPServer, bool) {
	switch connType {
	case "stdio":
		return CursorMCPServer{
			Command: connMap.GetFields()["command"].GetStringValue(),
			Args:    cursorStringList(connMap.GetFields()["args"]),
			Env:     cursorEnvFromEnvVars(connMap.GetFields()["env_vars"]),
		}, true
	case "streamable-http", "sse":
		return CursorMCPServer{
			URL:     connMap.GetFields()["url"].GetStringValue(),
			Headers: cursorStringMap(connMap.GetFields()["headers"]),
		}, true
	default:
		return CursorMCPServer{}, false
	}
}

// cursorNormalizeServerName strips common MCP server-name suffixes
// (mirroring translator.RecordToGHCopilot's normalization) so that, e.g.,
// "github-mcp-server" becomes "github" across every MCP export format in
// this repo.
func cursorNormalizeServerName(name string) string {
	name = strings.TrimSuffix(name, "-mcp-server")
	name = strings.TrimSuffix(name, "-server")
	name = strings.TrimSuffix(name, "-mcp")

	return name
}

// cursorStringList reads a structpb list-of-strings value. Returns nil
// (omitted from JSON output) if val is not a non-empty list.
func cursorStringList(val *structpb.Value) []string {
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

// cursorStringMap reads a structpb struct value as a flat string map.
// Returns nil (omitted from JSON output) if val is not a non-empty struct.
func cursorStringMap(val *structpb.Value) map[string]string {
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

// cursorEnvFromEnvVars converts an OASF 1.0.0 "env_vars" array (a list of
// {name, description, default_value} objects) into the flat env map Cursor's
// mcp.json expects.
//
// When an entry has no default_value, this emits "${env:NAME}" so Cursor
// resolves it from the user's environment at launch time (Cursor supports
// ${env:NAME} interpolation in command, args, env, url, and headers). This
// differs from translator.RecordToGHCopilot, which emits VS Code's
// "${input:NAME}" prompt placeholder for the same case -- that mechanism is
// VS Code/GH Copilot specific and has no Cursor equivalent.
func cursorEnvFromEnvVars(val *structpb.Value) map[string]string {
	list := val.GetListValue()
	if list == nil || len(list.GetValues()) == 0 {
		return nil
	}

	env := make(map[string]string, len(list.GetValues()))

	for _, v := range list.GetValues() {
		envVar := v.GetStructValue()
		if envVar == nil {
			continue
		}

		name := envVar.GetFields()["name"].GetStringValue()
		if name == "" {
			continue
		}

		if def := envVar.GetFields()["default_value"].GetStringValue(); def != "" {
			env[name] = def
		} else {
			env[name] = fmt.Sprintf("${env:%s}", name)
		}
	}

	if len(env) == 0 {
		return nil
	}

	return env
}
