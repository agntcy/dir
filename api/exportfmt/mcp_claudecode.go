// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package exportfmt

import (
	"encoding/json"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

func init() {
	RegisterFormatter(FormatMCPClaudeCode, &mcpClaudeCodeFormatter{})
}

// ClaudeCodeMCPServer models a single entry under "mcpServers" in Claude
// Code's project-scoped .mcp.json configuration file.
//
// Command/Args/Env describe a local, stdio-launched server. URL/Headers
// describe a remote server; Type is only set ("http" or "sse") for remote
// servers -- Claude Code has no "type" field for stdio and infers it from
// the presence of Command. See https://code.claude.com/docs/en/mcp.
type ClaudeCodeMCPServer struct {
	Type    string            `json:"type,omitempty"`
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

// ClaudeCodeMCPConfig models Claude Code's project-scoped .mcp.json file.
type ClaudeCodeMCPConfig struct {
	MCPServers map[string]ClaudeCodeMCPServer `json:"mcpServers"`
}

type mcpClaudeCodeFormatter struct{}

func (f *mcpClaudeCodeFormatter) Format(record *corev1.Record) ([]byte, error) {
	data := record.GetData()
	if data == nil {
		return nil, fmt.Errorf("record contains no data")
	}

	cfg, err := RecordToClaudeCode(data)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to translate record to Claude Code MCP config: %w", ErrUnsupportedRecord, err)
	}

	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Claude Code MCP config to JSON: %w", err)
	}

	raw = append(raw, '\n')

	return raw, nil
}

func (f *mcpClaudeCodeFormatter) FileExtension() string {
	return ExtJSON
}

// RecordToClaudeCode translates a record's "integration/mcp" module (OASF
// 1.0.0 "connections" format) into a ClaudeCodeMCPConfig. Exported (like
// translator.RecordToGHCopilot) so cli/cmd/export's batch exporter can merge
// several records' configs into one .mcp.json without round-tripping
// through JSON.
//
// This is a local implementation rather than a call to an oasf-sdk
// translator function (issue #1386 proposes a RecordToClaudeCode in
// oasf-sdk's translator package). Keeping it here lets the format ship
// without blocking on an oasf-sdk release; it can move upstream later
// without changing this package's public surface.
//
// Only the OASF 1.0.0 "connections" array format is supported. The legacy
// 0.7.0/0.8.0 "servers" array format (still handled by
// translator.RecordToGHCopilot for GitHub Copilot) is not.
func RecordToClaudeCode(record *structpb.Struct) (*ClaudeCodeMCPConfig, error) {
	name, server, err := recordToMCPServer(record, claudeCodeServerFromConnection)
	if err != nil {
		return nil, err
	}

	return &ClaudeCodeMCPConfig{
		MCPServers: map[string]ClaudeCodeMCPServer{name: server},
	}, nil
}

// claudeCodeServerFromConnection builds a ClaudeCodeMCPServer from a single
// OASF connection struct, given its "type" field. ok is false if connType is
// not one Claude Code can represent.
func claudeCodeServerFromConnection(connType string, connMap *structpb.Struct) (ClaudeCodeMCPServer, bool) {
	switch connType {
	case connTypeStdio:
		return ClaudeCodeMCPServer{
			Command: connMap.GetFields()["command"].GetStringValue(),
			Args:    mcpStringList(connMap.GetFields()["args"]),
			Env:     claudeCodeEnvFromEnvVars(connMap.GetFields()["env_vars"]),
		}, true
	case connTypeStreamableHTTP, connTypeSSE:
		return ClaudeCodeMCPServer{
			Type:    claudeCodeConnectionType(connType),
			URL:     connMap.GetFields()["url"].GetStringValue(),
			Headers: mcpStringMap(connMap.GetFields()["headers"]),
		}, true
	default:
		return ClaudeCodeMCPServer{}, false
	}
}

// claudeCodeConnectionType maps an OASF connection "type" to the value
// Claude Code expects for a remote server's "type" field. OASF's
// "streamable-http" becomes Claude Code's "http" -- the two schemas use
// different vocabulary for the same transport.
func claudeCodeConnectionType(oasfType string) string {
	if oasfType == connTypeStreamableHTTP {
		return "http"
	}

	return oasfType
}

// claudeCodeEnvFromEnvVars converts an OASF 1.0.0 "env_vars" array (a list of
// {name, description, default_value} objects) into the flat env map Claude
// Code's .mcp.json expects.
//
// When an entry has no default_value, this emits "${NAME}" so Claude Code
// resolves it from the user's shell environment at launch time (Claude Code
// supports $VAR / ${VAR} expansion in .mcp.json). This differs from
// translator.RecordToGHCopilot, which emits VS Code's "${input:NAME}" prompt
// placeholder for the same case -- that mechanism is VS Code/GH Copilot
// specific and has no Claude Code equivalent.
func claudeCodeEnvFromEnvVars(val *structpb.Value) map[string]string {
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
			env[name] = fmt.Sprintf("${%s}", name)
		}
	}

	if len(env) == 0 {
		return nil
	}

	return env
}
