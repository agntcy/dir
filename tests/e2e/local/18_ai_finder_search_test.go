// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"encoding/json"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/agntcy/dir/tests/e2e/shared/testdata"
	"github.com/agntcy/dir/tests/e2e/shared/utils"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

// Natural-language search over the HTTP gateway (POST /v1/search).
//
// These run against a real node with a real extractor, in whichever backend the
// environment deploys — in-process assets or an OASF-SDK server. That is the
// point: the handler unit tests drive a fake extractor and cannot catch a
// gateway that resolved no backend, dialed the wrong address, or ranks
// differently from `dirctl search`.
//
// searchQuery decomposes into seven signals, identically on both extractor
// backends: the skills "software_engineering/code_quality/code_review" (0.573)
// and "natural_language_processing/natural_language_generation" (0.436), plus the
// keywords *code*, *review*, *understands*, *natural* and *language*, each
// scoring 1.0. The seeds below are built against that signal set.
const searchQuery = "a code review agent that understands natural language and integrates with MCP servers"

// rankingDescription matches four of searchQuery's five keyword signals
// (*understands*, *natural*, *language*, *code*) and is shared verbatim by the
// two ranking seeds, so their keyword surface is identical and the only thing
// separating them is whether they carry a matching skill.
const rankingDescription = "Understands natural language questions about source code"

// rankingSkill is the skill searchQuery extracts with the highest score. A
// record carrying it matches one signal more than an otherwise identical record.
const rankingSkill = "software_engineering/code_quality/code_review"

const rankingSkillID = 60701

// queryOverMaxLen is one rune past the max_len the handler enforces on
// SearchAgentsRequest.query.
const queryOverMaxLen = 1025

