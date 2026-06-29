// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package importcmd

import (
	"fmt"
	"maps"
	"os"
	"time"

	typesv1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1"
	"github.com/agntcy/dir-importer/config"
	enricherconfig "github.com/agntcy/dir-importer/enricher/config"
	"github.com/agntcy/dir-importer/enricher/toolhost"
	scannerconfig "github.com/agntcy/dir-importer/scanner/config"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

// importFileConfig mirrors the YAML config file with explicit yaml tags.
// Fields intentionally absent: sign.* (CLI-only) and output_cids (CLI-only).
type importFileConfig struct {
	Type      string             `yaml:"type"`
	URL       string             `yaml:"url"`
	FilePath  string             `yaml:"file_path"`
	Filters   map[string]string  `yaml:"filters"`
	Limit     int                `yaml:"limit"`
	DryRun    bool               `yaml:"dry_run"`
	OutputDir string             `yaml:"output_dir"`
	Force     bool               `yaml:"force"`
	Debug     bool               `yaml:"debug"`
	Authors   []string           `yaml:"authors"`
	Enricher  enricherFileConfig `yaml:"enricher"`
	Scanner   scannerFileConfig  `yaml:"scanner"`
}

type enricherFileConfig struct {
	ToolHost              *toolHostFileConfig `yaml:"tool_host"`
	SkillsPromptTemplate  string              `yaml:"skills_prompt_template"`
	DomainsPromptTemplate string              `yaml:"domains_prompt_template"`
	RequestsPerMinute     int                 `yaml:"requests_per_minute"`
	SkipEnricher          bool                `yaml:"skip_enricher"`
	Skills                []taxonomyEntry     `yaml:"skills"`
	Domains               []taxonomyEntry     `yaml:"domains"`
}

type toolHostFileConfig struct {
	Model      string                         `yaml:"model"`
	MaxSteps   int                            `yaml:"max_steps"`
	MCPServers map[string]mcpServerFileConfig `yaml:"mcp_servers"`
}

type mcpServerFileConfig struct {
	Command string         `yaml:"command"`
	Args    []string       `yaml:"args"`
	Env     map[string]any `yaml:"env"`
}

type scannerFileConfig struct {
	Enabled       bool         `yaml:"enabled"`
	Timeout       yamlDuration `yaml:"timeout"`
	CLIPath       string       `yaml:"cli_path"`
	FailOnError   bool         `yaml:"fail_on_error"`
	FailOnWarning bool         `yaml:"fail_on_warning"`
}

type taxonomyEntry struct {
	Name string `yaml:"name"`
	ID   uint32 `yaml:"id"`
}

// yamlDuration allows duration strings ("5m", "30s") in YAML config files.
type yamlDuration time.Duration

func (d *yamlDuration) UnmarshalYAML(value *yaml.Node) error {
	v, err := time.ParseDuration(value.Value)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", value.Value, err)
	}

	*d = yamlDuration(v)

	return nil
}

// loadConfig populates the importer config (o.Config) from the optional --config
// YAML file and the command-line flags, with precedence: config file < flags.
//
//nolint:nestif
func (o *options) loadConfig(flags *pflag.FlagSet) error {
	o.Scanner = defaultScannerConfig()
	o.Enricher.RequestsPerMinute = enricherconfig.DefaultRequestsPerMinute

	fileHasToolHost := false

	if o.ConfigFile != "" {
		fc, err := parseConfigFile(o.ConfigFile)
		if err != nil {
			return err
		}

		fileHasToolHost = fc.Enricher.ToolHost != nil

		// The flags share storage with the config.Config fields, so applying the
		// file config below overwrites any flag value. Snapshot the explicitly-set
		// flags first, then restore them afterwards so flags stay above the file.
		scalarFlags := map[string]string{}

		flags.Visit(func(f *pflag.Flag) {
			if f.Name == "filter" {
				return
			}

			scalarFlags[f.Name] = f.Value.String()
		})

		var flagFilters map[string]string
		if flags.Changed("filter") {
			flagFilters = maps.Clone(o.Filters)
		}

		applyFileConfig(fc, &o.Config)

		for name, value := range scalarFlags {
			_ = flags.Set(name, value)
		}

		if flagFilters != nil {
			o.Filters = flagFilters
		}
	}

	// --type binds to a plain string (TypeFlag); map it onto the typed field,
	// letting an explicit flag override any type set by the file.
	if o.TypeFlag != "" {
		o.Type = config.ImportType(o.TypeFlag)
	}

	// Enricher tool host: an inline enricher.tool_host in the file wins;
	// otherwise fall back to the built-in default. Not needed when enrichment is
	// skipped.
	if !o.Enricher.SkipEnricher && !fileHasToolHost {
		o.Enricher.ToolHost = defaultToolHostConfig()
	}

	return nil
}

