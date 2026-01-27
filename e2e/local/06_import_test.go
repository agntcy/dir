// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"github.com/agntcy/dir/e2e/shared/config"
	"github.com/agntcy/dir/e2e/shared/utils"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Running dirctl end-to-end tests for the import command", func() {
	var cli *utils.CLI

	ginkgo.BeforeEach(func() {
		if cfg.DeploymentMode != config.DeploymentModeLocal {
			ginkgo.Skip("Skipping test, not in local mode")
		}

		utils.ResetCLIState()
		// Initialize CLI helper
		cli = utils.NewCLI()
	})

	ginkgo.Context("MCP registry import functionality", func() {
		ginkgo.It("should fail gracefully when enrichment cannot be initialized", func() {
			// Test that import fails with a clear error when enrichment is required but cannot be initialized
			// This happens when mcphost.json is missing or invalid (no LLM configured)
			output, err := cli.Command("import").
				WithArgs("--type=mcp", "--url=https://registry.modelcontextprotocol.io/v0.1", "--limit", "1", "--enrich-config=/nonexistent/path.json").
				Execute()

			ginkgo.GinkgoWriter.Printf("Import error output: %s\n", output)
			ginkgo.GinkgoWriter.Printf("Import error: %v\n", err)

			// Verify command failed
			gomega.Expect(err).To(gomega.HaveOccurred())

			// Verify error message indicates enrichment is mandatory
			// The error message is in the wrapped error, not the output (stderr vs stdout)
			gomega.Expect(err.Error()).To(gomega.ContainSubstring("enrichment is mandatory"))
		})
	})
})
