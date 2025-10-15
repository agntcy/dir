// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package importcmd

import (
	"errors"
	"fmt"

	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/agntcy/dir/importer/config"
	"github.com/agntcy/dir/importer/mcp"
	"github.com/agntcy/dir/importer/types"
	"github.com/spf13/cobra"
)

var (
	cfg          config.Config
	registryType string
)

var Command = &cobra.Command{
	Use:   "import",
	Short: "Import records from external registries",
	Long: `Import records from external registries into DIR.

Supported registries:
  - mcp: Model Context Protocol registry

The import command fetches records from the specified registry and pushes
them to DIR.

Examples:
  # Import from MCP registry
  dirctl import --type=mcp --url=https://registry.modelcontextprotocol.io

  # Import with filters
  dirctl import --type=mcp --url=https://registry.modelcontextprotocol.io --filter=updated_since=2025-08-07T13:15:04.280Z

  # Preview without importing
  dirctl import --type=mcp --url=https://registry.modelcontextprotocol.io --dry-run
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runImport(cmd)
	},
}

func init() {
	// Add flags
	Command.Flags().StringVar(&registryType, "type", "", "Registry type (mcp, a2a)")
	Command.Flags().StringVar(&cfg.RegistryURL, "url", "", "Registry base URL")
	Command.Flags().StringToStringVar(&cfg.Filters, "filter", nil, "Filters (key=value)")
	Command.Flags().IntVar(&cfg.Limit, "limit", 0, "Maximum number of records to import (0 = no limit)")
	Command.Flags().BoolVar(&cfg.DryRun, "dry-run", false, "Preview without importing")

	// Mark required flags
	Command.MarkFlagRequired("type") //nolint:errcheck
	Command.MarkFlagRequired("url")  //nolint:errcheck
}

func runImport(cmd *cobra.Command) error {
	// Get the client from the context
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	// Set the registry type from the string flag
	cfg.RegistryType = config.RegistryType(registryType)

	// Set the store client
	cfg.StoreClient = c.StoreClient()

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Create factory and register importers
	factory := types.NewFactory()
	mcp.Register(factory)

	// Create importer instance
	importer, err := factory.Create(cfg)
	if err != nil {
		return fmt.Errorf("failed to create importer: %w", err)
	}

	// Run import with progress reporting
	presenter.Printf(cmd, "Starting import from %s registry at %s...\n", cfg.RegistryType, cfg.RegistryURL)

	if cfg.DryRun {
		presenter.Printf(cmd, "Mode: DRY RUN (preview only)\n")
	}

	presenter.Printf(cmd, "\n")

	result, err := importer.Run(cmd.Context(), cfg)
	if err != nil {
		return fmt.Errorf("import failed: %w", err)
	}

	// Print summary
	printSummary(cmd, result)

	return nil
}

func printSummary(cmd *cobra.Command, result *types.ImportResult) {
	maxErrors := 10

	presenter.Printf(cmd, "\n=== Import Summary ===\n")
	presenter.Printf(cmd, "Total records:   %d\n", result.TotalRecords)
	presenter.Printf(cmd, "Imported:        %d\n", result.ImportedCount)
	presenter.Printf(cmd, "Failed:          %d\n", result.FailedCount)

	if len(result.Errors) > 0 {
		presenter.Printf(cmd, "\n=== Errors ===\n")

		for i, err := range result.Errors {
			if i < maxErrors { // Show only first 10 errors
				presenter.Printf(cmd, "  - %v\n", err)
			}
		}

		if len(result.Errors) > maxErrors {
			presenter.Printf(cmd, "  ... and %d more errors\n", len(result.Errors)-maxErrors)
		}
	}

	if cfg.DryRun {
		presenter.Printf(cmd, "\nNote: This was a dry run. No records were actually imported.\n")
	}
}
