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
		ginkgo.It("should fail gracefully when config file does not exist", func() {
			// Test that import fails with a clear error when the --config file path is invalid
			output, err := testEnv.CLI.Command("import").
				WithArgs("--type=mcp-registry", "--url=https://registry.modelcontextprotocol.io/v0.1", "--limit", "1", "--config=/nonexistent/path.yaml").
				Execute()

			ginkgo.GinkgoWriter.Printf("Import error output: %s\n", output)
			ginkgo.GinkgoWriter.Printf("Import error: %v\n", err)

			// Verify command failed
			gomega.Expect(err).To(gomega.HaveOccurred())

			// loadConfig returns "invalid configuration: failed to read import config file ..." when the file is missing.
			gomega.Expect(err.Error()).To(gomega.Or(
				gomega.ContainSubstring("invalid configuration"),
				gomega.ContainSubstring("failed to read import config file"),
				gomega.ContainSubstring("no such file or directory"),
			))
		})
	})
})
