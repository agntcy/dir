// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package toolhost

import (
	"encoding/json"
	"fmt"
	"os"
)

// dirMCPServerKey is the only MCP server entry the enricher uses (stdio to dir MCP).
const dirMCPServerKey = "dir-mcp-server"

// FileConfig is the enricher JSON config (model, MCP servers, tool loop limits).
type FileConfig struct {
	MCPServers map[string]MCPServerConfig `json:"mcpServers"`
	Model      string                     `json:"model"`
	MaxSteps   int                        `json:"max-steps"`
}

// MCPServerConfig describes one MCP server (stdio).
type MCPServerConfig struct {
	Command string         `json:"command"`
	Args    []string       `json:"args"`
	Env     map[string]any `json:"env,omitempty"`
}

// LoadFileConfig reads and parses the enricher config file.
func LoadFileConfig(path string) (*FileConfig, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read enricher config: %w", err)
	}

	var cfg FileConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, fmt.Errorf("parse enricher config: %w", err)
	}

	if cfg.Model == "" {
		return nil, fmt.Errorf("enricher config: model is required")
	}

	if _, err := cfg.DirMCPServer(); err != nil {
		return nil, err
	}

	if cfg.MaxSteps <= 0 {
		cfg.MaxSteps = 10
	}

	return &cfg, nil
}

// DirMCPServer returns the required dir MCP stdio server; other mcpServers keys are ignored.
func (c *FileConfig) DirMCPServer() (*MCPServerConfig, error) {
	if c == nil {
		return nil, fmt.Errorf("enricher config: nil")
	}

	s, ok := c.MCPServers[dirMCPServerKey]
	if !ok {
		return nil, fmt.Errorf("enricher config: mcpServers must include %q", dirMCPServerKey)
	}

	if s.Command == "" {
		return nil, fmt.Errorf("enricher config: mcpServers[%q].command is required", dirMCPServerKey)
	}

	return &s, nil
}

// ExtraEnv returns KEY=value pairs to append to the subprocess environment (merged with os.Environ by mcp-go).
func (m *MCPServerConfig) ExtraEnv() []string {
	if m == nil || len(m.Env) == 0 {
		return nil
	}

	out := make([]string, 0, len(m.Env))
	for k, v := range m.Env {
		s, ok := v.(string)
		if !ok {
			continue
		}

		out = append(out, fmt.Sprintf("%s=%s", k, s))
	}

	return out
}
