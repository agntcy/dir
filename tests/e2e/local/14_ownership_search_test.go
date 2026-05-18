// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"os"
	"path/filepath"

	"github.com/agntcy/dir/tests/e2e/shared/testdata"
	"github.com/agntcy/dir/tests/e2e/shared/utils"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

// Ownership claim search tests.
//
// Flow:
//  1. Push a record (no ownership declared in content).
//  2. Claim ownership via `dirctl ownership claim --record <CID> --owner <id>`.
//  3. Search with `--owner <id>` — expect CID in results.
//  4. Search with a non-matching owner — expect CID NOT in results.
//
// Authentication is disabled in local e2e tests, so the server skips the
// caller-identity == owner_id check and accepts any claim.

var _ = ginkgo.Describe("Ownership-based search", func() {
	ginkgo.BeforeEach(func() {
		utils.ResetCLIState()
	})

	var (
		tempDir   string
		recordCID string
	)

	const testOwner = "alice@acme.com"

	ginkgo.Context("searching by ownership claim", ginkgo.Ordered, func() {
		ginkgo.BeforeAll(func() {
			var err error

			tempDir, err = os.MkdirTemp("", "ownership-search-test")
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			recordPath := filepath.Join(tempDir, "record_080_v4.json")
			err = os.WriteFile(recordPath, testdata.ExpectedRecordV080V4JSON, 0o600)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Step 1: push the record.
			recordCID = testEnv.CLI.Push(recordPath).WithArgs("--output", "raw").ShouldSucceed()
			gomega.Expect(recordCID).NotTo(gomega.BeEmpty())

			// Step 2: claim ownership.
			testEnv.CLI.Ownership("claim").
				WithArgs("--record", recordCID, "--owner", testOwner).
				ShouldSucceed()
		})

		ginkgo.AfterAll(func() {
			if tempDir != "" {
				_ = os.RemoveAll(tempDir)
			}
		})

		// Exact match on owner_id.
		ginkgo.It("finds record by exact owner identity", func() {
			output := testEnv.CLI.Search().
				WithOwner(testOwner).
				ShouldSucceed()
			gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
		})

		// Wildcard suffix on domain.
		ginkgo.It("finds record with wildcard owner domain", func() {
			output := testEnv.CLI.Search().
				WithOwner("*@acme.com").
				ShouldSucceed()
			gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
		})

		// Wildcard prefix on local part.
		ginkgo.It("finds record with wildcard owner local part", func() {
			output := testEnv.CLI.Search().
				WithOwner("alice@*").
				ShouldSucceed()
			gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
		})

		// Ownership combined with name filter (AND logic).
		ginkgo.It("finds record when ownership combined with name filter", func() {
			output := testEnv.CLI.Search().
				WithName("*research-assistant*").
				WithOwner(testOwner).
				ShouldSucceed()
			gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
		})

		// Negative: wrong owner.
		ginkgo.It("returns no results for non-matching owner", func() {
			output := testEnv.CLI.Search().
				WithOwner("bob@acme.com").
				ShouldSucceed()
			gomega.Expect(output).NotTo(gomega.ContainSubstring(recordCID))
		})

		// Negative: wrong domain wildcard.
		ginkgo.It("returns no results for non-matching owner domain wildcard", func() {
			output := testEnv.CLI.Search().
				WithOwner("*@other.com").
				ShouldSucceed()
			gomega.Expect(output).NotTo(gomega.ContainSubstring(recordCID))
		})

		// Negative: conflicting owner + name filter.
		ginkgo.It("returns no results when owner matches but name does not", func() {
			output := testEnv.CLI.Search().
				WithName("nonexistent-agent").
				WithOwner(testOwner).
				ShouldSucceed()
			gomega.Expect(output).NotTo(gomega.ContainSubstring(recordCID))
		})
	})
})
