// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package export

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/api/exportfmt"
	"github.com/agntcy/dir/cli/cmd/search"
	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	recordutil "github.com/agntcy/dir/cli/util/records"
	"github.com/agntcy/dir/cli/util/reference"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "export <cid-or-name[:version][@digest]>",
	Short: "Export a record from the Directory to a local file or stdout",
	Long: `Export pulls a record from the Directory, transforms it to the requested format,
and writes the result to a file or stdout.

The --format flag selects the output format (required):

  agent-skill    SKILL.md artifact for agentic CLI consumption (Cursor, Claude Code, etc.)
  agent-skill-bundle  Skill bundle archive (.gzip) for multi-file skills
  a2a            A2A AgentCard JSON for Agent-to-Agent protocol interop
  mcp-ghcopilot  GitHub Copilot MCP configuration JSON
  mcp-claudecode Claude Code MCP configuration JSON (.mcp.json "mcpServers" shape)

For raw OASF record JSON, use 'dirctl pull' (which supports --output-file,
--output-dir, and search filters for batch retrieval).

Single-record examples:

  dirctl export bafyreib... --format=a2a --output-file=./agent-card.json
  dirctl export my-agent:1.0 --format=agent-skill --output-file=./SKILL.md
  dirctl export my-agent:1.0 --format=agent-skill-bundle --output-file=./skill.gzip
  dirctl export my-mcp-server --format=mcp-claudecode --output-file=.mcp.json

Batch export from search results:

  dirctl export --output-dir=./exports/ --format=a2a --name "web*"
  dirctl export --output-dir=./exports/ --format=agent-skill --skill "code*"
  dirctl export --output-dir=./exports/ --format=agent-skill-bundle --skill "code*"
  dirctl export --output-dir=./exports/ --format=mcp-ghcopilot --module "integration/mcp"
  dirctl export --output-dir=./claude-configs/ --format=mcp-claudecode --module "integration/mcp"
`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if opts.OutputDir != "" {
			if len(args) > 0 {
				return errors.New("positional argument and --output-dir are mutually exclusive")
			}

			return runBatchExport(cmd)
		}

		if len(args) == 0 {
			return errors.New("either a CID/name argument or --output-dir is required")
		}

		return runExport(cmd, args[0])
	},
}

// validateExportFormat rejects formats that `dirctl export` no longer serves.
// Raw OASF records are exported via `dirctl pull`; every other format is derived
// from the record's modules and handled by the format registry.
func validateExportFormat(format string) error {
	switch format {
	case "":
		return errors.New("--format is required (agent-skill, a2a, mcp-ghcopilot, or mcp-claudecode); for raw OASF records use `dirctl pull`")
	case exportfmt.FormatOASF:
		return errors.New("raw OASF export has moved to `dirctl pull` (it supports --output-file, --output-dir, and search filters); `dirctl export` no longer supports --format=oasf")
	default:
		return nil
	}
}

func runExport(cmd *cobra.Command, input string) error {
	if err := validateExportFormat(opts.Format); err != nil {
		return err
	}

	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	formatter, err := exportfmt.GetFormatter(opts.Format)
	if err != nil {
		return err
	}

	recordCID, err := reference.ResolveToCID(cmd.Context(), c, input)
	if err != nil {
		return err
	}

	record, err := c.Pull(cmd.Context(), &corev1.RecordRef{
		Cid: recordCID,
	})
	if err != nil {
		return fmt.Errorf("failed to pull record: %w", err)
	}

	output, err := formatter.Format(record)
	if err != nil {
		return fmt.Errorf("failed to format record as %s: %w", opts.Format, err)
	}

	if opts.OutputFile != "" {
		outPath := opts.OutputFile
		if filepath.Ext(outPath) == "" {
			outPath += formatter.FileExtension()
		}

		return writeFile(cmd, outPath, output)
	}

	presenter.Print(cmd, string(output))

	return nil
}

func runBatchExport(cmd *cobra.Command) error {
	if err := validateExportFormat(opts.Format); err != nil {
		return err
	}

	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	formatter, err := exportfmt.GetFormatter(opts.Format)
	if err != nil {
		return err
	}

	queries := search.BuildQueries(&opts.Filters)
	if len(queries) == 0 {
		return errors.New("at least one search filter is required for batch export (e.g. --name, --module)")
	}

	if err := os.MkdirAll(opts.OutputDir, 0o755); err != nil { //nolint:mnd
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	records, err := recordutil.SearchAndPull(cmd.Context(), c, queries, opts.Limit)
	if err != nil {
		return err
	}

	if len(records) == 0 {
		presenter.PrintSmartf(cmd, "No records matched the search criteria\n")

		return nil
	}

	var exported int

	if bf := getBatchFormatter(opts.Format); bf != nil {
		exported, err = bf.FormatBatch(records, opts.OutputDir, opts.AllVersions)
	} else {
		exported, err = defaultBatchExport(formatter, records, opts.OutputDir, opts.AllVersions)
	}

	if err != nil {
		return fmt.Errorf("batch export failed: %w", err)
	}

	presenter.PrintSmartf(cmd, "Exported %d record(s) to %s\n", exported, opts.OutputDir)

	return nil
}

func writeFile(cmd *cobra.Command, path string, data []byte) error {
	if err := os.WriteFile(path, data, 0o600); err != nil { //nolint:mnd
		return fmt.Errorf("failed to write file %s: %w", path, err)
	}

	presenter.PrintSmartf(cmd, "Exported to: %s\n", path)

	return nil
}
