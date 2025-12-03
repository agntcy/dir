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

var cidsCmd = &cobra.Command{
	Use:   "cids",
	Short: "Search for record CIDs",
	Long: `Search for records and return only their CIDs.

This is efficient for piping to other commands like pull or delete.

Usage examples:

1. Basic search with filters:
   dirctl search cids --name "my-agent-name" --version "v1.0.0"

2. Wildcard search:
   dirctl search cids --name "web*" --skill "python*"

3. Pipe to pull command:
   dirctl search cids --name "web*" --output raw | xargs -I {} dirctl pull {}

4. Output formats:
   dirctl search cids --name "web*" --output json
   dirctl search cids --name "web*" --output jsonl
   dirctl search cids --name "web*" --output raw
`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runCidsCommand(cmd)
	},
}

func init() {
	registerFlags(cidsCmd)
}

func runCidsCommand(cmd *cobra.Command) error {
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	// Build queries from direct field flags
	queries := buildQueriesFromFlags()

	result, err := c.SearchCIDs(cmd.Context(), &searchv1.SearchCIDsRequest{
		Limit:   &opts.Limit,
		Offset:  &opts.Offset,
		Queries: queries,
	})
	if err != nil {
		return fmt.Errorf("failed to search CIDs: %w", err)
	}

	// Collect results and convert to interface{} slice
	results := make([]interface{}, 0, opts.Limit)

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