// parseConfigFile reads and decodes the YAML config file into importFileConfig.
// Non-zero defaults are pre-seeded so unmentioned fields retain their defaults.
func parseConfigFile(path string) (importFileConfig, error) {
	fc := importFileConfig{
		Scanner: scannerFileConfig{
			Timeout: yamlDuration(scannerconfig.DefaultTimeout),
			CLIPath: scannerconfig.DefaultCLIPath,
		},
		Enricher: enricherFileConfig{
			RequestsPerMinute: enricherconfig.DefaultRequestsPerMinute,
		},
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fc, fmt.Errorf("failed to read import config file %q: %w", path, err)
	}

	if err := yaml.Unmarshal(data, &fc); err != nil {
		return fc, fmt.Errorf("failed to parse import config file %q: %w", path, err)
	}

	return fc, nil
}

// applyFileConfig copies the decoded file config onto cfg.
func applyFileConfig(fc importFileConfig, cfg *config.Config) {
	if fc.Type != "" {
		cfg.Type = config.ImportType(fc.Type)
	}

	if fc.URL != "" {
		cfg.RegistryURL = fc.URL
	}

	if fc.FilePath != "" {
		cfg.FilePath = fc.FilePath
	}

	if fc.Filters != nil {
		cfg.Filters = fc.Filters
	}

	if fc.Limit != 0 {
		cfg.Limit = fc.Limit
	}

	cfg.DryRun = fc.DryRun

	if fc.OutputDir != "" {
		cfg.OutputDir = fc.OutputDir
	}

	cfg.Force = fc.Force
	cfg.Debug = fc.Debug

	if len(fc.Authors) > 0 {
		cfg.Authors = fc.Authors
	}

	cfg.Scanner = scannerconfig.Config{
		Enabled:       fc.Scanner.Enabled,
		Timeout:       time.Duration(fc.Scanner.Timeout),
		CLIPath:       fc.Scanner.CLIPath,
		FailOnError:   fc.Scanner.FailOnError,
		FailOnWarning: fc.Scanner.FailOnWarning,
	}

	cfg.Enricher.RequestsPerMinute = fc.Enricher.RequestsPerMinute

	if fc.Enricher.SkillsPromptTemplate != "" {
		cfg.Enricher.SkillsPromptTemplate = fc.Enricher.SkillsPromptTemplate
	}

	if fc.Enricher.DomainsPromptTemplate != "" {
		cfg.Enricher.DomainsPromptTemplate = fc.Enricher.DomainsPromptTemplate
	}

	cfg.Enricher.SkipEnricher = fc.Enricher.SkipEnricher

	for _, s := range fc.Enricher.Skills {
		cfg.Enricher.Skills = append(cfg.Enricher.Skills,
			typesv1.Skill_builder{Name: s.Name, Id: s.ID}.Build())
	}

	for _, d := range fc.Enricher.Domains {
		cfg.Enricher.Domains = append(cfg.Enricher.Domains,
			typesv1.Domain_builder{Name: d.Name, Id: d.ID}.Build())
	}

	if fc.Enricher.ToolHost != nil {
		cfg.Enricher.ToolHost = toToolHostConfig(*fc.Enricher.ToolHost)
	}
}

func toToolHostConfig(fc toolHostFileConfig) toolhost.Config {
	servers := make(map[string]toolhost.MCPServerConfig, len(fc.MCPServers))

	for name, s := range fc.MCPServers {
		servers[name] = toolhost.MCPServerConfig{
			Command: s.Command,
			Args:    s.Args,
			Env:     s.Env,
		}
	}

	return toolhost.Config{
		Model:      fc.Model,
		MaxSteps:   fc.MaxSteps,
		MCPServers: servers,
	}
}

func defaultScannerConfig() scannerconfig.Config {
	return scannerconfig.Config{
		Enabled:       scannerconfig.DefaultScannerEnabled,
		Timeout:       scannerconfig.DefaultTimeout,
		CLIPath:       scannerconfig.DefaultCLIPath,
		FailOnError:   scannerconfig.DefaultFailOnError,
		FailOnWarning: scannerconfig.DefaultFailOnWarning,
	}
}

// defaultToolHostConfig is the built-in enricher tool host used when neither the
// config file nor enrichment-skip provides one.
func defaultToolHostConfig() toolhost.Config {
	return toolhost.Config{
		Model:    "azure:gpt-4o",
		MaxSteps: 10, //nolint:mnd
		MCPServers: map[string]toolhost.MCPServerConfig{
			"dir-mcp-server": {
				Command: "dirctl",
				Args:    []string{"mcp", "serve"},
				Env: map[string]any{
					"OASF_API_VALIDATION_SCHEMA_URL": "https://schema.oasf.outshift.com",
					"DIRECTORY_CLIENT_AUTH_MODE":     "insecure",
				},
			},
		},
	}
}
