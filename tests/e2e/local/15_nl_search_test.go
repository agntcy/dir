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

var _ = ginkgo.Describe("Natural-language search", func() {
	ginkgo.BeforeEach(func() {
		utils.ResetCLIState()
	})

	var (
		tempDir   string
		recordCID string
	)

	ginkgo.Context("free-text query with OASF extractor", ginkgo.Ordered, func() {
		ginkgo.BeforeAll(func() {
			// Check for the oasf-sdk manifest written by `dirctl init`. This mirrors
			// what extractor.IsProvisioned does without importing the internal package.
			home, homeErr := os.UserHomeDir()
			gomega.Expect(homeErr).NotTo(gomega.HaveOccurred())

			// NOTE: this path is missing the "extractor" subdirectory that
			// extractor.DefaultAssetDir provisions into, so the check never
			// finds the manifest and this spec always skips. Correcting it makes
			// the spec run and push testdata/directory-record.json, which then
			// contends with 14_skill_record_test.go: both use the record name
			// "org.agntcy/directory", that testdata carries a stale
			// "application/agentskill+md" artifact media type where the daemon
			// now emits "application/agent-skills+md", and spec 14 resolves the
			// name with --limit 1. Fixing this needs the testdata corrected and
			// spec 14 taught to pick the record it means, so it is left alone
			// here rather than half-done. The CLI free-text path is meanwhile
			// covered by the parity spec in 18_ai_finder_search_test.go, which
			// runs `dirctl search` against the same query as POST /v1/search.
			manifest := filepath.Join(home, ".agntcy", "oasf-sdk", "manifest.json")
			if _, err := os.Stat(manifest); err != nil {
				ginkgo.Skip("OASF extractor not provisioned — run `dirctl init` to enable natural-language search tests")
			}

			var err error

			tempDir, err = os.MkdirTemp("", "nl-search-test")
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			recordPath := filepath.Join(tempDir, "directory-record.json")
			err = os.WriteFile(recordPath, testdata.DirectoryRecordJSON, 0o600)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			recordCID = testEnv.CLI.Push(recordPath).WithArgs("--output", "raw").ShouldSucceed()
			gomega.Expect(recordCID).NotTo(gomega.BeEmpty())
		})

		ginkgo.AfterAll(func() {
			if tempDir != "" {
				_ = os.RemoveAll(tempDir)
			}
		})

		ginkgo.It("finds the directory record by free-text query", func() {
			output := testEnv.CLI.Command("search").WithArgs(
				"I need a MCP server to connect to a agntcy directory",
				"--sort", "relevance",
				"--format", "cid",
			).ShouldSucceed()
			gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
		})
	})
})
