// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"encoding/json"

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

	ginkgo.Context("MCP registry import functionality", ginkgo.Ordered, func() {
		ginkgo.It("should successfully import records from MCP registry with limit", func() {
			// Run import command with a limit of 10 records
			output := cli.Command("import").
				WithArgs("--type=mcp", "--url=https://registry.modelcontextprotocol.io/v0.1", "--limit", "10").
				ShouldSucceed()

			ginkgo.GinkgoWriter.Printf("Import output: %s\n", output)

			// Verify output indicates successful import
			gomega.Expect(output).NotTo(gomega.BeEmpty())
			gomega.Expect(output).To(gomega.ContainSubstring("Total records:   10"))
			gomega.Expect(output).To(gomega.ContainSubstring("Imported:        10"))
			gomega.Expect(output).To(gomega.ContainSubstring("Failed:          0"))
		})

		ginkgo.It("should accept enrichment flags without errors", func() {
			// Run import help command to verify the enrichment flags exist
			// We just want to verify the flags are recognized by the CLI
			output := cli.Command("import").
				WithArgs("--help").
				ShouldSucceed()

			ginkgo.GinkgoWriter.Printf("Import help output: %s\n", output)

			// Verify the new enrichment flags are documented
			gomega.Expect(output).To(gomega.ContainSubstring("--enrich-skills-prompt"))
			gomega.Expect(output).To(gomega.ContainSubstring("--enrich-domains-prompt"))
			gomega.Expect(output).To(gomega.ContainSubstring("--enrich-config"))
		})

		ginkgo.It("should accept force and debug flags", func() {
			// Test that --force flag is accepted
			output := cli.Command("import").
				WithArgs("--type=mcp", "--url=https://registry.modelcontextprotocol.io/v0.1", "--limit", "2", "--force").
				ShouldSucceed()

			ginkgo.GinkgoWriter.Printf("Import with --force flag: %s\n", output)

			gomega.Expect(output).To(gomega.ContainSubstring("Total records:   2"))
			gomega.Expect(output).To(gomega.ContainSubstring("Imported:        2"))

			// Test that --debug flag is accepted and runs without error
			output2 := cli.Command("import").
				WithArgs("--type=mcp", "--url=https://registry.modelcontextprotocol.io/v0.1", "--limit", "1", "--debug").
				ShouldSucceed()

			ginkgo.GinkgoWriter.Printf("Import with --debug flag: %s\n", output2)

			// Just verify the command succeeds with debug flag
			gomega.Expect(output2).To(gomega.ContainSubstring("Total records:"))
		})

		var recordRefs []string

		ginkgo.It("should find at least 10 imported MCP records", func() {
			// Search for records with integration/mcp module
			output := cli.Search().
				WithModule("integration/mcp").
				WithLimit(20).
				WithArgs("--output", "json").
				ShouldSucceed()

			ginkgo.GinkgoWriter.Printf("Search output: %s\n", output)

			// Parse the output
			err := json.Unmarshal([]byte(output), &recordRefs)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Verify we have at least 10 records (from all previous import tests)
			gomega.Expect(len(recordRefs)).To(gomega.BeNumerically(">=", 10),
				"Expected at least 10 imported MCP records, got %d", len(recordRefs))
		})

		ginkgo.It("should be able to pull an imported record", func() {
			// Try to pull the record
			pullOutput := cli.Pull(recordRefs[0]).WithArgs("--output", "json").ShouldSucceed()
			gomega.Expect(pullOutput).NotTo(gomega.BeEmpty())

			// Verify the pulled record has expected fields
			var record map[string]interface{}
			err := json.Unmarshal([]byte(pullOutput), &record)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Verify essential fields exist
			gomega.Expect(record).To(gomega.HaveKey("name"))
			gomega.Expect(record).To(gomega.HaveKey("version"))
			gomega.Expect(record).To(gomega.HaveKey("schema_version"))
			gomega.Expect(record).To(gomega.HaveKey("locators"))
		})
	})
})
