// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"github.com/agntcy/dir/tests/e2e/shared/utils"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Running dirctl end-to-end tests for the import command", func() {
	ginkgo.BeforeEach(func() {
		utils.ResetCLIState()
	})

	ginkgo.Context("MCP registry import functionality", func() {
		ginkgo.It("should fail gracefully when enrichment cannot be initialized", func() {
			// Test that import fails with a clear error when enrichment is required but cannot be initialized
			// This happens when enricher config is missing or invalid
			output, err := testEnv.CLI.Command("import").
				WithArgs("--type=mcp-registry", "--url=https://registry.modelcontextprotocol.io/v0.1", "--limit", "1", "--enrich-config=/nonexistent/path.json").
				Execute()

			ginkgo.GinkgoWriter.Printf("Import error output: %s\n", output)
			ginkgo.GinkgoWriter.Printf("Import error: %v\n", err)

			// Verify command failed
			gomega.Expect(err).To(gomega.HaveOccurred())

			// Enricher init fails when enricher config is missing/invalid; errors are wrapped by CLI and importer.
			// Validation catches nonexistent config files before any network/gRPC calls.
			gomega.Expect(err.Error()).To(gomega.Or(
				gomega.ContainSubstring("config file not found"),
				gomega.ContainSubstring("enricher configuration is invalid"),
				gomega.ContainSubstring("failed to create enricher"),
				gomega.ContainSubstring("enricher tool host"),
			))
		})
	})
})
