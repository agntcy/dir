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
	"github.com/spf13/cobra"
)

var recordsCmd = &cobra.Command{
	Use:   "records",
	Short: "Search for full records",
	Long: `Search for records and return the full record data.

This retrieves complete record information including all metadata, skills, domains, etc.

Usage examples:

1. Basic search with filters:
   dirctl search records --name "my-agent-name" --version "v1.0.0"

2. Wildcard search:
   dirctl search records --name "web*" --skill "python*"

3. Search with comparison operators:
   dirctl search records --version ">=1.0.0" --version "<2.0.0"
   dirctl search records --created-at ">=2024-01-01"

4. Output formats:
   dirctl search records --name "web*" --output json
   dirctl search records --name "web*" --output jsonl
`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runRecordsCommand(cmd)
	},
}

func init() {
	registerFlags(recordsCmd)
}

func runRecordsCommand(cmd *cobra.Command) error {
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	// Build queries from direct field flags
	queries := buildQueriesFromFlags()

	// Search for full records directly
	result, err := c.SearchRecords(cmd.Context(), &searchv1.SearchRecordsRequest{
		Limit:   &opts.Limit,
		Offset:  &opts.Offset,
		Queries: queries,
	})
	if err != nil {
		return fmt.Errorf("failed to search records: %w", err)
	}

	// Collect records
	results := make([]interface{}, 0, opts.Limit)

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
