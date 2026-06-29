// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package importcmd

import (
	"github.com/agntcy/dir-importer/config"
	signcmd "github.com/agntcy/dir/cli/cmd/sign"
)

var opts = &options{}

type options struct {
	config.Config
	TypeFlag      string
	Sign          bool
	OutputCIDFile string
	ConfigFile    string
}

func init() {
	flags := Command.Flags()

	// Config file
	flags.StringVar(&opts.ConfigFile, "config", "", "Path to a YAML import config file. Values are overridden by command-line flags")

	// File flags
	flags.StringVar(&opts.FilePath, "file-path", "", "Path to source: JSON file for mcp/a2a, or skill directory for agent-skill")

	// Import source
	flags.StringVar(&opts.TypeFlag, "type", "", "Import kind: mcp, mcp-registry, a2a, or agent-skill (local Agent Skills directory with SKILL.md)")
	flags.StringVar(&opts.RegistryURL, "url", "", "Registry base URL (required when --type=mcp-registry)")
	flags.StringToStringVar(&opts.Filters, "filter", nil, "Filters (key=value)")
	flags.IntVar(&opts.Limit, "limit", 0, "Maximum number of records to import (0 = no limit)")

	// Signing flags
	flags.BoolVar(&opts.Sign, "sign", false, "Sign records after pushing")
	flags.StringVar(&opts.OutputCIDFile, "output-cids", "", "File to write imported CIDs (one per line, for deferred signing)")
	signcmd.AddSigningFlags(flags)

	// Common flags
	flags.BoolVar(&opts.DryRun, "dry-run", false, "Preview without importing; transformed records are written to --output-dir (one JSON file per record) so they can be re-imported later via `dirctl import`")
	flags.StringVar(&opts.OutputDir, "output-dir", "", "Directory to write per-record JSON files when --dry-run is set (defaults to ./import-dry-run-<timestamp> in the current working directory)")
	flags.BoolVar(&opts.Force, "force", false, "Force push even if record already exists")
	flags.BoolVar(&opts.Debug, "debug", false, "Enable debug output for deduplication and validation failures")
}
