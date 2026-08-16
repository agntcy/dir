// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package nlsearch

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"sync"

	searchv1 "github.com/agntcy/dir/api/search/v1"
)

// DefaultFanOutLimit is the per-signal cap on candidate CIDs fetched during a
// fan-out. Large enough to give the scorer meaningful signal without pulling the
// entire index per query.
const DefaultFanOutLimit uint32 = 500

// Searcher runs a single RecordQuery and returns the matching record CIDs,
// bounded by limit. It abstracts over how the caller reaches the search layer:
// the CLI issues a SearchCIDs RPC, the server queries its database in-process.
// Sharing this interface is what keeps the two paths returning the same results
// for the same query.
type Searcher interface {
	SearchCIDs(ctx context.Context, query *searchv1.RecordQuery, limit uint32) ([]string, error)
}

// FanOutOptions tunes a fan-out pass.
type FanOutOptions struct {
	// PerSignalLimit caps the candidates fetched per signal. 0 uses
	// DefaultFanOutLimit.
	PerSignalLimit uint32
}

// SignalHit reports the outcome of querying one signal. Err is set when that
// signal's query failed; the signal simply contributes no hits in that case, so
// a partial backend failure degrades recall rather than failing the search.
// Callers decide how to surface it (the CLI warns, the server logs).
type SignalHit struct {
	Signal Signal
	Count  int
	Err    error
}

// Ranked is the scored output of a fan-out pass.
type Ranked struct {
	// CIDs are the matching records, most relevant first.
	CIDs []string
	// HitCount maps a CID to the number of signals that matched it.
	HitCount map[string]int
	// CIDSignals maps a CID to the labels of the signals that matched it.
	CIDSignals map[string][]string
	// PerSignal reports each signal's outcome, in the order the signals were given.
	PerSignal []SignalHit
}

// FanOutAndScore queries every signal independently and ranks the union of the
// results by how many signals matched each record.
//
// The union (rather than an intersection) is deliberate: a record that matches
// some of a query's signals is still a plausible answer, and ranking — not
// exclusion — is what separates it from a record that matches all of them.
//
// The ordering is a total order — hit count, then summed signal score, then CID —
// so repeated identical calls return identical rankings. That matters for
// paginated callers, where an unstable tie order would let records repeat or
// vanish between pages.
func FanOutAndScore(ctx context.Context, signals []Signal, searcher Searcher, opts FanOutOptions) Ranked {
	limit := opts.PerSignalLimit
	if limit == 0 {
		limit = DefaultFanOutLimit
	}

	// Collect into an index-addressed slice rather than appending on completion,
	// so the accumulation below runs in signal order regardless of which query
	// finishes first.
	type fanResult struct {
		cids []string
		err  error
	}

	results := make([]fanResult, len(signals))

	var wg sync.WaitGroup

	for i, sig := range signals {
		wg.Go(func() {
			cids, err := collectSignalCIDs(ctx, searcher, sig, limit)
			results[i] = fanResult{cids: cids, err: err}
		})
	}

	wg.Wait()

	ranked := Ranked{
		HitCount:   make(map[string]int),
		CIDSignals: make(map[string][]string),
		PerSignal:  make([]SignalHit, 0, len(signals)),
	}

	scoreSum := make(map[string]float64)

	for i, sig := range signals {
		r := results[i]
		ranked.PerSignal = append(ranked.PerSignal, SignalHit{Signal: sig, Count: len(r.cids), Err: r.err})

		if r.err != nil {
			continue
		}

		label := fmt.Sprintf("%s:%s", sig.Type, sig.Value)

		for _, cid := range r.cids {
			if ranked.HitCount[cid] == 0 {
				ranked.CIDs = append(ranked.CIDs, cid)
			}

			ranked.HitCount[cid]++
			ranked.CIDSignals[cid] = append(ranked.CIDSignals[cid], label)
			scoreSum[cid] += sig.Score
		}
	}

	slices.SortFunc(ranked.CIDs, func(a, b string) int {
		if d := cmp.Compare(ranked.HitCount[b], ranked.HitCount[a]); d != 0 {
			return d
		}

		if d := cmp.Compare(scoreSum[b], scoreSum[a]); d != 0 {
			return d
		}

		return cmp.Compare(a, b)
	})

	return ranked
}

// collectSignalCIDs queries one signal. Keyword signals have no single field to
// match, so they fan out to NAME and DESCRIPTION and are deduplicated — one
// keyword contributes at most one hit per record, keeping it comparable to a
// taxonomy signal when scoring.
func collectSignalCIDs(ctx context.Context, searcher Searcher, sig Signal, limit uint32) ([]string, error) {
	if sig.Type != SignalTypeKeyword {
		cids, err := searcher.SearchCIDs(ctx, &searchv1.RecordQuery{Type: sig.QueryType(), Value: sig.Value}, limit)
		if err != nil {
			return nil, fmt.Errorf("search %s %q: %w", sig.Type, sig.Value, err)
		}

		return cids, nil
	}

	types := []searchv1.RecordQueryType{
		searchv1.RecordQueryType_RECORD_QUERY_TYPE_NAME,
		searchv1.RecordQueryType_RECORD_QUERY_TYPE_DESCRIPTION,
	}

	perType := make([][]string, len(types))
	errs := make([]error, len(types))

	var wg sync.WaitGroup

	for i, qt := range types {
		wg.Go(func() {
			perType[i], errs[i] = searcher.SearchCIDs(ctx, &searchv1.RecordQuery{Type: qt, Value: sig.Value}, limit)
		})
	}

	wg.Wait()

	for _, err := range errs {
		if err != nil {
			return nil, fmt.Errorf("search keyword %q: %w", sig.Value, err)
		}
	}

	seen := make(map[string]struct{})

	var union []string

	for _, cids := range perType {
		for _, cid := range cids {
			if _, ok := seen[cid]; !ok {
				seen[cid] = struct{}{}
				union = append(union, cid)
			}
		}
	}

	return union, nil
}
