// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package importcmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	enricherconfig "github.com/agntcy/dir-importer/enricher/config"
	scannerconfig "github.com/agntcy/dir-importer/scanner/config"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestOptions builds an isolated options struct and flag set, then parses the
// given command-line args. It registers the flags that loadConfig consults
// (mirroring the subset of import.go's init relevant to config loading) so each
// test runs against a fresh, isolated flag set rather than the global command.
func newTestOptions(t *testing.T, args ...string) (*options, *pflag.FlagSet) {
	t.Helper()

	o := &options{}
	flags := pflag.NewFlagSet("import", pflag.ContinueOnError)

	flags.StringVar(&o.ConfigFile, "config", "", "")
	flags.StringVar(&o.FilePath, "file-path", "", "")
	flags.StringVar(&o.TypeFlag, "type", "", "")
	flags.StringVar(&o.RegistryURL, "url", "", "")
	flags.StringToStringVar(&o.Filters, "filter", nil, "")
	flags.IntVar(&o.Limit, "limit", 0, "")
	flags.BoolVar(&o.Sign, "sign", false, "")
	flags.StringVar(&o.OutputCIDFile, "output-cids", "", "")
	flags.BoolVar(&o.DryRun, "dry-run", false, "")
	flags.StringVar(&o.OutputDir, "output-dir", "", "")
	flags.BoolVar(&o.Force, "force", false, "")
	flags.BoolVar(&o.Debug, "debug", false, "")

	require.NoError(t, flags.Parse(args))

	return o, flags
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "import.config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	return path
}

func TestTypeAndURLFromFlagsWithoutFile(t *testing.T) {
	o, flags := newTestOptions(t, "--type", "mcp-registry", "--url", "https://registry.example.com/v0.1")

	require.NoError(t, o.loadConfig(flags))

	assert.Equal(t, "mcp-registry", string(o.Type))
	assert.Equal(t, "https://registry.example.com/v0.1", o.RegistryURL)
	require.NoError(t, o.Validate())
}

func TestLoadConfigDefaultsWithoutFile(t *testing.T) {
	o, flags := newTestOptions(t)

	require.NoError(t, o.loadConfig(flags))

	assert.Equal(t, scannerconfig.DefaultTimeout, o.Scanner.Timeout)
	assert.Equal(t, scannerconfig.DefaultScannerEnabled, o.Scanner.Enabled)
	assert.Equal(t, enricherconfig.DefaultRequestsPerMinute, o.Enricher.RequestsPerMinute)
	// tool-host falls back to the embedded default enricher.json.
	assert.Equal(t, "azure:gpt-4o", o.Enricher.ToolHost.Model)
}

func TestLoadConfigFromFile(t *testing.T) {
	path := writeConfig(t, `
type: mcp-registry
url: https://registry.example.com/v0.1
filters:
  search: analytics
limit: 100
scanner:
  enabled: true
  timeout: 30s
  cli_path: /usr/local/bin/mcp-scanner
  fail_on_error: true
enricher:
  requests_per_minute: 7
  tool_host:
    model: azure:gpt-4o
    max_steps: 12
    mcp_servers:
      dir-mcp-server:
        command: dirctl
        args: [mcp, serve]
        env:
          OASF_API_VALIDATION_SCHEMA_URL: https://schema.oasf.outshift.com
`)

	o, flags := newTestOptions(t, "--config", path)
	require.NoError(t, o.loadConfig(flags))

	assert.Equal(t, "mcp-registry", string(o.Type))
	assert.Equal(t, "https://registry.example.com/v0.1", o.RegistryURL)
	assert.Equal(t, map[string]string{"search": "analytics"}, o.Filters)
	assert.Equal(t, 100, o.Limit)
	assert.True(t, o.Scanner.Enabled)
	assert.Equal(t, 30*time.Second, o.Scanner.Timeout)
	assert.Equal(t, "/usr/local/bin/mcp-scanner", o.Scanner.CLIPath)
	assert.True(t, o.Scanner.FailOnError)
	assert.Equal(t, 7, o.Enricher.RequestsPerMinute)
	assert.Equal(t, 12, o.Enricher.ToolHost.MaxSteps)
}