var _ = ginkgo.Describe("AI Finder SearchAgents HTTP API", func() {
	ginkgo.BeforeEach(func() {
		utils.ResetCLIState()
	})

	var (
		tempDir string

		// CIDs of the seeded records, by role in the ranking.
		skillMatchCID  string
		keywordOnlyCID string
		unrelatedCID   string
		seededCIDs     []string
	)

	ginkgo.Context("with a seeded record set", ginkgo.Ordered, func() {
		ginkgo.BeforeAll(func() {
			skipUnlessExtractorGateway()

			var err error

			tempDir, err = os.MkdirTemp("", "ai-finder-search-test")
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Each seed renames the record it is built from, which changes its
			// CID, so these never collide with the same testdata pushed by
			// another spec. All three keep an integration module, without which
			// the catalog cannot project them and search would not consider
			// them candidates at all.
			//
			// The two ranking seeds share a description and carry names holding
			// none of the query's keywords, so they match exactly the same
			// keyword signals. Ranking is by number of signals matched, ties
			// broken on summed signal score, and a keyword scores 1.0 against a
			// skill's 0.573 — so a record cannot be shown to rank above another
			// by carrying a matching skill unless the keyword surface is held
			// equal. Varying the descriptions instead lets the extra keyword hit
			// outweigh the skill hit, which is correct behaviour and tells us
			// nothing about the skill.

			// Matches the four keywords plus rankingSkill: five signals.
			skillMatchCID = pushSeedRecord(tempDir, "skill-match", testdata.ExpectedRecordV110JSON, map[string]any{
				"name":        "e2e_search_ranking_alpha",
				"description": rankingDescription,
				"skills": []map[string]any{
					{"name": rankingSkill, "id": rankingSkillID},
				},
			})

			// The same four keywords, and skills the query does not extract, so
			// it matches four signals — one fewer than the record above.
			keywordOnlyCID = pushSeedRecord(tempDir, "keyword-only", testdata.ExpectedRecordV110JSON, map[string]any{
				"name":        "e2e_search_ranking_beta",
				"description": rankingDescription,
			})

			// Matches no signal at all: its name and description hold none of the
			// keywords and its skill is not one the query extracts. It is here to
			// check that the ranked set leaves such a record out.
			unrelatedCID = pushSeedRecord(tempDir, "unrelated", testdata.ExpectedRecordV100JSON, map[string]any{
				"name": "e2e_search_burger_agent",
			})

			seededCIDs = []string{skillMatchCID, keywordOnlyCID, unrelatedCID}
		})

		ginkgo.AfterAll(func() {
			for _, cid := range seededCIDs {
				if cid != "" {
					_, _ = testEnv.CLI.Delete(cid).Execute()
				}
			}

			if tempDir != "" {
				_ = os.RemoveAll(tempDir)
			}
		})

		ginkgo.It("returns relevance-ranked entries for a free-text query", func(ctx ginkgo.SpecContext) {
			resp := postSearch(ctx, searchRequest{Query: searchQuery, PageSize: 100})

			gomega.Expect(resp.TotalCount).To(gomega.BeNumerically(">=", 2),
				"the seeded records should give the ranking at least two entries to order")
			gomega.Expect(resp.cids()).To(gomega.ContainElement(skillMatchCID))
			gomega.Expect(resp.cids()).To(gomega.ContainElement(keywordOnlyCID))
			gomega.Expect(resp.cids()).NotTo(gomega.ContainElement(unrelatedCID),
				"a record matching no signal should not be in the ranked set")

			// Every entry is a catalog projection, identified by URN rather than
			// a bare CID. Asserting it here pins the hydration step: a handler
			// returning raw CIDs would still satisfy the assertions above.
			for _, entry := range resp.Results {
				gomega.Expect(entry.Identifier).To(gomega.ContainSubstring(":cid:"),
					"entry identifier should be a urn:ai:<host>:cid:<cid> URN")
				gomega.Expect(entry.DisplayName).NotTo(gomega.BeEmpty())
			}
		})

		ginkgo.It("ranks by the number of signals a record matches", func(ctx ginkgo.SpecContext) {
			// The two seeds differ by exactly one matched signal, so this
			// asserts the ordering rule itself rather than a property of any
			// particular query.
			cids := postSearch(ctx, searchRequest{Query: searchQuery, PageSize: 100}).cids()

			skillRank := indexOf(cids, skillMatchCID)
			keywordRank := indexOf(cids, keywordOnlyCID)

			gomega.Expect(skillRank).To(gomega.BeNumerically(">=", 0), "the five-signal record should be ranked")
			gomega.Expect(keywordRank).To(gomega.BeNumerically(">=", 0), "the four-signal record should be ranked")
			gomega.Expect(skillRank).To(gomega.BeNumerically("<", keywordRank),
				"the record matching five signals should outrank the otherwise identical one matching four")
		})

		ginkgo.It("returns a stable ranking across repeated identical requests", func(ctx ginkgo.SpecContext) {
			// Ranking ties break on summed signal score then CID, giving a total
			// order. Without it a paginated caller would see records repeat or
			// vanish between pages.
			first := postSearch(ctx, searchRequest{Query: searchQuery, PageSize: 100}).cids()
			gomega.Expect(first).NotTo(gomega.BeEmpty())

			for range 2 {
				gomega.Expect(postSearch(ctx, searchRequest{Query: searchQuery, PageSize: 100}).cids()).
					To(gomega.Equal(first), "repeated identical requests should return the same order")
			}
		})

		ginkgo.It("pages the ranked set into disjoint pages", func(ctx ginkgo.SpecContext) {
			full := postSearch(ctx, searchRequest{Query: searchQuery, PageSize: 100})
			gomega.Expect(len(full.cids())).To(gomega.BeNumerically(">=", 2),
				"paging assertions need at least two ranked entries")

			pageOne := postSearch(ctx, searchRequest{Query: searchQuery, PageSize: 1})
			gomega.Expect(pageOne.cids()).To(gomega.Equal(full.cids()[:1]))
			gomega.Expect(pageOne.NextPageToken).NotTo(gomega.BeEmpty())

			pageTwo := postSearch(ctx, searchRequest{
				Query:     searchQuery,
				PageSize:  1,
				PageToken: pageOne.NextPageToken,
			})
			gomega.Expect(pageTwo.cids()).To(gomega.Equal(full.cids()[1:2]))
			gomega.Expect(pageTwo.cids()).NotTo(gomega.ContainElement(pageOne.cids()[0]),
				"consecutive pages should be disjoint")

			// A page covering the whole ranked set reports no continuation, so a
			// caller following the tokens terminates.
			if full.TotalCount <= 100 {
				gomega.Expect(full.NextPageToken).To(gomega.BeEmpty())
			}
		})

		ginkgo.It("returns the same CIDs in the same order as `dirctl search`", func(ctx ginkgo.SpecContext) {
			// Only the local backend is shared: the CLI loads the same
			// provisioned assets the gateway does. Under kind the OASF-SDK
			// Service is ClusterIP-only, so the CLI cannot reach the backend the
			// gateway uses and the two rankings are not comparable.
			if testEnv.Config.ExtractorMode != "local" {
				ginkgo.Skip("this environment's CLI and gateway do not share an extractor, so their rankings are not comparable")
			}

			// Both paths run the same nlsearch decomposition and fan-out, so the
			// same query must produce the same order. They do not see the same
			// record universe: the gateway restricts candidates to records the
			// catalog can project, the CLI does not. Comparing the two lists
			// filtered to the seeded records asserts the shared ordering without
			// depending on what else the store holds.
			apiOrder := postSearch(ctx, searchRequest{Query: searchQuery, PageSize: 100}).cids()

			raw := testEnv.CLI.Command("search").WithArgs(
				searchQuery,
				"--format", "cid",
				"--limit", "100",
				"--output", "jsonl",
			).ShouldSucceed()

			gomega.Expect(retain(cliCIDs(raw), seededCIDs)).To(gomega.Equal(retain(apiOrder, seededCIDs)),
				"`dirctl search` and POST /v1/search should rank the seeded records identically")
		})

		ginkgo.It("rejects an empty query with HTTP 400", func(ctx ginkgo.SpecContext) {
			status, _ := gatewayPostJSON(ctx, "/v1/search", searchRequest{Query: "   "})

			gomega.Expect(status).To(gomega.Equal(http.StatusBadRequest))
		})

		ginkgo.It("rejects an over-long query with HTTP 400", func(ctx ginkgo.SpecContext) {
			// The service registers no protovalidate interceptor, so the proto's
			// max_len is enforced in handler code. This is the e2e check that it
			// actually is.
			status, _ := gatewayPostJSON(ctx, "/v1/search", searchRequest{
				Query: strings.Repeat("a", queryOverMaxLen),
			})

			gomega.Expect(status).To(gomega.Equal(http.StatusBadRequest))
		})
	})
})

