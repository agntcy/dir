// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package search

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	corev1 "github.com/agntcy/dir/api/core/v1"
	searchv1 "github.com/agntcy/dir/api/search/v1"
	"github.com/agntcy/dir/cli/internal/extractor"
	"github.com/agntcy/dir/cli/internal/nlsearch"
	"github.com/agntcy/dir/cli/presenter"
	"github.com/agntcy/dir/client"
	sdk "github.com/agntcy/oasf-sdk/pkg/extractor"
	"github.com/spf13/cobra"
)

// fanOutLimit is the per-signal cap on candidate CIDs fetched during NL
// fan-out. Large enough to give the client-side scorer meaningful signal
// without pulling the entire index per query.
const fanOutLimit uint32 = 500

// scoredResults holds the output of a fan-out search pass with full attribution
// data needed for verbose output and final ranking.
type scoredResults struct {
	ranked   []string            // CIDs sorted by hit count descending
	hitCount map[string]int      // CID → number of signals that matched
	cidSigs  map[string][]string // CID → signal labels that matched (for verbose)
	sigHits  []sigHit            // per-signal result summary (for verbose)
}

type sigHit struct {
	signal nlsearch.Signal
	count  int
	err    error
}

// runNLSearch decomposes a free-text query into signals using the OASF
// extractor, fans out one SearchCIDs request per signal, then scores and
// returns results ranked by the number of signals that matched each record.
func runNLSearch(cmd *cobra.Command, query string, c *client.Client) error {
	ext, err := extractor.LoadConfigured()
	if err != nil {
		return fmt.Errorf("natural-language search requires the OASF extractor — run `dirctl init` to set it up: %w", err)
	}

	var queryOpts []sdk.QueryOption
	if len(opts.Filters.SchemaVersions) > 0 {
		queryOpts = append(queryOpts, sdk.Versions(opts.Filters.SchemaVersions...))
	}

	signals, err := nlsearch.Decompose(cmd.Context(), query, ext, queryOpts...)
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

	scored := fanOutAndScore(cmd, c, signals)

	if opts.Verbose {
		cmd.PrintErrf("[nl-search] per-signal hits:\n")

		for _, sh := range scored.sigHits {
			if sh.err != nil {
				cmd.PrintErrf("  %-8s  %-52s  ERROR: %v\n", sh.signal.Type, sh.signal.Value, sh.err)
			} else {
				cmd.PrintErrf("  %-8s  %-52s  → %d CIDs\n", sh.signal.Type, sh.signal.Value, sh.count)
			}
		}

		cmd.PrintErrf("[nl-search] ranked results (%d unique, %d signals):\n", len(scored.ranked), len(signals))

		for _, cid := range scored.ranked {
			cmd.PrintErrf("  %s  hits=%d/%d  signals=[%s]\n",
				cid, scored.hitCount[cid], len(signals),
				strings.Join(scored.cidSigs[cid], ", "))
		}
	}

	start := min(int(opts.Offset), len(scored.ranked))

	end := len(scored.ranked)
	if opts.Limit > 0 && int(opts.Limit) < end-start {
		end = start + int(opts.Limit)
	}

	return outputNLResults(cmd, c, scored.ranked[start:end])
}

// fanOutAndScore launches one SearchCIDs goroutine per signal, collects the
// results, scores each CID by how many signals returned it, and returns a
// scoredResults with full attribution data.
func fanOutAndScore(cmd *cobra.Command, c *client.Client, signals []nlsearch.Signal) scoredResults {
	type fanResult struct {
		signal nlsearch.Signal
		cids   []string
		err    error
	}

	resultCh := make(chan fanResult, len(signals))

	for _, sig := range signals {
		go func() {
			cids, err := collectCIDs(cmd, c, sig)
			resultCh <- fanResult{signal: sig, cids: cids, err: err}
		}()
	}

	hitCount := make(map[string]int)
	cidSigs := make(map[string][]string)

	var ranked []string

	sigHits := make([]sigHit, 0, len(signals))

	for range signals {
		r := <-resultCh

		sh := sigHit{signal: r.signal, count: len(r.cids), err: r.err}
		sigHits = append(sigHits, sh)

		if r.err != nil {
			cmd.PrintErrf("warning: signal query failed (%s %q): %v\n", r.signal.Type, r.signal.Value, r.err)

			continue
		}

		label := fmt.Sprintf("%s:%s", r.signal.Type, r.signal.Value)

		for _, cid := range r.cids {
			if hitCount[cid] == 0 {
				ranked = append(ranked, cid)
			}

			hitCount[cid]++
			cidSigs[cid] = append(cidSigs[cid], label)
		}
	}

	sort.SliceStable(ranked, func(i, j int) bool {
		return hitCount[ranked[i]] > hitCount[ranked[j]]
	})

	return scoredResults{ranked: ranked, hitCount: hitCount, cidSigs: cidSigs, sigHits: sigHits}
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

// collectCIDs issues SearchCIDs request(s) for one NL signal and returns all
// matching CIDs up to fanOutLimit. Keyword signals fan out to both NAME and
// DESCRIPTION queries so that one keyword = one hit regardless of field matched.
func collectCIDs(cmd *cobra.Command, c *client.Client, sig nlsearch.Signal) ([]string, error) {
	if sig.Type == nlsearch.SignalTypeKeyword {
		return collectKeywordCIDs(cmd, c, sig.Value)
	}

	return fetchCIDs(cmd, c, sig.QueryType(), sig.Value)
}

// collectKeywordCIDs fans out a keyword signal to NAME and DESCRIPTION queries
// concurrently and returns the union of CIDs (deduplicated). This ensures one
// keyword produces at most one hit per record in the scorer.
func collectKeywordCIDs(cmd *cobra.Command, c *client.Client, value string) ([]string, error) {
	type result struct {
		cids []string
		err  error
	}

	fetch := func(qt searchv1.RecordQueryType) <-chan result {
		ch := make(chan result, 1)

		go func() {
			cids, err := fetchCIDs(cmd, c, qt, value)
			ch <- result{cids: cids, err: err}
		}()

		return ch
	}

	nameCh := fetch(searchv1.RecordQueryType_RECORD_QUERY_TYPE_NAME)
	descCh := fetch(searchv1.RecordQueryType_RECORD_QUERY_TYPE_DESCRIPTION)

	nameRes := <-nameCh
	descRes := <-descCh

	if nameRes.err != nil {
		return nil, nameRes.err
	}

	if descRes.err != nil {
		return nil, descRes.err
	}

	seen := make(map[string]struct{}, len(nameRes.cids)+len(descRes.cids))

	var union []string

	for _, cid := range append(nameRes.cids, descRes.cids...) {
		if _, ok := seen[cid]; !ok {
			seen[cid] = struct{}{}
			union = append(union, cid)
		}
	}

	return union, nil
}

// fetchCIDs issues a single SearchCIDs request and returns matching CIDs up to fanOutLimit.
func fetchCIDs(cmd *cobra.Command, c *client.Client, qt searchv1.RecordQueryType, value string) ([]string, error) {
	limit := fanOutLimit

	result, err := c.SearchCIDs(cmd.Context(), &searchv1.SearchCIDsRequest{
		Queries: []*searchv1.RecordQuery{
			{Type: qt, Value: value},
		},
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
		case <-cmd.Context().Done():
			return nil, cmd.Context().Err()
		}
	}
}
