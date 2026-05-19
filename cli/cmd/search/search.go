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

6. Search for trusted records only (signature verification passed):
   dirctl search --trusted
   dirctl search --name "web*" --trusted

7. Search by annotation key:value pairs:
   dirctl search --annotation 'manager:alice'
   dirctl search --annotation 'team:*'
   dirctl search --annotation 'env:prod' --annotation 'region:us-*'

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

	// Keep the full proto response so renderers can reach rank_score
	// and rank_explanation without re-fetching.
	responses := make([]*searchv1.SearchCIDsResponse, 0, opts.Limit)

	for {
		select {
		case resp := <-result.ResCh():
			if resp != nil && resp.GetRecordCid() != "" {
				responses = append(responses, resp)
			}
		case err := <-result.ErrCh():
			return fmt.Errorf("error receiving CID: %w", err)
		case <-result.DoneCh():
			return printCIDResults(cmd, responses)
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

	responses := make([]*searchv1.SearchRecordsResponse, 0, opts.Limit)

	for {
		select {
		case resp := <-result.ResCh():
			if resp != nil && resp.GetRecord() != nil {
				responses = append(responses, resp)
			}
		case err := <-result.ErrCh():
			return fmt.Errorf("error receiving record: %w", err)
		case <-result.DoneCh():
			return printRecordResults(cmd, responses)
		case <-cmd.Context().Done():
			return cmd.Context().Err()
		}
	}
}

// printCIDResults dispatches on --output: human prints "[score] cid"
// (plus an --explain breakdown), raw prints bare CIDs (xargs-safe),
// json/jsonl serializes the full proto.
func printCIDResults(cmd *cobra.Command, responses []*searchv1.SearchCIDsResponse) error {
	outputOpts := presenter.GetOutputOptions(cmd)
	if outputOpts.IsStructuredOutput() {
		return printStructured(cmd, "record CIDs", responses, outputOpts.Format == presenter.FormatRaw, func(r *searchv1.SearchCIDsResponse) string {
			return r.GetRecordCid()
		})
	}

	if len(responses) == 0 {
		presenter.Println(cmd, "No record CIDs found")

		return nil
	}

	for _, r := range responses {
		presenter.Println(cmd, presenter.FormatRankedLine(r.GetRankScore(), r.GetRecordCid()))

		if opts.Explain {
			presenter.Println(cmd, "       "+presenter.FormatRankExplanation(r.GetRankExplanation()))
		}
	}

	return nil
}

// printRecordResults mirrors printCIDResults but for SearchRecords.
// Human mode prints just "[score] cid"; use --output json for the body.
func printRecordResults(cmd *cobra.Command, responses []*searchv1.SearchRecordsResponse) error {
	outputOpts := presenter.GetOutputOptions(cmd)
	if outputOpts.IsStructuredOutput() {
		return printStructured(cmd, "records", responses, outputOpts.Format == presenter.FormatRaw, func(r *searchv1.SearchRecordsResponse) string {
			return r.GetRecord().GetCid()
		})
	}

	if len(responses) == 0 {
		presenter.Println(cmd, "No records found")

		return nil
	}

	for _, r := range responses {
		presenter.Println(cmd, presenter.FormatRankedLine(r.GetRankScore(), r.GetRecord().GetCid()))

		if opts.Explain {
			presenter.Println(cmd, "       "+presenter.FormatRankExplanation(r.GetRankExplanation()))
		}
	}

	return nil
}

// printStructured emits bare CIDs for raw output and forwards everything
// else to PrintMessage so json/jsonl serialize the full proto.
func printStructured[T any](cmd *cobra.Command, title string, responses []T, isRaw bool, cidOf func(T) string) error {
	if isRaw {
		for _, r := range responses {
			presenter.Println(cmd, cidOf(r))
		}

		return nil
	}

	items := make([]any, 0, len(responses))
	for _, r := range responses {
		items = append(items, r)
	}

	return presenter.PrintMessage(cmd, title, title+" found", items)
}
