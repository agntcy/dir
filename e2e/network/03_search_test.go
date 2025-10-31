// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package network

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

// Test file dedicated to testing remote routing search functionality with OR logic and minMatchScore

// Package-level variables for cleanup (accessible by AfterSuite)
// CIDs are now tracked in network_suite_test.go

var _ = ginkgo.Describe("Running dirctl end-to-end tests for remote routing search with OR logic", func() {
	var cli *utils.CLI
	var cid string

	// Setup temp record file
	tempDir := os.Getenv("E2E_COMPILE_OUTPUT_DIR")
	if tempDir == "" {
		tempDir = os.TempDir()
	}
	tempPath := filepath.Join(tempDir, "record_v1alpha1_remote_search_test.json")

	// Create directory and write V1Alpha1 record data
	_ = os.MkdirAll(filepath.Dir(tempPath), 0o755)
	_ = os.WriteFile(tempPath, testdata.ExpectedRecordV070JSON, 0o600)

	ginkgo.BeforeEach(func() {
		if cfg.DeploymentMode != config.DeploymentModeNetwork {
			ginkgo.Skip("Skipping test, not in network mode")
		}

		// ✅ CRITICAL: Reset CLI state to prevent flag accumulation across test executions
		utils.ResetCLIState()

		// Initialize CLI helper
		cli = utils.NewCLI()
	})

	ginkgo.Context("setup for remote search testing", func() {
		ginkgo.It("should push record_070.json to peer 1", func() {
			cid = cli.Push(tempPath).WithArgs("--output", "raw").OnServer(utils.Peer1Addr).ShouldSucceed()

			// Track CID for cleanup
			RegisterCIDForCleanup(cid, "search")

			// Validate that the returned CID correctly represents the pushed data
			utils.LoadAndValidateCID(cid, tempPath)
		})

		ginkgo.It("should publish record_070.json to routing on peer 1 only", func() {
			// ONLY publish on Peer 1 - this creates the scenario:
			// - Peer 1: has record locally (published)
			// - Peer 2: will see it as remote via DHT
			cli.Routing().Publish(cid).OnServer(utils.Peer1Addr).ShouldSucceed()

			// Wait for DHT propagation (same timing as working network deploy test)
			time.Sleep(15 * time.Second)
			ginkgo.GinkgoWriter.Printf("Published CID to routing on Peer 1: %s", cid)
		})

		ginkgo.It("should verify setup - peer 1 has local record, peer 2 does not", func() {
			// Debug: Check local records on both peers
			peer1LocalRecords := cli.Routing().List().OnServer(utils.Peer1Addr).ShouldSucceed()
			ginkgo.GinkgoWriter.Printf("=== PEER 1 LOCAL RECORDS ===\n%s", peer1LocalRecords)

			peer2LocalRecords := cli.Routing().List().OnServer(utils.Peer2Addr).ShouldSucceed()
			ginkgo.GinkgoWriter.Printf("=== PEER 2 LOCAL RECORDS ===\n%s", peer2LocalRecords)

			// Peer 1 should have the record locally
			gomega.Expect(peer1LocalRecords).To(gomega.ContainSubstring(cid))

			// Peer 2 should NOT have the record locally (will see it as remote)
			gomega.Expect(peer2LocalRecords).To(gomega.ContainSubstring("No local records found"))
		})
	})

	ginkgo.Context("OR logic with minMatchScore tests", func() {
		ginkgo.It("should debug: test working pattern first (minScore=1)", func() {
			// First, let's replicate the WORKING test pattern from dirctl_network_deploy_test.go
			// This should work since the original test works
			output := cli.Routing().Search().
				WithSkill("natural_language_processing"). // Same as working test - should match via prefix
				WithMinScore(1).                          // Explicit minScore=1 (same as default)
				WithLimit(10).
				OnServer(utils.Peer2Addr). // Search FROM Peer 2 to find Peer 1's records
				ShouldSucceed()            // Don't wait - should be immediate since working test works

			ginkgo.GinkgoWriter.Printf("=== DEBUG: Working pattern with explicit minScore=1 ===\n%s", output)

			// Should find the record like the working test does
			gomega.Expect(output).To(gomega.ContainSubstring(cid))
			ginkgo.GinkgoWriter.Printf("✅ SUCCESS: Working pattern with explicit minScore=1 found record")
		})

		ginkgo.It("should debug: test exact skill matching (minScore=1)", func() {
			// Test exact skill matching with minScore=1
			output := cli.Routing().Search().
				WithSkill("natural_language_processing/natural_language_generation/text_completion"). // Exact match - should work
				WithMinScore(1).                                                                      // Only need 1 match
				WithLimit(10).
				OnServer(utils.Peer2Addr).
				ShouldSucceed()

			ginkgo.GinkgoWriter.Printf("=== DEBUG: Exact skill with minScore=1 ===\n%s", output)

			// Should find the record
			gomega.Expect(output).To(gomega.ContainSubstring(cid))
			ginkgo.GinkgoWriter.Printf("✅ SUCCESS: Exact skill matching with minScore=1 found record")
		})

		ginkgo.It("should debug: test two skills with minScore=2", func() {
			// Test two exact skills with minScore=2 (should match both and pass threshold)
			output := cli.Routing().Search().
				WithSkill("natural_language_processing/natural_language_generation/text_completion"). // Query 1 - ✅ should match
				WithSkill("natural_language_processing/analytical_reasoning/problem_solving").        // Query 2 - ✅ should match
				WithMinScore(2).                                                                      // Need both queries to match
				WithLimit(10).
				OnServer(utils.Peer2Addr).
				ShouldSucceed()

			ginkgo.GinkgoWriter.Printf("=== DEBUG: Two exact skills with minScore=2 ===\n%s", output)

			// Should find the record since both skills should match
			gomega.Expect(output).To(gomega.ContainSubstring(cid))
			ginkgo.GinkgoWriter.Printf("✅ SUCCESS: Two skills with minScore=2 found record")
		})

		ginkgo.It("should demonstrate OR logic success - minScore=2 finds record", func() {
			// Now test the full OR logic: 2 real skills + 1 fake skill, requiring minScore=2
			output := cli.Routing().Search().
				WithSkill("natural_language_processing/natural_language_generation/text_completion"). // Query 1 - ✅ should match
				WithSkill("natural_language_processing/analytical_reasoning/problem_solving").        // Query 2 - ✅ should match
				WithSkill("NonexistentSkill").                                                        // Query 3 - ❌ won't match
				WithMinScore(2).                                                                      // Only need 2/3 queries to match
				WithLimit(10).
				OnServer(utils.Peer2Addr).
				ShouldSucceed()

			ginkgo.GinkgoWriter.Printf("=== DEBUG: Full OR logic test (minScore=2) ===\n%s", output)

			// Should find the record since 2/3 queries match
			gomega.Expect(output).To(gomega.ContainSubstring(cid))
			ginkgo.GinkgoWriter.Printf("✅ SUCCESS: OR logic test found record with minScore=2 (2/3 queries matched)")
		})

		ginkgo.It("should demonstrate threshold filtering - minScore=3 filters out record", func() {
			// Test threshold filtering: same queries but higher minScore should find NO records
			// Same 2/3 queries match, but now we require minScore=3 (all queries must match)
			output := cli.Routing().Search().
				WithSkill("natural_language_processing/natural_language_generation/text_completion"). // Query 1 - ✅ should match
				WithSkill("natural_language_processing/analytical_reasoning/problem_solving").        // Query 2 - ✅ should match
				WithSkill("NonexistentSkill").                                                        // Query 3 - ❌ doesn't match
				WithMinScore(3).                                                                      // Require ALL 3 queries to match
				WithLimit(10).
				OnServer(utils.Peer2Addr). // Search FROM Peer 2 to find Peer 1's records
				ShouldSucceed()            // Should succeed but return "No remote records found"

			ginkgo.GinkgoWriter.Printf("=== THRESHOLD TEST RESULT (minScore=3) ===\n%s", output)

			// Should find NO records because minScore=3 but record only matches 2/3 queries
			gomega.Expect(output).To(gomega.ContainSubstring("No remote records found"))
			gomega.Expect(output).NotTo(gomega.ContainSubstring(cid)) // Should NOT contain the CID

			ginkgo.GinkgoWriter.Printf("✅ SUCCESS: Threshold filtering worked - no records found with minScore=3 (only 2/3 queries matched)")
		})

		ginkgo.It("should demonstrate single query match - minScore=1 finds record", func() {
			// Test with single query to verify basic functionality
			cli.Routing().Search().
				WithSkill("natural_language_processing/natural_language_generation/text_completion"). // Query 1 - ✅ should match
				WithMinScore(1).                                                                      // Only need 1 query to match
				WithLimit(10).
				OnServer(utils.Peer2Addr).                   // Search FROM Peer 2 to find Peer 1's records
				ShouldEventuallyContain(cid, 60*time.Second) // Shorter timeout since DHT is already propagated

			ginkgo.GinkgoWriter.Printf("✅ SUCCESS: Single query search found record with minScore=1")
		})

		ginkgo.It("should demonstrate all queries match - minScore=2 with 2 real queries", func() {
			// Test with 2 real queries that should both match, requiring both (minScore=2)
			cli.Routing().Search().
				WithSkill("natural_language_processing/natural_language_generation/text_completion"). // Query 1 - ✅ should match
				WithSkill("natural_language_processing/analytical_reasoning/problem_solving").        // Query 2 - ✅ should match
				WithMinScore(2).                                                                      // Need both queries to match
				WithLimit(10).
				OnServer(utils.Peer2Addr).                   // Search FROM Peer 2 to find Peer 1's records
				ShouldEventuallyContain(cid, 60*time.Second) // Shorter timeout since DHT is already propagated

			ginkgo.GinkgoWriter.Printf("✅ SUCCESS: All matching queries search found record with minScore=2")
		})
	})

	ginkgo.Context("edge case tests", func() {
		ginkgo.It("should handle minScore=0 (should default to minScore=1)", func() {
			// Test edge case: minScore=0 should default to minScore=1 per proto specification
			// Proto: "If not set, it will return records that match at least one query"
			output := cli.Routing().Search().
				WithSkill("natural_language_processing/natural_language_generation/text_completion"). // Query 1 - ✅ should match
				WithMinScore(0).                                                                      // Should default to 1
				WithLimit(10).
				WithArgs("--output", "json").
				OnServer(utils.Peer2Addr).
				ShouldSucceed()

			// With minScore=0 defaulting to 1, should find record since query matches
			gomega.Expect(output).To(gomega.ContainSubstring(cid))
			gomega.Expect(output).To(gomega.ContainSubstring("\"match_score\": 1"))

			ginkgo.GinkgoWriter.Printf("✅ SUCCESS: minScore=0 correctly defaults to minScore=1 per proto spec")
		})

		ginkgo.It("should handle empty queries with appropriate error", func() {
			// Test edge case: no queries should return helpful error message
			// This is the correct production behavior to prevent expensive full scans
			output := cli.Routing().Search().
				WithMinScore(0).
				WithLimit(10).
				OnServer(utils.Peer2Addr).
				ShouldSucceed() // Command succeeds but returns error message

			// Should get helpful error message, not crash or return all records
			gomega.Expect(output).To(gomega.ContainSubstring("No search criteria specified"))
			gomega.Expect(output).To(gomega.ContainSubstring("Use --skill, --locator, --domain, or --module flags"))

			ginkgo.GinkgoWriter.Printf("✅ SUCCESS: Empty queries properly rejected with helpful error message")

			// CLEANUP: This is the last test in this Describe block
			// Clean up search test records to ensure isolation from subsequent test files
			ginkgo.DeferCleanup(func() {
				CleanupNetworkRecords(remoteSearchTestCIDs, "search tests")
			})
		})
	})
})
