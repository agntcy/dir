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
			// Check for the oasf-sdk extractor manifest written by `dirctl init`.
			// This mirrors extractor.IsProvisioned without importing the internal package.
			home, homeErr := os.UserHomeDir()
			gomega.Expect(homeErr).NotTo(gomega.HaveOccurred())

			manifest := filepath.Join(home, ".agntcy", "oasf-sdk", "extractor", "manifest.json")
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
			// Delete the pushed record so it does not leak into the shared daemon.
			// The fixture is named "org.agntcy/directory", the same name the daemon
			// self-publishes its skill record under, so a leaked copy would shadow
			// it in name searches run by other suites (e.g. 14_skill_record_test).
			// Mirrors the cleanup in 16_extractor_enricher_test.go.
			if recordCID != "" {
				_ = testEnv.CLI.Delete(recordCID).ShouldSucceed()
			}
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