// pushSeedRecord writes a record built from base with the given top-level fields
// replaced, pushes it, and returns its CID. Renaming a record changes its CID,
// which is what keeps these seeds distinct from the same testdata pushed
// elsewhere in the suite.
func pushSeedRecord(dir, label string, base []byte, overrides map[string]any) string {
	ginkgo.GinkgoHelper()

	var doc map[string]any
	gomega.Expect(json.Unmarshal(base, &doc)).To(gomega.Succeed())

	maps.Copy(doc, overrides)

	data, err := json.Marshal(doc)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	path := filepath.Join(dir, label+".json")
	gomega.Expect(os.WriteFile(path, data, 0o600)).To(gomega.Succeed())

	cid := testEnv.CLI.Push(path).WithArgs("--output", "raw").ShouldSucceed()
	gomega.Expect(cid).NotTo(gomega.BeEmpty(), "pushing seed record %q should return a CID", label)

	return cid
}

// cliCIDs parses the CIDs out of `dirctl search --format cid --output jsonl`,
// which prints one JSON-quoted CID per line, in rank order.
func cliCIDs(raw string) []string {
	ginkgo.GinkgoHelper()

	var out []string

	for line := range strings.SplitSeq(strings.TrimSpace(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line == "[]" {
			continue
		}

		var cid string
		gomega.Expect(json.Unmarshal([]byte(line), &cid)).To(gomega.Succeed(),
			"unexpected line in `dirctl search` jsonl output: %q", line)

		out = append(out, cid)
	}

	return out
}

// retain filters cids down to the members of keep, preserving the order of cids.
func retain(cids, keep []string) []string {
	wanted := make(map[string]struct{}, len(keep))
	for _, cid := range keep {
		wanted[cid] = struct{}{}
	}

	out := make([]string, 0, len(keep))

	for _, cid := range cids {
		if _, ok := wanted[cid]; ok {
			out = append(out, cid)
		}
	}

	return out
}

// indexOf returns the position of cid in cids, or -1 when absent.
func indexOf(cids []string, cid string) int {
	for i, c := range cids {
		if c == cid {
			return i
		}
	}

	return -1
}