// TestSignIsFlagOnly verifies that signing is driven by flags and that a sign:
// block in the config file does not enable signing.
func TestSignIsFlagOnly(t *testing.T) {
	path := writeConfig(t, `
type: mcp-registry
url: https://registry.example.com/v0.1
sign:
  enabled: true
`)

	// Config requests signing, but no --sign flag: signing stays off.
	o, flags := newTestOptions(t, "--config", path)
	require.NoError(t, o.loadConfig(flags))
	assert.False(t, o.Sign)

	// --sign flag enables it regardless of the config file.
	o, flags = newTestOptions(t, "--config", path, "--sign")
	require.NoError(t, o.loadConfig(flags))
	assert.True(t, o.Sign)
}

// TestToolHostEnvKeysPreserveCase guards against viper lowercasing env var
// names in the enricher tool-host MCP server configuration.
func TestToolHostEnvKeysPreserveCase(t *testing.T) {
	path := writeConfig(t, `
type: mcp-registry
url: https://registry.example.com/v0.1
enricher:
  tool_host:
    model: azure:gpt-4o
    mcp_servers:
      dir-mcp-server:
        command: dirctl
        args: [mcp, serve]
        env:
          OASF_API_VALIDATION_SCHEMA_URL: https://schema.oasf.outshift.com
          DIRECTORY_CLIENT_AUTH_MODE: insecure
`)

	o, flags := newTestOptions(t, "--config", path)
	require.NoError(t, o.loadConfig(flags))

	server, ok := o.Enricher.ToolHost.MCPServers["dir-mcp-server"]
	require.True(t, ok)
	assert.Equal(t, "https://schema.oasf.outshift.com", server.Env["OASF_API_VALIDATION_SCHEMA_URL"])
	assert.Equal(t, "insecure", server.Env["DIRECTORY_CLIENT_AUTH_MODE"])
}

func TestFlagOverridesConfigFile(t *testing.T) {
	path := writeConfig(t, `
type: mcp-registry
url: https://config.example.com/v0.1
limit: 100
`)

	o, flags := newTestOptions(t, "--config", path, "--url", "https://flag.example.com/v0.1", "--limit", "5")
	require.NoError(t, o.loadConfig(flags))

	assert.Equal(t, "https://flag.example.com/v0.1", o.RegistryURL)
	assert.Equal(t, 5, o.Limit)
}

func TestFilterFlagOverridesConfigFilters(t *testing.T) {
	path := writeConfig(t, `
type: mcp-registry
url: https://config.example.com/v0.1
filters:
  search: analytics
`)

	o, flags := newTestOptions(t, "--config", path, "--filter", "search=devtools")
	require.NoError(t, o.loadConfig(flags))

	assert.Equal(t, map[string]string{"search": "devtools"}, o.Filters)
}

func TestLoadConfigSkipEnricher(t *testing.T) {
	path := writeConfig(t, `
type: mcp-registry
url: https://registry.example.com/v0.1
enricher:
  skip_enricher: true
  skills:
    - name: natural_language_processing/text_completion
      id: 10201
  domains:
    - name: technology
      id: 1
`)

	o, flags := newTestOptions(t, "--config", path)
	require.NoError(t, o.loadConfig(flags))

	assert.True(t, o.Enricher.SkipEnricher)
	require.Len(t, o.Enricher.Skills, 1)
	assert.Equal(t, "natural_language_processing/text_completion", o.Enricher.Skills[0].GetName())
	assert.Equal(t, uint32(10201), o.Enricher.Skills[0].GetId())
	require.Len(t, o.Enricher.Domains, 1)
	assert.Equal(t, "technology", o.Enricher.Domains[0].GetName())
	assert.Equal(t, uint32(1), o.Enricher.Domains[0].GetId())

	// Skipping enrichment must not require a tool host, and the config must validate.
	require.NoError(t, o.Enricher.Validate())
}

// TestReferenceConfigIsValid ensures the committed reference file keeps parsing.
func TestReferenceConfigIsValid(t *testing.T) {
	o, flags := newTestOptions(t, "--config", "import.config.yaml")
	require.NoError(t, o.loadConfig(flags))

	assert.Equal(t, "mcp-registry", string(o.Type))
	assert.Equal(t, "azure:gpt-4o", o.Enricher.ToolHost.Model)

	server, ok := o.Enricher.ToolHost.MCPServers["dir-mcp-server"]
	require.True(t, ok)
	assert.Equal(t, "https://schema.oasf.outshift.com", server.Env["OASF_API_VALIDATION_SCHEMA_URL"])
}
