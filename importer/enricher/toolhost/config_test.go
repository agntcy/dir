// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package toolhost

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFileConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "enricher.json")

	content := `{
  "mcpServers": {
    "dir-mcp-server": {
      "command": "dirctl",
      "args": ["mcp", "serve"]
    }
  },
  "model": "azure:gpt-4o",
  "max-steps": 7
}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFileConfig(path)
	if err != nil {
		t.Fatalf("LoadFileConfig: %v", err)
	}

	if cfg.Model != "azure:gpt-4o" {
		t.Fatalf("model: got %q", cfg.Model)
	}

	if cfg.MaxSteps != 7 {
		t.Fatalf("max-steps: got %d", cfg.MaxSteps)
	}

	srv, err := cfg.DirMCPServer()
	if err != nil {
		t.Fatal(err)
	}

	if srv.Command != "dirctl" {
		t.Fatalf("command: %q", srv.Command)
	}

	if len(srv.Args) != 2 || srv.Args[0] != "mcp" || srv.Args[1] != "serve" {
		t.Fatalf("args: %#v", srv.Args)
	}
}

func TestLoadFileConfig_rejectsWithoutDirMCPServer(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "enricher.json")

	content := `{
  "mcpServers": {
    "other-server": {
      "command": "dirctl",
      "args": ["mcp", "serve"]
    }
  },
  "model": "azure:gpt-4o",
  "max-steps": 7
}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := LoadFileConfig(path)
	if err == nil {
		t.Fatal("expected error when mcpServers has no dir-mcp-server entry")
	}
}
