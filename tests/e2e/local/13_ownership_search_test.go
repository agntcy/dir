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

// Test data for OASF 0.8.0 record (record_080_v4.json).
// The ownership feature stores ownership claims as OCI referrers.
// The DB owners table is the search index derived from those referrers.
// The controller indexes the claim on push; the reconciler re-syncs it.

const testOwnerID = "spiffe://example.org/agent/test-owner"

var _ = ginkgo.Describe("Ownership-based search", func() {
	ginkgo.BeforeEach(func() {
		utils.ResetCLIState()
	})

	var (
		tempDir   string
		recordCID string
	)

	ginkgo.Context("searching by owner identity", ginkgo.Ordered, func() {
		ginkgo.BeforeAll(func() {
			var err error

			tempDir, err = os.MkdirTemp("", "ownership-search-test")
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			recordPath := filepath.Join(tempDir, "record_080_v4.json")
			err = os.WriteFile(recordPath, testdata.ExpectedRecordV080V4JSON, 0o600)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			recordCID = testEnv.CLI.Push(recordPath).WithArgs("--output", "raw").ShouldSucceed()
			gomega.Expect(recordCID).NotTo(gomega.BeEmpty())

			// Attach an ownership claim referrer (no auth enforcement in test environment).
			testEnv.CLI.Ownership().Claim(recordCID, testOwnerID).ShouldSucceed()
		})

		ginkgo.AfterAll(func() {
			if tempDir != "" {
				_ = os.RemoveAll(tempDir)
			}
		})

		// Exact match
		ginkgo.It("finds record by exact owner identity", func() {
			output := testEnv.CLI.Search().
				WithOwner(testOwnerID).
				ShouldSucceed()
			gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
		})

		// Wildcard on domain
		ginkgo.It("finds record with wildcard on SPIFFE domain", func() {
			output := testEnv.CLI.Search().
				WithOwner("spiffe://example.org/*").
				ShouldSucceed()
			gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
		})

		// Wildcard on local part
		ginkgo.It("finds record with wildcard on SPIFFE local path", func() {
			output := testEnv.CLI.Search().
				WithOwner("*test-owner*").
				ShouldSucceed()
			gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
		})

		// Combined with name filter
		ginkgo.It("finds record when owner is combined with name filter", func() {
			output := testEnv.CLI.Search().
				WithOwner(testOwnerID).
				WithName("*research-assistant*").
				ShouldSucceed()
			gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
		})

		// Negative: wrong owner
		ginkgo.It("does not find record with non-matching owner", func() {
			output := testEnv.CLI.Search().
				WithOwner("spiffe://other.org/agent/other-owner").
				ShouldSucceed()
			gomega.Expect(output).NotTo(gomega.ContainSubstring(recordCID))
		})

		// Negative: wrong domain wildcard
		ginkgo.It("does not find record with wildcard matching different domain", func() {
			output := testEnv.CLI.Search().
				WithOwner("spiffe://other.org/*").
				ShouldSucceed()
			gomega.Expect(output).NotTo(gomega.ContainSubstring(recordCID))
		})

		// Negative: correct owner but wrong name
		ginkgo.It("does not find record when owner matches but name does not", func() {
			output := testEnv.CLI.Search().
				WithOwner(testOwnerID).
				WithName("nonexistent-agent-xyz").
				ShouldSucceed()
			gomega.Expect(output).NotTo(gomega.ContainSubstring(recordCID))
		})
	})
})
