// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package search

import (
	"context"
	"errors"
	"fmt"
	"strings"

	corev1 "github.com/agntcy/dir/api/core/v1"
	searchv1 "github.com/agntcy/dir/api/search/v1"
	"github.com/agntcy/dir/cli/presenter"
	"github.com/agntcy/dir/client"
	clientconfig "github.com/agntcy/dir/client/config"
	"github.com/agntcy/dir/utils/extractor"
	"github.com/agntcy/dir/utils/nlsearch"
	"github.com/spf13/cobra"
)

// clientSearcher adapts the dir client to nlsearch.Searcher so the CLI and the
// API server score the same way from the same code.
type clientSearcher struct {
	cmd *cobra.Command
	c   *client.Client
}

func (s clientSearcher) SearchCIDs(ctx context.Context, query *searchv1.RecordQuery, limit uint32) ([]string, error) {
	result, err := s.c.SearchCIDs(ctx, &searchv1.SearchCIDsRequest{
		Queries:  []*searchv1.RecordQuery{query},
		SortMode: searchv1.SortMode_SORT_MODE_RECENCY,
		Limit:    &limit,
	})
	if err != nil {
		return nil, fmt.Errorf("search CIDs: %w", err)
	}

	var cids []string

	for {
		select {
		case resp := <-result.ResCh():
			if cid := resp.GetRecordCid(); cid != "" {
				cids = append(cids, cid)
			}
		case err := <-result.ErrCh():
			return nil, fmt.Errorf("receive CID: %w", err)
		case <-result.DoneCh():
			return cids, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// runNLSearch decomposes a free-text query into signals using the OASF
// extractor, fans out one SearchCIDs request per signal, then scores and
// returns results ranked by the number of signals that matched each record.
func runNLSearch(cmd *cobra.Command, query string, c *client.Client) error {
	ext, err := clientconfig.ResolveConfigured()
	if err != nil {
		return fmt.Errorf("natural-language search requires the OASF extractor — run `dirctl init` to set it up: %w", err)
	}

	defer func() { _ = ext.Close() }()

	// Only the included schema versions restrict the extractor; --exclude-schema-version
	// filters results and has no taxonomy to point the extractor at.
	signals, err := nlsearch.Decompose(cmd.Context(), query, ext, extractor.ExtractOptions{Versions: opts.Filters.SchemaVersions.Include})
	if err != nil {
		return fmt.Errorf("decompose query: %w", err)
	}

	if len(signals) == 0 {
		return errors.New("no search signals extracted from query; try a more descriptive phrase")
	}

	if opts.Verbose {
		cmd.PrintErrf("[nl-search] signals extracted (%d):\n", len(signals))

		for _, s := range signals {
			cmd.PrintErrf("  %-8s  %-52s  score=%.2f\n", s.Type, s.Value, s.Score)
		}
	}

	scored := nlsearch.FanOutAndScore(cmd.Context(), signals, clientSearcher{cmd: cmd, c: c}, nlsearch.FanOutOptions{})

	for _, sh := range scored.PerSignal {
		if sh.Err != nil {
			cmd.PrintErrf("warning: signal query failed (%s %q): %v\n", sh.Signal.Type, sh.Signal.Value, sh.Err)
		}
	}

	if opts.Verbose {
		cmd.PrintErrf("[nl-search] per-signal hits:\n")

		for _, sh := range scored.PerSignal {
			if sh.Err != nil {
				cmd.PrintErrf("  %-8s  %-52s  ERROR: %v\n", sh.Signal.Type, sh.Signal.Value, sh.Err)
			} else {
				cmd.PrintErrf("  %-8s  %-52s  → %d CIDs\n", sh.Signal.Type, sh.Signal.Value, sh.Count)
			}
		}

		cmd.PrintErrf("[nl-search] ranked results (%d unique, %d signals):\n", len(scored.CIDs), len(signals))

		for _, cid := range scored.CIDs {
			cmd.PrintErrf("  %s  hits=%d/%d  signals=[%s]\n",
				cid, scored.HitCount[cid], len(signals),
				strings.Join(scored.CIDSignals[cid], ", "))
		}
	}

	start := min(int(opts.Offset), len(scored.CIDs))

	end := len(scored.CIDs)
	if opts.Limit > 0 && int(opts.Limit) < end-start {
		end = start + int(opts.Limit)
	}

	return outputNLResults(cmd, c, scored.CIDs[start:end])
}

// outputNLResults formats and prints a ranked CID slice according to --format.
func outputNLResults(cmd *cobra.Command, c *client.Client, paged []string) error {
	switch opts.Format {
	case "cid":
		results := make([]any, len(paged))
		for i, cid := range paged {
			results[i] = cid
		}

		return presenter.PrintMessage(cmd, "record CIDs", "Record CIDs found", results)

	case "record":
		refs := make([]*corev1.RecordRef, len(paged))
		for i, cid := range paged {
			refs[i] = &corev1.RecordRef{Cid: cid}
		}

		records, err := c.PullBatch(cmd.Context(), refs)
		if err != nil {
			return fmt.Errorf("fetch records: %w", err)
		}

		results := make([]any, len(records))
		for i, r := range records {
			results[i] = r
		}

		return presenter.PrintMessage(cmd, "records", "Records found", results)

	default:
		return fmt.Errorf("invalid format: %s (valid values: cid, record)", opts.Format)
	}
}
