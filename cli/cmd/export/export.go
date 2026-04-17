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
	searchv1 "github.com/agntcy/dir/api/search/v1"
	"github.com/agntcy/dir/cli/cmd/export/format"
	"github.com/agntcy/dir/cli/cmd/search"
	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/agntcy/dir/cli/util/reference"
	"github.com/agntcy/dir/client"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "export <cid-or-name[:version][@digest]>",
	Short: "Export a record from the Directory to a local file or stdout",
	Long: `Export pulls a record from the Directory, transforms it to the requested format,
and writes the result to a file or stdout.

The --format flag selects the output format:

  oasf           Raw OASF record JSON (default)
  agent-skill    SKILL.md artifact for agentic CLI consumption (Cursor, Claude Code, etc.)
  a2a            A2A AgentCard JSON for Agent-to-Agent protocol interop
  mcp-ghcopilot  GitHub Copilot MCP configuration JSON

Single-record examples:

  dirctl export bafyreib... --format=a2a --output-file=./agent-card.json
  dirctl export my-agent:1.0 --format=agent-skill --output-file=./SKILL.md

Batch export from search results:

  dirctl export --output-dir=./exports/ --format=a2a --name "web*"
  dirctl export --output-dir=./exports/ --format=agent-skill --skill "code*"
  dirctl export --output-dir=./exports/ --format=mcp-ghcopilot --module "integration/mcp"
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

func runExport(cmd *cobra.Command, input string) error {
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	formatter, err := format.GetFormatter(opts.Format)
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
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	formatter, err := format.GetFormatter(opts.Format)
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

	records, err := searchAndPull(cmd, c, queries)
	if err != nil {
		return err
	}

	if len(records) == 0 {
		presenter.PrintSmartf(cmd, "No records matched the search criteria\n")

		return nil
	}

	var exported int

	if bf, ok := formatter.(format.BatchFormatter); ok {
		exported, err = bf.FormatBatch(records, opts.OutputDir, opts.AllVersions)
	} else {
		exported, err = format.DefaultBatchExport(formatter, records, opts.OutputDir, opts.AllVersions)
	}

	if err != nil {
		return fmt.Errorf("batch export failed: %w", err)
	}

	presenter.PrintSmartf(cmd, "Exported %d record(s) to %s\n", exported, opts.OutputDir)

	return nil
}

func searchAndPull(cmd *cobra.Command, c *client.Client, queries []*searchv1.RecordQuery) ([]*corev1.Record, error) {
	cids, err := collectCIDs(cmd, c, queries)
	if err != nil {
		return nil, err
	}

	records := make([]*corev1.Record, 0, len(cids))

	for _, cid := range cids {
		record, err := c.Pull(cmd.Context(), &corev1.RecordRef{Cid: cid})
		if err != nil {
			return nil, fmt.Errorf("failed to pull record %s: %w", cid, err)
		}

		records = append(records, record)
	}

	return records, nil
}

func collectCIDs(cmd *cobra.Command, c *client.Client, queries []*searchv1.RecordQuery) ([]string, error) {
	result, err := c.SearchCIDs(cmd.Context(), &searchv1.SearchCIDsRequest{
		Limit:   &opts.Limit,
		Queries: queries,
	})
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	var cids []string

	for {
		select {
		case resp := <-result.ResCh():
			if cid := resp.GetRecordCid(); cid != "" {
				cids = append(cids, cid)
			}
		case err := <-result.ErrCh():
			return nil, fmt.Errorf("error during search: %w", err)
		case <-result.DoneCh():
			return cids, nil
		case <-cmd.Context().Done():
			return nil, cmd.Context().Err()
		}
	}
}

func writeFile(cmd *cobra.Command, path string, data []byte) error {
	if err := os.WriteFile(path, data, 0o600); err != nil { //nolint:mnd
		return fmt.Errorf("failed to write file %s: %w", path, err)
	}

	presenter.PrintSmartf(cmd, "Exported to: %s\n", path)

	return nil
}
