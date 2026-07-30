// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package search

import (
	"errors"
	"fmt"

	searchv1 "github.com/agntcy/dir/api/search/v1"
	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/agntcy/dir/client"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "search [query]",
	Short: "Search for records in the directory",
	Long: `Search for records in the directory.

Provide a free-text query as a positional argument to use natural-language
search: the OASF extractor (set up by 'dirctl init') decomposes the phrase
into skill, domain, and keyword signals. Each signal is queried independently
and results are ranked by how many signals matched.

Omit the positional argument to use structured search with explicit flags
(--name, --skill, --domain, etc.).

Examples:

1. Natural-language search (requires 'dirctl init'):
   dirctl search "Github MCP server that manages issues"
   dirctl search "real-time fraud detection for banking" --format record

2. Structured search for CIDs (default, efficient for piping):
   dirctl search --name "web*" | xargs -I {} dirctl pull {}

3. Structured search with full records:
   dirctl search --name "web*" --format record --output json

4. Wildcard search examples:
   dirctl search --name "web*"
   dirctl search --version "v1.*"
   dirctl search --skill "python*" --skill "*script"
   dirctl search --domain "*education*"

5. Comparison operators (for version, created-at, schema-version):
   dirctl search --version ">=1.0.0" --version "<2.0.0"
   dirctl search --created-at ">=2024-01-01"

6. Search for verified records only:
   dirctl search --verified
   dirctl search --name "cisco.com/*" --verified

7. Search for trusted records only (signature verification passed):
   dirctl search --trusted
   dirctl search --name "web*" --trusted

8. Search for security-scanned records where all scanners reported safe:
   dirctl search --safe
   dirctl search --name "web*" --safe

9. Search for records whose highest scan severity meets or exceeds a threshold:
   dirctl search --scan-severity HIGH
   dirctl search --safe --scan-severity MEDIUM

10. Search by annotation key:value pairs:
    dirctl search --annotation 'manager:alice'
    dirctl search --annotation 'team:*'
    dirctl search --annotation 'env:prod' --annotation 'region:us-*'

Supported wildcards:
  * - matches zero or more characters
  ? - matches exactly one character
`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, ok := ctxUtils.GetClientFromContext(cmd.Context())
		if !ok {
			return errors.New("failed to get client from context")
		}

		if len(args) == 1 {
			return runNLSearch(cmd, args[0], c)
		}

		return runStructuredSearch(cmd, c)
	},
}

func init() {
	registerFlags(Command)
	presenter.AddOutputFlags(Command)
}

// runStructuredSearch handles the flag-driven search path (unchanged behaviour).
func runStructuredSearch(cmd *cobra.Command, c *client.Client) error {
	queries := buildQueriesFromFlags()

	switch opts.Format {
	case "cid":
		return searchCIDs(cmd, c, queries)
	case "record":
		return searchRecords(cmd, c, queries)
	default:
		return fmt.Errorf("invalid format: %s (valid values: cid, record)", opts.Format)
	}
}

func searchCIDs(cmd *cobra.Command, c *client.Client, queries []*searchv1.RecordQuery) error {
	result, err := c.SearchCIDs(cmd.Context(), &searchv1.SearchCIDsRequest{
		Limit:    &opts.Limit,
		Offset:   &opts.Offset,
		Queries:  queries,
		SortMode: sortMode(),
	})
	if err != nil {
		return fmt.Errorf("failed to search CIDs: %w", err)
	}

	results := make([]any, 0, opts.Limit)

	for {
		select {
		case resp := <-result.ResCh():
			cid := resp.GetRecordCid()
			if cid != "" {
				results = append(results, cid)
			}
		case err := <-result.ErrCh():
			return fmt.Errorf("error receiving CID: %w", err)
		case <-result.DoneCh():
			return presenter.PrintMessage(cmd, "record CIDs", "Record CIDs found", results)
		case <-cmd.Context().Done():
			return cmd.Context().Err()
		}
	}
}

func searchRecords(cmd *cobra.Command, c *client.Client, queries []*searchv1.RecordQuery) error {
	result, err := c.SearchRecords(cmd.Context(), &searchv1.SearchRecordsRequest{
		Limit:    &opts.Limit,
		Offset:   &opts.Offset,
		Queries:  queries,
		SortMode: sortMode(),
	})
	if err != nil {
		return fmt.Errorf("failed to search records: %w", err)
	}

	results := make([]any, 0, opts.Limit)

	for {
		select {
		case resp := <-result.ResCh():
			record := resp.GetRecord()
			if record != nil {
				results = append(results, record)
			}
		case err := <-result.ErrCh():
			return fmt.Errorf("error receiving record: %w", err)
		case <-result.DoneCh():
			return presenter.PrintMessage(cmd, "records", "Records found", results)
		case <-cmd.Context().Done():
			return cmd.Context().Err()
		}
	}
}
