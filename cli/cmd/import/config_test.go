// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package importcmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	enricherconfig "github.com/agntcy/dir-importer/enricher/config"
	"github.com/agntcy/dir-importer/transformer"
	"github.com/agntcy/oasf-sdk/pkg/validator"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
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
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600)) //nolint:gosec // G703: path is inside t.TempDir().

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

	require.NotNil(t, o.Enricher.LLM)
	assert.Equal(t, enricherconfig.DefaultRequestsPerMinute, o.Enricher.LLM.RequestsPerMinute)
	assert.Equal(t, "azure:gpt-4o", o.Enricher.LLM.ToolHost.Model)
}

func TestLoadConfigFromFile(t *testing.T) {
	path := writeConfig(t, `
type: mcp-registry
url: https://registry.example.com/v0.1
filters:
  search: analytics
limit: 100
enricher:
  llm:
    requests_per_minute: 7
    tool_host:
      model: azure:gpt-4o
      max_steps: 12
      mcp_servers:
        dir-mcp-server:
          command: dir-mcp
          env:
            OASF_API_VALIDATION_SCHEMA_URL: https://schema.oasf.outshift.com
`)

	o, flags := newTestOptions(t, "--config", path)
	require.NoError(t, o.loadConfig(flags))

	assert.Equal(t, "mcp-registry", string(o.Type))
	assert.Equal(t, "https://registry.example.com/v0.1", o.RegistryURL)
	assert.Equal(t, map[string]string{"search": "analytics"}, o.Filters)
	assert.Equal(t, 100, o.Limit)
	require.NotNil(t, o.Enricher.LLM)
	assert.Equal(t, 7, o.Enricher.LLM.RequestsPerMinute)
	assert.Equal(t, 12, o.Enricher.LLM.ToolHost.MaxSteps)
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
  llm:
    tool_host:
      model: azure:gpt-4o
      mcp_servers:
        dir-mcp-server:
          command: dir-mcp
          env:
            OASF_API_VALIDATION_SCHEMA_URL: https://schema.oasf.outshift.com
            DIRECTORY_CLIENT_AUTH_MODE: insecure
`)

	o, flags := newTestOptions(t, "--config", path)
	require.NoError(t, o.loadConfig(flags))

	require.NotNil(t, o.Enricher.LLM)
	server, ok := o.Enricher.LLM.ToolHost.MCPServers["dir-mcp-server"]
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

func TestLoadConfigExtractorEnricherRequiresProvisionedAssets(t *testing.T) {
	t.Parallel()

	path := writeConfig(t, `
type: mcp-registry
url: https://registry.example.com/v0.1
enricher:
  extractor:
    asset_dir: `+filepath.Join(t.TempDir(), "missing-extractor")+`
`)

	o, flags := newTestOptions(t, "--config", path)
	err := o.loadConfig(flags)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "extractor enricher")
	assert.Contains(t, err.Error(), "dirctl init")
}

func TestLoadConfigStaticEnricher(t *testing.T) {
	path := writeConfig(t, `
type: mcp-registry
url: https://registry.example.com/v0.1
enricher:
  static:
    skills:
      - name: natural_language_processing/text_completion
        id: 10201
    domains:
      - name: technology
        id: 1
`)

	o, flags := newTestOptions(t, "--config", path)
	require.NoError(t, o.loadConfig(flags))

	require.NotNil(t, o.Enricher.Static)
	require.Len(t, o.Enricher.Static.Skills, 1)
	assert.Equal(t, "natural_language_processing/text_completion", o.Enricher.Static.Skills[0].GetName())
	assert.Equal(t, uint32(10201), o.Enricher.Static.Skills[0].GetId())
	require.Len(t, o.Enricher.Static.Domains, 1)
	assert.Equal(t, "technology", o.Enricher.Static.Domains[0].GetName())
	assert.Equal(t, uint32(1), o.Enricher.Static.Domains[0].GetId())

	require.NoError(t, o.Enricher.Validate())
}

// TestReferenceConfigIsValid ensures the committed reference file keeps parsing.
func TestReferenceConfigIsValid(t *testing.T) {
	o, flags := newTestOptions(t, "--config", "import.config.yaml")
	require.NoError(t, o.loadConfig(flags))

	assert.Equal(t, "mcp-registry", string(o.Type))
	require.NotNil(t, o.Enricher.LLM)
	assert.Equal(t, "azure:gpt-4o", o.Enricher.LLM.ToolHost.Model)

	server, ok := o.Enricher.LLM.ToolHost.MCPServers["dir-mcp-server"]
	require.True(t, ok)
	assert.Equal(t, "https://schema.oasf.outshift.com", server.Env["OASF_API_VALIDATION_SCHEMA_URL"])
}

// TestReferenceStaticEnricherIsValid uncomments the static enricher example in
// the reference file and validates its taxonomy against OASF.
func TestReferenceStaticEnricherIsValid(t *testing.T) {
	raw, err := os.ReadFile("import.config.yaml")
	require.NoError(t, err)

	var (
		lines    []string
		inStatic bool
	)

	for line := range strings.SplitSeq(string(raw), "\n") {
		switch {
		case strings.TrimSpace(line) == "# static:":
			inStatic = true
			line = "  static:"
		case inStatic && strings.HasPrefix(line, "  #   "):
			line = "  " + strings.TrimPrefix(line, "  # ")
		default:
			inStatic = false
		}

		lines = append(lines, line)
	}

	// Swap the active llm block out so only the static enricher remains.
	cfg := strings.Join(lines, "\n")
	start := strings.Index(cfg, "  llm:")
	end := strings.Index(cfg, "  static:")

	require.NotEqual(t, -1, start)
	require.Greater(t, end, start)

	path := writeConfig(t, cfg[:start]+cfg[end:])

	o, flags := newTestOptions(t, "--config", path)
	require.NoError(t, o.loadConfig(flags))

	require.NotNil(t, o.Enricher.Static)
	require.NoError(t, o.Enricher.Validate())

	validateStaticEnricherAgainstOASF(t, o.Enricher.Static)
}

// validateStaticEnricherAgainstOASF stamps the static skills/domains on a minimal
// record and validates it against the live OASF schema server, so renamed or
// removed taxonomy classes are caught. It runs only when OASF_SCHEMA_URL is set,
// keeping the default unit run off the network:
//
//	OASF_SCHEMA_URL=https://schema.oasf.outshift.com go test ./cmd/import/...
func validateStaticEnricherAgainstOASF(t *testing.T, static *enricherconfig.StaticConfig) {
	t.Helper()

	schemaURL := os.Getenv("OASF_SCHEMA_URL")
	if schemaURL == "" {
		t.Skip("set OASF_SCHEMA_URL to validate the static enricher taxonomy against OASF")
	}

	require.NotEmpty(t, static.Skills)
	require.NotEmpty(t, static.Domains)

	skills := make([]any, 0, len(static.Skills))
	for _, s := range static.Skills {
		skills = append(skills, map[string]any{"name": s.GetName(), "id": float64(s.GetId())})
	}

	domains := make([]any, 0, len(static.Domains))
	for _, d := range static.Domains {
		domains = append(domains, map[string]any{"name": d.GetName(), "id": float64(d.GetId())})
	}

	record, err := structpb.NewStruct(map[string]any{
		"name":           "example.com/static-enricher-test",
		"version":        "1.0.0",
		"schema_version": transformer.DefaultSchemaVersion,
		"description":    "Record used to validate the static enricher taxonomy.",
		"authors":        []any{"AGNTCY"},
		"created_at":     "2026-01-01T00:00:00Z",
		"locators":       []any{map[string]any{"type": "url", "urls": []any{"https://example.com"}}},
		"skills":         skills,
		"domains":        domains,
	})
	require.NoError(t, err)

	v, err := validator.New(schemaURL)
	require.NoError(t, err)

	valid, errs, _, err := v.ValidateRecord(t.Context(), record)
	require.NoError(t, err)
	assert.True(t, valid, "static enricher taxonomy is not valid OASF: %v", errs)
}
