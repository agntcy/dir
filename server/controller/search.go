// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"fmt"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	searchv1 "github.com/agntcy/dir/api/search/v1"
	databaseutils "github.com/agntcy/dir/server/database/utils"
	"github.com/agntcy/dir/server/ranking"
	rankingcfg "github.com/agntcy/dir/server/ranking/config"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
)

var searchLogger = logging.Logger("controller/search")

type searchCtlr struct {
	searchv1.UnimplementedSearchServiceServer
	db    types.DatabaseAPI
	store types.StoreAPI
	cfg   rankingcfg.Config
}

func NewSearchController(db types.DatabaseAPI, store types.StoreAPI, cfg rankingcfg.Config) searchv1.SearchServiceServer {
	return &searchCtlr{
		UnimplementedSearchServiceServer: searchv1.UnimplementedSearchServiceServer{},
		db:                               db,
		store:                            store,
		cfg:                              cfg,
	}
}

func (c *searchCtlr) SearchCIDs(req *searchv1.SearchCIDsRequest, srv searchv1.SearchService_SearchCIDsServer) error {
	searchLogger.Debug("Called search controller's SearchCIDs method", "req", req)

	scored, err := c.fetchAndScore(req.GetQueries())
	if err != nil {
		return err
	}

	for _, s := range ranking.Paginate(scored, int(req.GetOffset()), int(req.GetLimit())) {
		if err := srv.Send(&searchv1.SearchCIDsResponse{
			RecordCid:       s.CID,
			RankScore:       s.Result.Score,
			RankExplanation: s.Result.Explanation,
		}); err != nil {
			return fmt.Errorf("failed to send record CID: %w", err)
		}
	}

	return nil
}

func (c *searchCtlr) SearchRecords(req *searchv1.SearchRecordsRequest, srv searchv1.SearchService_SearchRecordsServer) error {
	searchLogger.Debug("Called search controller's SearchRecords method", "req", req)

	scored, err := c.fetchAndScore(req.GetQueries())
	if err != nil {
		return err
	}

	for _, s := range ranking.Paginate(scored, int(req.GetOffset()), int(req.GetLimit())) {
		if err := srv.Context().Err(); err != nil {
			return fmt.Errorf("client disconnected: %w", err)
		}

		record, err := c.store.Pull(srv.Context(), &corev1.RecordRef{Cid: s.CID})
		if err != nil {
			searchLogger.Warn("Failed to pull record from store", "cid", s.CID, "error", err)

			continue
		}

		if err := srv.Send(&searchv1.SearchRecordsResponse{
			Record:          record,
			RankScore:       s.Result.Score,
			RankExplanation: s.Result.Explanation,
		}); err != nil {
			return fmt.Errorf("failed to send record: %w", err)
		}
	}

	return nil
}

// fetchAndScore loads up to cfg.MaxCandidates matching records, scores
// each, and returns them sorted by rank_score DESC (CID tiebreak).
// Pagination is applied by the caller — we need the whole candidate
// set in memory to sort.
//
// Popularity is always 0 here (the local controller does not consult
// the DHT) and query-relevance is always 1.0 (the SQL WHERE clause
// already filtered out non-matches).
func (c *searchCtlr) fetchAndScore(queries []*searchv1.RecordQuery) ([]ranking.Scored, error) {
	filterOptions, err := databaseutils.QueryToFilters(queries)
	if err != nil {
		return nil, fmt.Errorf("failed to create filter options: %w", err)
	}

	// Pull one extra row to detect truncation before silently ranking a
	// partial set. cfg is already normalized, so MaxCandidates > 0.
	filterOptions = append(filterOptions, types.WithLimit(c.cfg.MaxCandidates+1))

	records, err := c.db.GetRecords(filterOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to get records: %w", err)
	}

	if len(records) > c.cfg.MaxCandidates {
		searchLogger.Warn("Candidate set exceeds MaxCandidates; ranking partial result. "+
			"Tighten your query or raise ranking.max_candidates.",
			"max_candidates", c.cfg.MaxCandidates, "matched", len(records))

		records = records[:c.cfg.MaxCandidates]
	}

	now := time.Now().UTC()

	// Dedupe by CID. GetRecords joins against child tables, so a record
	// with two matching skills comes back twice; we must collapse those
	// to one rank-per-record without changing GetRecords semantics.
	scored := make([]ranking.Scored, 0, len(records))
	seen := make(map[string]struct{}, len(records))

	for _, rec := range records {
		cid := rec.GetCid()
		if _, dup := seen[cid]; dup {
			continue
		}

		seen[cid] = struct{}{}

		scored = append(scored, ranking.Scored{
			CID:    cid,
			Item:   rec,
			Result: c.scoreRecord(rec, now),
		})
	}

	ranking.SortDescending(scored)

	return scored, nil
}

// scoreRecord builds Signals for one record and runs Score. Verification
// lookup failures downgrade Status to PARTIAL rather than failing the
// whole request.
func (c *searchCtlr) scoreRecord(rec types.Record, now time.Time) ranking.Result {
	data, err := rec.GetRecordData()
	if err != nil {
		searchLogger.Warn("Failed to decode record for ranking; using defaults", "cid", rec.GetCid(), "error", err)

		return ranking.Score(ranking.Signals{
			Status: corev1.ScoreStatus_SCORE_STATUS_DEFAULTS_ONLY,
		}, c.cfg)
	}

	signed := false
	if sa, ok := rec.(types.SignedAware); ok {
		signed = sa.GetSigned()
	} else if sig := data.GetSignature(); sig != nil {
		signed = true
	}

	signals := ranking.SignalsFromRecord(data, signed, now)
	// Every SQL row is a match by definition.
	signals.QueryRelevance = 1.0

	if verif, vErr := ranking.FetchVerificationSignals(c.db, rec.GetCid()); vErr == nil {
		signals.SigVerified = verif.SigVerified
		signals.NameVerified = verif.NameVerified
	} else {
		searchLogger.Warn("Failed to fetch verification signals; treating as unverified",
			"cid", rec.GetCid(), "error", vErr)

		signals.Status = corev1.ScoreStatus_SCORE_STATUS_PARTIAL
	}

	return ranking.Score(signals, c.cfg)
}
