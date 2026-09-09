// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/agntcy/dir/tests/e2e/shared/testdata"
	"github.com/agntcy/dir/tests/e2e/shared/utils"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

// Taxonomy extraction over the HTTP gateway (POST /v1/extract).
//
// extractText describes an agent concretely enough that both extractor backends
// return skills and domains above the score floor rather than keywords alone.
const extractText = "an agent that reviews source code, files defects and answers questions about a codebase"

// textOverMaxLen is one rune past the max_len the handler enforces on
// ExtractTaxonomyRequest.text.
const textOverMaxLen = 1025

// taxonomyVersions are the OASF schema versions the extractor draws classes
// from, newest first, kept in step with taxonomy_versions in the extractor asset
// manifest. Each maps to a testdata record that already validates under it.
//
// One extraction mixes versions freely, and ExtractTaxonomyRequest carries no
// version, so each returned class is checked against every version and has to be
// recognised by at least one. OASF renamed whole trees between 1.0.0 and 1.1.0,
// so a single response can hold classes no single version accepts together: for
// the phrase below, the skills split across 1.1.0 and 1.0.0 and so do the
// domains. Probing classes together would make this spec fail whenever the
// extractor's picks happen to straddle that boundary.
var taxonomyVersions = []string{"1.1.0", "1.0.0", "0.8.0", "0.7.0"}

// probeBase returns a testdata record that validates under the given schema
// version. Building each probe on one of these keeps every version's own record
// shape out of this test — locators, for one, took a single `url` before 1.0.0
// and a `urls` list after — so the class under test is the only thing a
// rejection can be about.
func probeBase(version string) []byte {
	switch version {
	case "1.1.0":
		return testdata.ExpectedRecordV110JSON
	case "1.0.0":
		return testdata.ExpectedRecordV100JSON
	case "0.8.0":
		return testdata.ExpectedRecordV080V4JSON
	case "0.7.0":
		return testdata.ExpectedRecordV070JSON
	}

	return nil
}

var _ = ginkgo.Describe("AI Finder ExtractTaxonomy HTTP API", func() {
	ginkgo.BeforeEach(func() {
		utils.ResetCLIState()
		skipUnlessExtractorGateway()
	})

	ginkgo.It("returns scored skills and domains for a descriptive phrase", func(ctx ginkgo.SpecContext) {
		resp := postExtract(ctx, extractText)

		gomega.Expect(resp.Skills).NotTo(gomega.BeEmpty(), "extractor should return at least one skill")
		gomega.Expect(resp.Domains).NotTo(gomega.BeEmpty(), "extractor should return at least one domain")

		assertScoredClasses(resp.Skills, "skill")
		assertScoredClasses(resp.Domains, "domain")
	})

	ginkgo.It("returns class names that exist in the OASF taxonomy", func(ctx ginkgo.SpecContext) {
		// The extractor's own tests can only check it against its own assets.
		// Pushing a record built from what it returned checks it against the
		// node's configured OASF schema endpoint instead, which rejects an
		// unknown class id or name outright. That is the property that makes the
		// suggestions usable: a class the schema does not recognise cannot be
		// put on a record, so offering it would be a dead end.
		//
		// Every returned class is checked, not just the best one. The endpoint
		// offers the whole list, so a bogus class at any rank is a bogus
		// suggestion, and the lower ranks are where the taxonomy versions mix.
		resp := postExtract(ctx, extractText)

		gomega.Expect(resp.Skills).NotTo(gomega.BeEmpty())
		gomega.Expect(resp.Domains).NotTo(gomega.BeEmpty())

		tempDir, err := os.MkdirTemp("", "ai-finder-extract-test")
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		defer func() { _ = os.RemoveAll(tempDir) }()

		for i, skill := range resp.Skills {
			assertClassIsInTaxonomy(tempDir, "skill", i, skill)
		}

		for i, domain := range resp.Domains {
			assertClassIsInTaxonomy(tempDir, "domain", i, domain)
		}
	})

	ginkgo.It("rejects empty text with HTTP 400", func(ctx ginkgo.SpecContext) {
		status, _ := gatewayPostJSON(ctx, "/v1/extract", map[string]string{"text": "   "})

		gomega.Expect(status).To(gomega.Equal(http.StatusBadRequest))
	})

	ginkgo.It("rejects over-long text with HTTP 400", func(ctx ginkgo.SpecContext) {
		status, _ := gatewayPostJSON(ctx, "/v1/extract", map[string]string{
			"text": strings.Repeat("a", textOverMaxLen),
		})

		gomega.Expect(status).To(gomega.Equal(http.StatusBadRequest))
	})
})

