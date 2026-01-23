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
	Use:   "search",
	Short: "Search for records in the directory",
	Long: `Search for records in the directory using various filters and options.

The --format flag controls what is returned:
- cid: Return only record CIDs (default, efficient for piping)
- record: Return full record data

Examples:

1. Search for CIDs only (default, efficient for piping):
   dirctl search --name "web*" | xargs -I {} dirctl pull {}

2. Search and get full records:
   dirctl search --name "web*" --format record --output json

3. Wildcard search examples:
   dirctl search --name "web*"
   dirctl search --version "v1.*"
   dirctl search --skill "python*" --skill "*script"
   dirctl search --domain "*education*"

4. Comparison operators (for version, created-at, schema-version):
   dirctl search --version ">=1.0.0" --version "<2.0.0"
   dirctl search --created-at ">=2024-01-01"

5. Search for verified records only:
   dirctl search --verified
   dirctl search --name "cisco.com/*" --verified

Supported wildcards:
  * - matches zero or more characters
  ? - matches exactly one character
`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runSearchCommand(cmd)
	},
}

func init() {
	registerFlags(Command)
	presenter.AddOutputFlags(Command)
}

func runSearchCommand(cmd *cobra.Command) error {
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	// Build queries from direct field flags
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
		Limit:   &opts.Limit,
		Offset:  &opts.Offset,
		Queries: queries,
	})
	if err != nil {
		return fmt.Errorf("failed to search CIDs: %w", err)
	}

	// Collect results and convert to any slice
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
		Limit:   &opts.Limit,
		Offset:  &opts.Offset,
		Queries: queries,
	})
	if err != nil {
		return fmt.Errorf("failed to search records: %w", err)
	}

	// Collect records
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
