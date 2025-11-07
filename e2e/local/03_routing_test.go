// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"os"
	"path/filepath"
	"time"

	"github.com/agntcy/dir/e2e/shared/config"
	"github.com/agntcy/dir/e2e/shared/testdata"
	"github.com/agntcy/dir/e2e/shared/utils"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Running dirctl routing commands in local single node deployment", ginkgo.Ordered, func() {
	var cli *utils.CLI
	var cid string

	// Setup temp record file
	tempDir := os.Getenv("E2E_COMPILE_OUTPUT_DIR")
	if tempDir == "" {
		tempDir = os.TempDir()
	}
	tempPath := filepath.Join(tempDir, "record_v3_local_routing_test.json")

	// Create directory and write record data
	_ = os.MkdirAll(filepath.Dir(tempPath), 0o755)
	_ = os.WriteFile(tempPath, testdata.ExpectedRecordV070JSON, 0o600)

	ginkgo.BeforeEach(func() {
		if cfg.DeploymentMode != config.DeploymentModeLocal {
			ginkgo.Skip("Skipping test, not in local mode")
		}

		utils.ResetCLIState()
		// Initialize CLI helper
		cli = utils.NewCLI()
	})

	ginkgo.Context("routing publish command", func() {
		ginkgo.It("should push a record first (prerequisite for publish)", func() {
			cid = cli.Push(tempPath).WithArgs("--output", "raw").ShouldSucceed()

			// Validate that the returned CID correctly represents the pushed data
			utils.LoadAndValidateCID(cid, tempPath)
		})

		ginkgo.It("should publish a record to local routing", func() {
			output := cli.Routing().Publish(cid).ShouldSucceed()

			// Should confirm successful publishing
			gomega.Expect(output).To(gomega.ContainSubstring("Successfully submitted publication request"))
			gomega.Expect(output).To(gomega.ContainSubstring(cid))

			// Wait for publish operation to complete (publishing is asynchronous)
			time.Sleep(utils.PublishProcessingDelay)
		})

		ginkgo.It("should fail to publish non-existent record", func() {
			_ = cli.Routing().Publish("non-existent-cid").ShouldFail()
		})
	})

	ginkgo.Context("routing list command", func() {
		ginkgo.It("should list all local records without filters", func() {
			output := cli.Routing().List().ShouldSucceed()

			// Should show the published record
			gomega.Expect(output).To(gomega.ContainSubstring("Local records"))
			gomega.Expect(output).To(gomega.ContainSubstring(cid))
		})

		ginkgo.It("should list record by CID", func() {
			output := cli.Routing().List().WithCid(cid).ShouldSucceed()

			// Should find the specific record
			gomega.Expect(output).To(gomega.ContainSubstring("Local records"))
			gomega.Expect(output).To(gomega.ContainSubstring(cid))
		})

		ginkgo.It("should list records by skill filter", func() {
			output := cli.Routing().List().WithSkill("natural_language_processing/natural_language_generation/text_completion").ShouldSucceed()

			// Should find records with NLP skills
			gomega.Expect(output).To(gomega.ContainSubstring("Local records"))
			gomega.Expect(output).To(gomega.ContainSubstring(cid))
			gomega.Expect(output).To(gomega.ContainSubstring("/skills/natural_language_processing"))
		})

		ginkgo.It("should list records by specific skill", func() {
			output := cli.Routing().List().WithSkill("natural_language_processing/natural_language_generation/text_completion").ShouldSucceed()

			// Should find records with specific skill
			gomega.Expect(output).To(gomega.ContainSubstring("Local records"))
			gomega.Expect(output).To(gomega.ContainSubstring(cid))
		})

		ginkgo.It("should list records by locator filter", func() {
			output := cli.Routing().List().WithLocator("docker_image").ShouldSucceed()

			// Should find records with docker-image locator
			gomega.Expect(output).To(gomega.ContainSubstring("Local records"))
			gomega.Expect(output).To(gomega.ContainSubstring(cid))
		})

		ginkgo.It("should list records with multiple filters (AND logic)", func() {
			output := cli.Routing().List().
				WithSkill("natural_language_processing/natural_language_generation/text_completion").
				WithLocator("docker_image").
				ShouldSucceed()

			// Should find records matching both criteria
			gomega.Expect(output).To(gomega.ContainSubstring("Local records"))
			gomega.Expect(output).To(gomega.ContainSubstring(cid))
		})

		ginkgo.It("should return empty results for non-matching skill", func() {
			output := cli.Routing().List().WithSkill("NonExistentSkill").ShouldSucceed()

			// Should not find any records
			gomega.Expect(output).NotTo(gomega.ContainSubstring(cid))
			gomega.Expect(output).To(gomega.ContainSubstring("No local records found"))
		})

		ginkgo.It("should return empty results for non-existent CID", func() {
			output := cli.Routing().List().WithCid("non-existent-cid").ShouldSucceed()

			// Should show helpful message about using search for network discovery
			gomega.Expect(output).To(gomega.ContainSubstring("No local records found"))
		})

		ginkgo.It("should respect limit parameter", func() {
			output := cli.Routing().List().WithLimit(1).ShouldSucceed()

			// Should limit results (though we only have one record anyway)
			gomega.Expect(output).To(gomega.ContainSubstring("Local records"))
		})
	})

	ginkgo.Context("routing search command", func() {
		ginkgo.It("should search for local records (but return empty in local mode)", func() {
			output := cli.Routing().Search().WithSkill("Natural Language Processing").ShouldSucceed()

			// In local single-node mode, search should find no remote records
			// since there are no other peers
			gomega.Expect(output).To(gomega.ContainSubstring("No remote records found"))
		})

		ginkgo.It("should handle search with multiple criteria", func() {
			output := cli.Routing().Search().
				WithSkill("Natural Language Processing").
				WithLocator("docker-image").
				WithLimit(10).
				WithMinScore(2).
				ShouldSucceed()

			// Should complete without error, but find no remote records in local mode
			gomega.Expect(output).To(gomega.ContainSubstring("No remote records found"))
		})

		ginkgo.It("should handle OR logic search with partial query matching", func() {
			// Test OR logic: 3 queries but only require 2 matches (minScore=2)
			// This demonstrates the new OR behavior where records matching ≥2 queries are returned
			output := cli.Routing().Search().
				WithSkill("Natural Language Processing/Text Completion"). // Query 1 - might match
				WithSkill("Natural Language Processing/Problem Solving"). // Query 2 - might match
				WithSkill("NonexistentSkill").                            // Query 3 - won't match
				WithLimit(10).
				WithMinScore(2). // Only need 2/3 queries to match
				ShouldSucceed()

			// Should complete without error, but find no remote records in local mode
			// In network mode with remote records, this would find records matching
			// "Natural Language Processing/Text Completion" + "Natural Language Processing/Problem Solving" even without "NonexistentSkill"
			gomega.Expect(output).To(gomega.ContainSubstring("No remote records found"))

			// Verify the command structure was parsed correctly for OR logic
			gomega.Expect(output).NotTo(gomega.ContainSubstring("error"))
		})
	})

	ginkgo.Context("routing info command", func() {
		ginkgo.It("should show routing statistics for local records", func() {
			output := cli.Routing().Info().ShouldSucceed()

			// Should show local routing summary
			gomega.Expect(output).To(gomega.ContainSubstring("Local Routing Summary"))
			gomega.Expect(output).To(gomega.ContainSubstring("Total Records: 1"))
			gomega.Expect(output).To(gomega.ContainSubstring("Skills Distribution"))
			gomega.Expect(output).To(gomega.ContainSubstring("natural_language_processing/natural_language_generation/text_completion"))
		})

		ginkgo.It("should show helpful tips in routing info", func() {
			output := cli.Routing().Info().ShouldSucceed()

			// Should provide helpful usage tips
			gomega.Expect(output).To(gomega.ContainSubstring("Tips"))
			gomega.Expect(output).To(gomega.ContainSubstring("routing list --skill"))
			gomega.Expect(output).To(gomega.ContainSubstring("routing search --skill"))
		})
	})

	ginkgo.Context("routing unpublish command", func() {
		ginkgo.It("should unpublish a previously published record", func() {
			output := cli.Routing().Unpublish(cid).ShouldSucceed()

			// Should confirm successful unpublishing
			gomega.Expect(output).To(gomega.ContainSubstring("Successfully unpublished"))
			gomega.Expect(output).To(gomega.ContainSubstring(cid))
		})

		ginkgo.It("should fail to unpublish non-existent record", func() {
			_ = cli.Routing().Unpublish("non-existent-cid").ShouldFail()
		})

		ginkgo.It("should not find unpublished record in local list", func() {
			// After unpublishing, the record should not appear in local routing
			output := cli.Routing().List().WithCid(cid).ShouldSucceed()

			// Should not find the unpublished record
			gomega.Expect(output).To(gomega.ContainSubstring("No local records found"))
		})
	})

	ginkgo.Context("routing command integration", func() {
		ginkgo.It("should show empty routing info after unpublish", func() {
			output := cli.Routing().Info().ShouldSucceed()

			// Should show no records after unpublishing
			gomega.Expect(output).To(gomega.ContainSubstring("Local Routing Summary"))
			gomega.Expect(output).To(gomega.ContainSubstring("Total Records: 0"))
		})

		ginkgo.It("should validate routing command help", func() {
			output := cli.Routing().WithArgs("--help").ShouldSucceed()

			// Should show all routing subcommands
			gomega.Expect(output).To(gomega.ContainSubstring("publish"))
			gomega.Expect(output).To(gomega.ContainSubstring("unpublish"))
			gomega.Expect(output).To(gomega.ContainSubstring("list"))
			gomega.Expect(output).To(gomega.ContainSubstring("search"))
			gomega.Expect(output).To(gomega.ContainSubstring("info"))
		})
	})
})
