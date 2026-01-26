// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package importcmd

import (
	signcmd "github.com/agntcy/dir/cli/cmd/sign"
	"github.com/agntcy/dir/importer/config"
	"github.com/agntcy/dir/importer/enricher"
)

var opts = &options{}

type options struct {
	config.Config
	RegistryType string
	Sign         bool // Sign records after pushing (flag binding)
}

func init() {
	flags := Command.Flags()

	// Registry flags
	flags.StringVar(&opts.RegistryType, "type", "", "Registry type (mcp, a2a)")
	flags.StringVar(&opts.RegistryURL, "url", "", "Registry base URL")
	flags.StringToStringVar(&opts.Filters, "filter", nil, "Filters (key=value)")
	flags.IntVar(&opts.Limit, "limit", 0, "Maximum number of records to import (0 = no limit)")
	flags.BoolVar(&opts.DryRun, "dry-run", false, "Preview without importing")
	flags.BoolVar(&opts.Force, "force", false, "Force push even if record already exists")
	flags.BoolVar(&opts.Debug, "debug", false, "Enable debug output for deduplication and validation failures")

	// Enrichment is mandatory - these flags configure the enrichment process
	flags.StringVar(&opts.EnricherConfigFile, "enrich-config", enricher.DefaultConfigFile, "Path to MCPHost configuration file (mcphost.json)")
	flags.StringVar(&opts.EnricherSkillsPromptTemplate, "enrich-skills-prompt", "", "Optional: path to custom skills prompt template file or inline prompt (empty = use default)")
	flags.StringVar(&opts.EnricherDomainsPromptTemplate, "enrich-domains-prompt", "", "Optional: path to custom domains prompt template file or inline prompt (empty = use default)")

	// Signing flags
	flags.BoolVar(&opts.Sign, "sign", false, "Sign records after pushing")
	signcmd.AddSigningFlags(flags)

	// Mark required flags
	Command.MarkFlagRequired("type") //nolint:errcheck
	Command.MarkFlagRequired("url")  //nolint:errcheck
}