// assertScoredClasses checks the shape and ordering of one returned class list:
// hierarchical names, real ids, scores inside (0,1], tiers starting at 1, and
// descending score, which is the order callers rank suggestions by.
func assertScoredClasses(classes []scoredClass, kind string) {
	ginkgo.GinkgoHelper()

	for i, cl := range classes {
		gomega.Expect(cl.Name).NotTo(gomega.BeEmpty(), "%s %d should have a name", kind, i)
		gomega.Expect(cl.Name).To(gomega.ContainSubstring("/"),
			"%s %q should be a hierarchical OASF class name", kind, cl.Name)
		gomega.Expect(cl.ID).To(gomega.BeNumerically(">", 0), "%s %q should have an OASF uid", kind, cl.Name)
		gomega.Expect(cl.Score).To(gomega.BeNumerically(">", 0), "%s %q should have a positive score", kind, cl.Name)
		gomega.Expect(cl.Score).To(gomega.BeNumerically("<=", 1), "%s %q score should be normalised", kind, cl.Name)
		gomega.Expect(cl.Tier).To(gomega.BeNumerically(">=", 1), "%s %q should have a tier", kind, cl.Name)

		if i > 0 {
			gomega.Expect(cl.Score).To(gomega.BeNumerically("<=", classes[i-1].Score),
				"%ss should be returned in descending score order", kind)
		}
	}
}

// assertClassIsInTaxonomy pushes a record carrying one extracted class and
// requires at least one OASF schema version to accept it. The node validates a
// push against its configured schema endpoint, which rejects an unknown class id
// or name, so a class no version defines fails every attempt. Accepted records
// are deleted again.
//
// One class per record, because a response can hold classes that no single
// version accepts together. rank is the class's position in the response, so a
// failure names which suggestion was bad.
func assertClassIsInTaxonomy(dir, kind string, rank int, class scoredClass) {
	ginkgo.GinkgoHelper()

	label := fmt.Sprintf("%s-%d", kind, rank)

	for _, version := range taxonomyVersions {
		path := filepath.Join(dir, label+"-"+version+".json")
		writeTaxonomyProbeRecord(path, version, label, kind+"s", class)

		cid, err := testEnv.CLI.Push(path).WithArgs("--output", "raw").SuppressStderr().Execute()
		if err == nil && strings.TrimSpace(cid) != "" {
			_, _ = testEnv.CLI.Delete(strings.TrimSpace(cid)).Execute()

			return
		}
	}

	ginkgo.Fail(fmt.Sprintf(
		"extracted %s %d, %q (id %d), was rejected by every OASF version tried (%s), "+
			"so it is not a class any of them define",
		kind, rank, class.Name, class.ID, strings.Join(taxonomyVersions, ", ")))
}

// writeTaxonomyProbeRecord writes a probe record for one class, built from the
// testdata record that already validates under this schema version so nothing
// but the class can be rejected. field is "skills" or "domains" and is replaced
// wholesale; the base record's own value for the other one is left in place,
// which is what gives a domain probe a skill the same version recognises —
// skills are required, and the base's are valid there by construction.
//
// Modules are dropped. The record only has to validate, and without one it has
// no catalog projection, so a probe can never turn up in the search specs'
// results even if a delete is lost.
func writeTaxonomyProbeRecord(path, schemaVersion, label, field string, class scoredClass) {
	ginkgo.GinkgoHelper()

	base := probeBase(schemaVersion)
	gomega.Expect(base).NotTo(gomega.BeNil(), "no probe base record for OASF version %s", schemaVersion)

	var doc map[string]any
	gomega.Expect(json.Unmarshal(base, &doc)).To(gomega.Succeed())

	// Renamed so the probe never shares a CID with the same testdata pushed
	// elsewhere in the suite, and so a leaked probe is identifiable.
	doc["name"] = "e2e_extract_taxonomy_probe_" + label + "_" + strings.ReplaceAll(schemaVersion, ".", "_")
	doc["description"] = "Record used only to check an extracted OASF class against the schema"
	doc[field] = []map[string]any{{"name": class.Name, "id": class.ID}}

	delete(doc, "modules")

	data, err := json.Marshal(doc)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	gomega.Expect(os.WriteFile(path, data, 0o600)).To(gomega.Succeed())
}
