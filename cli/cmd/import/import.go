// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package importcmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/agntcy/dir-importer/factory"
	"github.com/agntcy/dir-importer/types"
	signcmd "github.com/agntcy/dir/cli/cmd/sign"
	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "import",
	Short: "Import MCP, A2A, or Agent Skill records into DIR from a registry, JSON file, or skill directory",
	Long: `Import MCP server, A2A AgentCard, or Agent Skill records into DIR. Records are transformed, enriched, optionally
scanned, then pushed. The same pipeline runs for every source.

Import kinds (--type):
  mcp            Local JSON: one MCP server object or a JSON array (--file-path)
  mcp-registry   HTTP MCP registry, e.g. v0.1 list API (--url)
  a2a            Local JSON: one A2A AgentCard or an array of cards (--file-path)
  agent-skill    Local directory: one Agent Skills folder containing SKILL.md (--file-path); see https://agentskills.io/specification

Examples:
  dirctl import --config import.config.yaml
  dirctl import --config import.config.yaml --filter=search=analytics --dry-run --output-dir=./out

  dirctl import --type=mcp-registry --url=https://registry.modelcontextprotocol.io/v0.1
  dirctl import --type=mcp --file-path=./servers.json
  dirctl import --type=a2a --file-path=./agent.json
  dirctl import --type=agent-skill --file-path=./my-skill

  dirctl import --type=mcp-registry --url=https://registry.modelcontextprotocol.io/v0.1 --sign --key=./cosign.key
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runImport(cmd)
	},
}

func runImport(cmd *cobra.Command) error {
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	if err := opts.loadConfig(cmd.Flags()); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	if opts.Sign {
		opts.SignFunc = func(ctx context.Context, cid string) error {
			return signcmd.Sign(ctx, c, cid)
		}
	}

	if err := opts.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	importer, err := factory.Create(cmd.Context(), c, opts.Config)
	if err != nil {
		return fmt.Errorf("failed to create importer: %w", err)
	}

	var result *types.ImportResult

	if opts.DryRun {
		result = importer.DryRun(cmd.Context())
	} else {
		result = importer.Run(cmd.Context())
	}

	printSummary(cmd, result)

	if opts.OutputCIDFile != "" && len(result.ImportedCIDs) > 0 {
		content := strings.Join(result.ImportedCIDs, "\n") + "\n"

		if err := os.WriteFile(opts.OutputCIDFile, []byte(content), 0o600); err != nil { //nolint:mnd
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
			if i < maxErrors {
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

		if result.OutputDir != "" {
			presenter.Printf(cmd, "Records saved to directory: %s\n", result.OutputDir)
			presenter.Printf(cmd, "Each record is written as <cid>.record.json and can be re-imported via dirctl push.\n")
		}
	}
}
