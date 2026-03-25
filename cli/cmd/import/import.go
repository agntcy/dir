// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package importcmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	signcmd "github.com/agntcy/dir/cli/cmd/sign"
	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/agntcy/dir/importer/config"
	"github.com/agntcy/dir/importer/factory"
	"github.com/agntcy/dir/importer/types"
	"github.com/spf13/cobra"
)

const cidOutputFilePerm = 0o600

var Command = &cobra.Command{
	Use:   "import",
	Short: "Import records from external registries",
	Long: `Import records from external registries into DIR.

Supported registries:
  - mcp: Model Context Protocol registry v0.1

The import command fetches records from the specified registry and pushes
them to DIR.

Examples:
  # Import from MCP registry with default enrichment configuration
  dirctl import --type=mcp --url=https://registry.modelcontextprotocol.io

  # Import with filters
  # Available filters: https://registry.modelcontextprotocol.io/docs#/operations/list-servers-v0.1#Query-Parameters
  dirctl import --type=mcp --url=https://registry.modelcontextprotocol.io --filter=updated_since=2025-08-07T13:15:04.280Z

  # Preview without importing
  dirctl import --type=mcp --url=https://registry.modelcontextprotocol.io --dry-run

  # Use custom MCPHost configuration and prompt templates
  dirctl import --type=mcp --url=https://registry.modelcontextprotocol.io \
    --enrich-skills-prompt=/path/to/custom-skills-prompt.md \
    --enrich-domains-prompt=/path/to/custom-domains-prompt.md

  # Import and sign records with OIDC (opens browser for authentication)
  dirctl import --type=mcp --url=https://registry.modelcontextprotocol.io --sign

  # Import and sign records with a private key
  dirctl import --type=mcp --url=https://registry.modelcontextprotocol.io --sign --key=/path/to/cosign.key
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runImport(cmd)
	},
}

func runImport(cmd *cobra.Command) error {
	// Get the client from the context
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	// Set the registry type from the string flag
	opts.Config.RegistryType = config.RegistryType(opts.RegistryType)

	// Set up signing function if enabled
	if opts.Sign {
		opts.SignFunc = func(ctx context.Context, cid string) error {
			return signcmd.Sign(ctx, c, cid)
		}
	}

	// Validate configuration
	if err := opts.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Create importer instance from pre-initialized factory
	importer, err := factory.Create(cmd.Context(), c, opts.Config)
	if err != nil {
		return fmt.Errorf("failed to create importer: %w", err)
	}

	// Run import with progress reporting
	presenter.Printf(cmd, "Starting import from %s registry at %s...\n", opts.Config.RegistryType, opts.RegistryURL)

	if opts.DryRun {
		presenter.Printf(cmd, "Mode: DRY RUN (preview only)\n")
	}

	if opts.Sign {
		presenter.Printf(cmd, "Signing: ENABLED\n")
	}

	presenter.Printf(cmd, "\n")

	var result *types.ImportResult

	if opts.DryRun {
		result = importer.DryRun(cmd.Context())
	} else {
		result = importer.Run(cmd.Context())
	}

	printSummary(cmd, result)

	// Write CIDs to output file if specified
	if opts.OutputCIDFile != "" && len(result.ImportedCIDs) > 0 {
		content := strings.Join(result.ImportedCIDs, "\n") + "\n"

		if err := os.WriteFile(opts.OutputCIDFile, []byte(content), cidOutputFilePerm); err != nil {
			return fmt.Errorf("failed to write CIDs to file: %w", err)
		}

		presenter.Printf(cmd, "CIDs written to: %s\n", opts.OutputCIDFile)
	}

	return nil
}

func printSummary(cmd *cobra.Command, result *types.ImportResult) {
	maxErrors := 10

	presenter.Printf(cmd, "\n=== Import Summary ===\n")
	presenter.Printf(cmd, "Total records:   %d\n", result.TotalRecords)
	presenter.Printf(cmd, "Imported:        %d\n", result.ImportedCount)
	presenter.Printf(cmd, "Skipped:         %d\n", result.SkippedCount)
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

	if len(result.ScannerFindings) > 0 {
		presenter.Printf(cmd, "\n=== Scanner findings ===\n")

		for i, msg := range result.ScannerFindings {
			if i < maxErrors {
				presenter.Printf(cmd, "  - %s\n", msg)
			}
		}

		if len(result.ScannerFindings) > maxErrors {
			presenter.Printf(cmd, "  ... and %d more\n", len(result.ScannerFindings)-maxErrors)
		}
	}

	if opts.DryRun {
		presenter.Printf(cmd, "\nNote: This was a dry run. No records were actually imported.\n")

		if result.OutputFile != "" {
			presenter.Printf(cmd, "Records saved to: %s\n", result.OutputFile)
		}
	}
}
