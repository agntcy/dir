// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package importcmd

import (
	"fmt"
	"maps"
	"os"

	typesv1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1"
	"github.com/agntcy/dir-importer/config"
	enricherconfig "github.com/agntcy/dir-importer/enricher/config"
	"github.com/agntcy/dir-importer/enricher/toolhost"
	internalextractor "github.com/agntcy/dir/cli/internal/extractor"
	sdk "github.com/agntcy/oasf-sdk/pkg/extractor"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

// importFileConfig mirrors the YAML config file with explicit yaml tags.
// Fields intentionally absent: sign.* (CLI-only) and output_cids (CLI-only).
type importFileConfig struct {
	Type          string             `yaml:"type"`
	URL           string             `yaml:"url"`
	FilePath      string             `yaml:"file_path"`
	Filters       map[string]string  `yaml:"filters"`
	Limit         int                `yaml:"limit"`
	DryRun        bool               `yaml:"dry_run"`
	OutputDir     string             `yaml:"output_dir"`
	Force         bool               `yaml:"force"`
	Debug         bool               `yaml:"debug"`
	Authors       []string           `yaml:"authors"`
	SchemaVersion string             `yaml:"schema_version"`
	Enricher      enricherFileConfig `yaml:"enricher"`
}

type enricherFileConfig struct {
	LLM       *llmEnricherFileConfig       `yaml:"llm"`
	Static    *staticEnricherFileConfig    `yaml:"static"`
	Extractor *extractorEnricherFileConfig `yaml:"extractor"`
}

// extractorEnricherFileConfig configures the local OASF extractor enricher.
// Both fields are optional: when absent the extractor uses the config saved by
// `dirctl init`; explicit values override it.
type extractorEnricherFileConfig struct {
	OASFUrl  string `yaml:"oasf_url"`
	AssetDir string `yaml:"asset_dir"`
}

type llmEnricherFileConfig struct {
	ToolHost              *toolHostFileConfig `yaml:"tool_host"`
	SkillsPromptTemplate  string              `yaml:"skills_prompt_template"`
	DomainsPromptTemplate string              `yaml:"domains_prompt_template"`
	RequestsPerMinute     int                 `yaml:"requests_per_minute"`
}

type staticEnricherFileConfig struct {
	Skills  []taxonomyEntry `yaml:"skills"`
	Domains []taxonomyEntry `yaml:"domains"`
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

type taxonomyEntry struct {
	Name string `yaml:"name"`
	ID   uint32 `yaml:"id"`
}

// loadConfig populates the importer config (o.Config) from the optional --config
// YAML file and the command-line flags, with precedence: config file < flags.
//
//nolint:nestif
func (o *options) loadConfig(flags *pflag.FlagSet) error {
	fileHasEnricher := false

	if o.ConfigFile != "" {
		fc, err := parseConfigFile(o.ConfigFile)
		if err != nil {
			return err
		}

		fileHasEnricher = fc.Enricher.LLM != nil || fc.Enricher.Static != nil || fc.Enricher.Extractor != nil

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

		// Extractor enricher needs a live SDK instance; load it after all file
		// config is applied and flags are restored.
		if fc.Enricher.Extractor != nil {
			sdkExt, loadErr := loadExtractorFromConfig(fc.Enricher.Extractor)
			if loadErr != nil {
				return fmt.Errorf("extractor enricher: %w", loadErr)
			}

			// The extractor classifies against its newest provisioned OASF version
			// (oasfExtractorAdapter uses sdk.Latest()), so it assigns classes that
			// only exist in that version. Records are validated against the schema of
			// their own schema_version, so an unset version would fall back to the
			// transformer default (an older release) and the server would reject the
			// newer classes as unknown. Align the stamped version with the extractor
			// unless the config pins one explicitly.
			if o.SchemaVersion == "" {
				o.SchemaVersion = sdkExt.LatestVersion()
			}

			o.Enricher = enricherconfig.Config{
				Extractor: &enricherconfig.ExtractorConfig{
					Extractor: &oasfExtractorAdapter{ext: sdkExt},
				},
			}
		}
	}

	// --type binds to a plain string (TypeFlag); map it onto the typed field,
	// letting an explicit flag override any type set by the file.
	if o.TypeFlag != "" {
		o.Type = config.ImportType(o.TypeFlag)
	}

	// Default to LLM enrichment with the built-in tool host when no enricher
	// method was provided via the config file.
	if !fileHasEnricher {
		o.Enricher = enricherconfig.Config{
			LLM: &enricherconfig.LLMConfig{
				ToolHost:          defaultToolHostConfig(),
				RequestsPerMinute: enricherconfig.DefaultRequestsPerMinute,
			},
		}
	}

	return nil
}

// parseConfigFile reads and decodes the YAML config file into importFileConfig.
// Non-zero defaults are pre-seeded so unmentioned fields retain their defaults.
func parseConfigFile(path string) (importFileConfig, error) {
	var fc importFileConfig

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

	if fc.SchemaVersion != "" {
		cfg.SchemaVersion = fc.SchemaVersion
	}

	if fc.Enricher.LLM != nil {
		llm := fc.Enricher.LLM
		llmCfg := &enricherconfig.LLMConfig{
			RequestsPerMinute:     llm.RequestsPerMinute,
			SkillsPromptTemplate:  llm.SkillsPromptTemplate,
			DomainsPromptTemplate: llm.DomainsPromptTemplate,
		}

		if llmCfg.RequestsPerMinute == 0 {
			llmCfg.RequestsPerMinute = enricherconfig.DefaultRequestsPerMinute
		}

		if llm.ToolHost != nil {
			llmCfg.ToolHost = toToolHostConfig(*llm.ToolHost)
		} else {
			llmCfg.ToolHost = defaultToolHostConfig()
		}

		cfg.Enricher = enricherconfig.Config{LLM: llmCfg}
	} else if fc.Enricher.Static != nil {
		staticCfg := &enricherconfig.StaticConfig{}

		for _, s := range fc.Enricher.Static.Skills {
			staticCfg.Skills = append(staticCfg.Skills,
				typesv1.Skill_builder{Name: s.Name, Id: s.ID}.Build())
		}

		for _, d := range fc.Enricher.Static.Domains {
			staticCfg.Domains = append(staticCfg.Domains,
				typesv1.Domain_builder{Name: d.Name, Id: d.ID}.Build())
		}

		cfg.Enricher = enricherconfig.Config{Static: staticCfg}
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

// loadExtractorFromConfig loads the local OASF extractor for the extractor enricher.
// When both OASFUrl and AssetDir are empty it defers to the config saved by
// `dirctl init`; explicit values override the saved config.
func loadExtractorFromConfig(fc *extractorEnricherFileConfig) (*sdk.Extractor, error) {
	if fc.OASFUrl == "" && fc.AssetDir == "" {
		ext, err := internalextractor.LoadConfigured()
		if err != nil {
			return nil, fmt.Errorf("load configured extractor: %w", err)
		}

		return ext, nil
	}

	ext, err := internalextractor.Load(internalextractor.Config{
		OASFURL:  fc.OASFUrl,
		AssetDir: fc.AssetDir,
	})
	if err != nil {
		return nil, fmt.Errorf("load extractor: %w", err)
	}

	return ext, nil
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
