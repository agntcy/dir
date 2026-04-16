// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package importcmd

import (
	signcmd "github.com/agntcy/dir/cli/cmd/sign"
	"github.com/agntcy/dir/importer/config"
	enricherconfig "github.com/agntcy/dir/importer/enricher/config"
	scannerconfig "github.com/agntcy/dir/importer/scanner/config"
)

var opts = &options{}

type options struct {
	config.Config
	TypeFlag      string
	Sign          bool
	OutputCIDFile string
}

func init() {
	flags := Command.Flags()

	// File flags
	flags.StringVar(&opts.FilePath, "file-path", "", "Path to source: JSON file for mcp/a2a, or skill directory for agent-skill")

	// Import source
	flags.StringVar(&opts.TypeFlag, "type", "", "Import kind: mcp, mcp-registry, a2a, or agent-skill (local Agent Skills directory with SKILL.md)")
	flags.StringVar(&opts.RegistryURL, "url", "", "Registry base URL (required when --type=mcp-registry)")
	flags.StringToStringVar(&opts.Filters, "filter", nil, "Filters (key=value)")
	flags.IntVar(&opts.Limit, "limit", 0, "Maximum number of records to import (0 = no limit)")

	// Enrichment flags
	flags.StringVar(&opts.Enricher.ConfigFile, "enrich-config", enricherconfig.DefaultConfigFile, "Path to enricher configuration (JSON: model, mcpServers, max-steps)")
	flags.StringVar(&opts.Enricher.SkillsPromptTemplate, "enrich-skills-prompt", "", "Path to custom skills prompt template file")
	flags.StringVar(&opts.Enricher.DomainsPromptTemplate, "enrich-domains-prompt", "", "Path to custom domains prompt template file")
	flags.IntVar(&opts.Enricher.RequestsPerMinute, "enrich-rate-limit", enricherconfig.DefaultRequestsPerMinute, "Maximum LLM API requests per minute (to avoid rate limit errors)")

	// Scanner flags
	flags.BoolVar(&opts.Scanner.Enabled, "scanner-enabled", scannerconfig.DefaultScannerEnabled, "Run all registered security scanners on each record")
	flags.DurationVar(&opts.Scanner.Timeout, "scanner-timeout", scannerconfig.DefaultTimeout, "Timeout per record scan")
	flags.StringVar(&opts.Scanner.CLIPath, "scanner-cli-path", scannerconfig.DefaultCLIPath, "Path to mcp-scanner binary (default: mcp-scanner from PATH)")
	flags.BoolVar(&opts.Scanner.FailOnError, "scanner-fail-on-error", scannerconfig.DefaultFailOnError, "Do not import records that have error-severity scanner findings")
	flags.BoolVar(&opts.Scanner.FailOnWarning, "scanner-fail-on-warning", scannerconfig.DefaultFailOnWarning, "Do not import records that have warning-severity scanner findings")

	// Signing flags
	flags.BoolVar(&opts.Sign, "sign", false, "Sign records after pushing")
	flags.StringVar(&opts.OutputCIDFile, "output-cids", "", "File to write imported CIDs (one per line, for deferred signing)")
	signcmd.AddSigningFlags(flags)

	// Common flags
	flags.BoolVar(&opts.DryRun, "dry-run", false, "Preview without importing")
	flags.BoolVar(&opts.Force, "force", false, "Force push even if record already exists")
	flags.BoolVar(&opts.Debug, "debug", false, "Enable debug output for deduplication and validation failures")

	Command.MarkFlagRequired("type") //nolint:errcheck
}
