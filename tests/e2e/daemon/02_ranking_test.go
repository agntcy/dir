// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package daemon

import (
	"context"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	searchv1 "github.com/agntcy/dir/api/search/v1"
	"github.com/agntcy/dir/client/streaming"
	"github.com/agntcy/dir/tests/e2e/shared/testdata"
	"github.com/agntcy/dir/tests/e2e/shared/utils"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

// Three fixtures, all carrying a skill under natural_language_processing/*:
//   - record_100.json:    schema 1.0.0, 4 entries — top schema_recency
//   - record_080_v4.json: schema 0.8.0, 5 entries — top completeness
//   - record_080_v5.json: schema 0.8.0, 1 entry   — worst on every signal
//
// Predicted order: v100 > v080/v4 > v080/v5. schema_recency edges out
// completeness: ~46 points vs ~6. record_070 is avoided here because
// 01_daemon_test.go pushes it and the OCI store rejects duplicate blob
// writes for the same CID. We assert relative order and explanation
// shape, not absolute scores (freshness depends on time.Now()).

type rankingFixture struct {
	name string
	data []byte
	cid  string
}

var _ = ginkgo.Describe("Ranking e2e", ginkgo.Ordered, ginkgo.Serial, func() {
	const rankingSkillQuery = "natural_language_processing/*"

	var fixtures []rankingFixture

	ginkgo.BeforeAll(func(ctx context.Context) {
		// Push is idempotent on CID; safe even if an earlier spec already
		// inserted one of these fixtures.
		toPush := []rankingFixture{
			{name: "v1.0.0", data: testdata.ExpectedRecordV100JSON},
			{name: "v0.8.0/v4", data: testdata.ExpectedRecordV080V4JSON},
			{name: "v0.8.0/v5", data: testdata.ExpectedRecordV080V5JSON},
		}

		fixtures = make([]rankingFixture, 0, len(toPush))
		for _, f := range toPush {
			rec, err := corev1.UnmarshalRecord(f.data)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "unmarshal %s", f.name)

			ref, err := testEnv.Client.Push(ctx, rec)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "push %s", f.name)
			gomega.Expect(ref.GetCid()).NotTo(gomega.BeEmpty(), "push %s returned empty CID", f.name)

			f.cid = ref.GetCid()
			fixtures = append(fixtures, f)
		}
	})

	ginkgo.It("returns results sorted by rank_score DESC", func(ctx context.Context) {
		resps := searchAllRecords(ctx, &searchv1.SearchRecordsRequest{
			Queries: []*searchv1.RecordQuery{{
				Type:  searchv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL_NAME,
				Value: rankingSkillQuery,
			}},
		})

		// Other specs push records under the same prefix; we may see a
		// superset and filter down later.
		gomega.Expect(len(resps)).To(gomega.BeNumerically(">=", len(fixtures)),
			"skill wildcard %q should return at least our three ranking fixtures",
			rankingSkillQuery)

		// Score-DESC invariant must hold for the full stream.
		for i := 1; i < len(resps); i++ {
			gomega.Expect(resps[i].GetRankScore()).To(
				gomega.BeNumerically("<=", resps[i-1].GetRankScore()),
				"results must be sorted by rank_score DESC, but pos %d (%d) > pos %d (%d)",
				i, resps[i].GetRankScore(), i-1, resps[i-1].GetRankScore())
		}

		// Every row carries an explanation with a non-UNSPECIFIED status.
		// QueryRelevance is always 1000 (every SQL row is a match by
		// definition) and popularity is always 0 (no DHT for local search).
		for i, r := range resps {
			gomega.Expect(r.GetRankExplanation()).NotTo(gomega.BeNil(),
				"result %d missing rank_explanation", i)
			gomega.Expect(r.GetRankExplanation().GetScoreStatus()).
				NotTo(gomega.Equal(corev1.ScoreStatus_SCORE_STATUS_UNSPECIFIED),
					"result %d has unspecified score_status", i)
			gomega.Expect(r.GetRankExplanation().GetQueryRelevance()).
				To(gomega.Equal(uint32(1000)),
					"local-search query_relevance must be 1000 (perfect match)")
			gomega.Expect(r.GetRankExplanation().GetPopularity()).
				To(gomega.Equal(uint32(0)),
					"local-search popularity must be 0 (no DHT input)")
		}

		// Project to our subset to assert relative order. seenCIDs is
		// inlined into the failure message because Gomega's proto dump
		// truncates and hides CIDs.
		seenCIDs := make([]string, 0, len(resps))
		for _, r := range resps {
			seenCIDs = append(seenCIDs, r.GetRecord().GetCid())
		}

		ours := filterByCID(resps, fixtureCIDs(fixtures))
		gomega.Expect(ours).To(gomega.HaveLen(len(fixtures)),
			"every fixture CID must appear in the ranked stream once;\n  seen=%v\n  fixtures=%v",
			seenCIDs, fixtures)

		gomega.Expect(ours[0].GetRecord().GetCid()).
			To(gomega.Equal(cidFor(fixtures, "v1.0.0")),
				"v1.0.0 fixture should outrank both v0.8.0 fixtures (schema_recency)")
		gomega.Expect(ours[1].GetRecord().GetCid()).
			To(gomega.Equal(cidFor(fixtures, "v0.8.0/v4")),
				"v0.8.0/v4 should outrank v0.8.0/v5 (more taxonomy entries)")
		gomega.Expect(ours[2].GetRecord().GetCid()).
			To(gomega.Equal(cidFor(fixtures, "v0.8.0/v5")),
				"v0.8.0/v5 should rank last (worst on every per-record signal)")
	})

	ginkgo.It("respects offset and limit on the ranked window", func(ctx context.Context) {
		// Compare against the unbounded stream so the test is robust to
		// whatever other specs put in the DB.
		full := searchAllRecords(ctx, &searchv1.SearchRecordsRequest{
			Queries: []*searchv1.RecordQuery{{
				Type:  searchv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL_NAME,
				Value: rankingSkillQuery,
			}},
		})
		gomega.Expect(len(full)).To(gomega.BeNumerically(">=", 2),
			"need at least two rows in the unbounded stream to test offset+limit")

		page := searchAllRecords(ctx, &searchv1.SearchRecordsRequest{
			Queries: []*searchv1.RecordQuery{{
				Type:  searchv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL_NAME,
				Value: rankingSkillQuery,
			}},
			Offset: utils.Ptr[uint32](1),
			Limit:  utils.Ptr[uint32](1),
		})
		gomega.Expect(page).To(gomega.HaveLen(1),
			"offset=1, limit=1 must return exactly one row")
		gomega.Expect(page[0].GetRecord().GetCid()).
			To(gomega.Equal(full[1].GetRecord().GetCid()),
				"offset=1, limit=1 must return the second row of the unbounded ranked stream")
		gomega.Expect(page[0].GetRankScore()).
			To(gomega.Equal(full[1].GetRankScore()),
				"score must be stable across paginated requests")
	})

	ginkgo.It("propagates rank_score on SearchCIDs as well", func(ctx context.Context) {
		cids := searchAllCIDs(ctx, &searchv1.SearchCIDsRequest{
			Queries: []*searchv1.RecordQuery{{
				Type:  searchv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL_NAME,
				Value: rankingSkillQuery,
			}},
		})

		gomega.Expect(len(cids)).To(gomega.BeNumerically(">=", len(fixtures)),
			"SearchCIDs should return at least our three ranking fixtures")

		for i := 1; i < len(cids); i++ {
			gomega.Expect(cids[i].GetRankScore()).To(
				gomega.BeNumerically("<=", cids[i-1].GetRankScore()),
				"SearchCIDs results must be sorted by rank_score DESC")
		}

		gomega.Expect(cids[0].GetRankExplanation()).NotTo(gomega.BeNil())
	})
})

