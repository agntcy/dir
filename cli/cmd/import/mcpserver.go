// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package importcmd

import (
	"fmt"
	"os/exec"
	"runtime"

	enricherconfig "github.com/agntcy/dir-importer/enricher/config"
)

const (
	// dirMCPServerKey is the tool host entry the LLM enricher requires.
	dirMCPServerKey = "dir-mcp-server"

	// dirMCPBinary is the command the default tool host launches.
	dirMCPBinary = "dir-mcp"

	dirMCPReleasesURL = "https://github.com/agntcy/dir-mcp/releases/latest"
)

// lookPath is swapped by tests.
var lookPath = exec.LookPath

// checkMCPServerInstalled verifies the LLM enricher's MCP server command can be
// found, so a missing binary fails up front with install instructions rather
// than as an opaque subprocess error mid-import. It is a no-op for any other
// enricher.
func checkMCPServerInstalled(cfg enricherconfig.Config) error {
	if cfg.LLM == nil {
		return nil
	}

	srv, ok := cfg.LLM.ToolHost.MCPServers[dirMCPServerKey]
	if !ok || srv.Command == "" {
		return nil
	}

	if _, err := lookPath(srv.Command); err == nil {
		return nil
	}

	return fmt.Errorf(`LLM enrichment needs the dir-mcp server, but %q was not found.

Install it from %s:
  1. Download the asset for your platform: mcp-server-%s-%s
  2. Make it executable and put it on your PATH as %q:
       chmod +x mcp-server-%s-%s
       mv mcp-server-%s-%s /usr/local/bin/%s
Or point tool_host.mcp_servers.%s.command in your import config at an existing binary.
Alternatively, use extractor enrichment, which needs no external server`,
		srv.Command, dirMCPReleasesURL,
		runtime.GOOS, runtime.GOARCH,
		dirMCPBinary,
		runtime.GOOS, runtime.GOARCH,
		runtime.GOOS, runtime.GOARCH, dirMCPBinary,
		dirMCPServerKey)
}
