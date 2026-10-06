// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package importcmd

import (
	"errors"
	"testing"

	enricherconfig "github.com/agntcy/dir-importer/enricher/config"
	"github.com/agntcy/dir-importer/enricher/toolhost"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func llmWithCommand(cmd string) enricherconfig.Config {
	return enricherconfig.Config{LLM: &enricherconfig.LLMConfig{
		ToolHost: toolhost.Config{MCPServers: map[string]toolhost.MCPServerConfig{
			dirMCPServerKey: {Command: cmd},
		}},
	}}
}

func TestCheckMCPServerInstalled(t *testing.T) {
	prev := lookPath

	t.Cleanup(func() { lookPath = prev })

	lookPath = func(string) (string, error) { return "", errors.New("not found") }

	err := checkMCPServerInstalled(llmWithCommand(dirMCPBinary))
	require.Error(t, err)
	assert.Contains(t, err.Error(), dirMCPReleasesURL)
	assert.Contains(t, err.Error(), "extractor")

	lookPath = func(string) (string, error) { return "/usr/local/bin/dir-mcp", nil }

	require.NoError(t, checkMCPServerInstalled(llmWithCommand(dirMCPBinary)))

	// Other enrichers never need the server.
	lookPath = func(string) (string, error) { return "", errors.New("not found") }

	require.NoError(t, checkMCPServerInstalled(enricherconfig.Config{}))
}

func TestDefaultToolHostUsesDirMCPBinary(t *testing.T) {
	srv, ok := defaultToolHostConfig().MCPServers[dirMCPServerKey]
	require.True(t, ok)
	assert.Equal(t, dirMCPBinary, srv.Command)
	assert.Empty(t, srv.Args)
}