func searchAllRecords(ctx context.Context, req *searchv1.SearchRecordsRequest) []*searchv1.SearchRecordsResponse {
	ginkgo.GinkgoHelper()

	stream, err := testEnv.Client.SearchRecords(ctx, req)
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "SearchRecords call failed")

	return drainStream(ctx, stream, "SearchRecords")
}

func searchAllCIDs(ctx context.Context, req *searchv1.SearchCIDsRequest) []*searchv1.SearchCIDsResponse {
	ginkgo.GinkgoHelper()

	stream, err := testEnv.Client.SearchCIDs(ctx, req)
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "SearchCIDs call failed")

	return drainStream(ctx, stream, "SearchCIDs")
}

// drainStream collects every value from stream. Fails the spec on
// error, 30s timeout, or context cancellation. The fixed timeout is
// shorter than Ginkgo's 2h suite cap so wedged streams fail loudly.
func drainStream[T any](ctx context.Context, stream streaming.StreamResult[T], label string) []*T {
	ginkgo.GinkgoHelper()

	deadline := time.NewTimer(30 * time.Second)
	defer deadline.Stop()

	var out []*T

	for {
		select {
		case resp, ok := <-stream.ResCh():
			if !ok {
				return out
			}

			out = append(out, resp)
		case e := <-stream.ErrCh():
			gomega.Expect(e).NotTo(gomega.HaveOccurred(), "%s stream error", label)
		case <-stream.DoneCh():
			return out
		case <-deadline.C:
			ginkgo.Fail(label + " stream did not complete within 30s")

			return out
		case <-ctx.Done():
			ginkgo.Fail("context cancelled before " + label + " stream completed")

			return out
		}
	}
}

func cidFor(fixtures []rankingFixture, name string) string {
	for _, f := range fixtures {
		if f.name == name {
			return f.cid
		}
	}

	return ""
}

func fixtureCIDs(fixtures []rankingFixture) map[string]struct{} {
	out := make(map[string]struct{}, len(fixtures))
	for _, f := range fixtures {
		out[f.cid] = struct{}{}
	}

	return out
}

// filterByCID keeps order so we can assert the relative ranking of our
// subset, ignoring whatever other rows the daemon happens to know.
func filterByCID(resps []*searchv1.SearchRecordsResponse, keep map[string]struct{}) []*searchv1.SearchRecordsResponse {
	out := make([]*searchv1.SearchRecordsResponse, 0, len(keep))
	for _, r := range resps {
		if _, ok := keep[r.GetRecord().GetCid()]; ok {
			out = append(out, r)
		}
	}

	return out
}
