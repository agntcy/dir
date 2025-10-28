// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package network

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/agntcy/dir/e2e/shared/config"
	"github.com/agntcy/dir/e2e/shared/testdata"
	"github.com/agntcy/dir/e2e/shared/utils"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

// Test file dedicated to testing GossipSub label announcement functionality.
// This verifies that labels are efficiently propagated via GossipSub mesh to ALL subscribed peers.

// Package-level variables for cleanup (accessible by AfterSuite)
// CIDs are now tracked in network_suite_test.go

var _ = ginkgo.Describe("Running GossipSub label announcement tests", ginkgo.Ordered, func() {
	var cli *utils.CLI
	var cid string

	// Setup temp record file
	tempDir := os.Getenv("E2E_COMPILE_OUTPUT_DIR")
	if tempDir == "" {
		tempDir = os.TempDir()
	}
	tempPath := filepath.Join(tempDir, "record_v070_gossipsub_test.json")

	// Create directory and write record data
	_ = os.MkdirAll(filepath.Dir(tempPath), 0o755)
	_ = os.WriteFile(tempPath, testdata.ExpectedRecordV070JSON, 0o600)

	ginkgo.BeforeEach(func() {
		if cfg.DeploymentMode != config.DeploymentModeNetwork {
			ginkgo.Skip("Skipping test, not in network mode")
		}

		// Reset CLI state to ensure clean test environment
		utils.ResetCLIState()

		// Initialize CLI helper
		cli = utils.NewCLI()
	})

	ginkgo.Context("GossipSub wide propagation to all peers", func() {
		ginkgo.It("should push record_v070.json to peer 1", func() {
			cid = cli.Push(tempPath).WithArgs("--output", "raw").OnServer(utils.Peer1Addr).ShouldSucceed()

			// Track CID for cleanup
			RegisterCIDForCleanup(cid, "gossipsub")

			// Validate that the returned CID correctly represents the pushed data
			utils.LoadAndValidateCID(cid, tempPath)
		})

		ginkgo.It("should publish record to routing on peer 1", func() {
			// Publish triggers both DHT.Provide() and GossipSub.PublishLabels()
			cli.Routing().Publish(cid).OnServer(utils.Peer1Addr).ShouldSucceed()

			ginkgo.GinkgoWriter.Printf("Published CID to routing with GossipSub: %s", cid)
		})

		ginkgo.It("should propagate labels via GossipSub to all subscribed peers", func() {
			// GossipSub propagates much faster than DHT alone
			// Expected: ~5 seconds vs 15 seconds for DHT-only propagation
			ginkgo.GinkgoWriter.Printf("Waiting 5 seconds for GossipSub label propagation...")
			time.Sleep(5 * time.Second)

			// Verify Peer2 received labels via GossipSub
			ginkgo.GinkgoWriter.Printf("Testing label discovery on Peer2...")
			utils.ResetCLIState()
			output2 := cli.Routing().Search().
				WithSkill("natural_language_processing").
				WithLimit(10).
				OnServer(utils.Peer2Addr).
				ShouldSucceed()

			gomega.Expect(output2).To(gomega.ContainSubstring(cid))
			ginkgo.GinkgoWriter.Printf("✅ Peer2 discovered labels via GossipSub")

			// Verify Peer3 also received labels via GossipSub
			ginkgo.GinkgoWriter.Printf("Testing label discovery on Peer3...")
			utils.ResetCLIState()
			output3 := cli.Routing().Search().
				WithSkill("natural_language_processing").
				WithLimit(10).
				OnServer(utils.Peer3Addr).
				ShouldSucceed()

			gomega.Expect(output3).To(gomega.ContainSubstring(cid))
			ginkgo.GinkgoWriter.Printf("✅ Peer3 discovered labels via GossipSub")

			ginkgo.GinkgoWriter.Printf("✅ SUCCESS: GossipSub propagated labels to ALL 3 peers (not just k-closest)")
		})

		ginkgo.It("should verify labels are discoverable from both remote peers", func() {
			// Additional verification with different skill query
			utils.ResetCLIState()
			output2 := cli.Routing().Search().
				WithSkill("natural_language_processing/natural_language_generation/text_completion").
				OnServer(utils.Peer2Addr).
				ShouldSucceed()

			gomega.Expect(output2).To(gomega.ContainSubstring(cid))
			gomega.Expect(output2).To(gomega.ContainSubstring("match_score"))

			utils.ResetCLIState()
			output3 := cli.Routing().Search().
				WithSkill("natural_language_processing/analytical_reasoning/problem_solving").
				OnServer(utils.Peer3Addr).
				ShouldSucceed()

			gomega.Expect(output3).To(gomega.ContainSubstring(cid))
			gomega.Expect(output3).To(gomega.ContainSubstring("match_score"))

			ginkgo.GinkgoWriter.Printf("✅ Both peers can search with specific skill queries")
		})
	})

	ginkgo.Context("GossipSub performance and timing", func() {
		var perfCID string
		var perfPath string

		ginkgo.BeforeAll(func() {
			// Setup separate record for performance testing
			perfPath = filepath.Join(tempDir, "record_v070_gossipsub_perf_test.json")
			_ = os.WriteFile(perfPath, testdata.ExpectedRecordV070JSON, 0o600)
		})

		ginkgo.It("should push performance test record to peer 1", func() {
			perfCID = cli.Push(perfPath).WithArgs("--output", "raw").OnServer(utils.Peer1Addr).ShouldSucceed()
			RegisterCIDForCleanup(perfCID, "gossipsub")
		})

		ginkgo.It("should discover labels in under 7 seconds via GossipSub", func() {
			// Publish the record
			cli.Routing().Publish(perfCID).OnServer(utils.Peer1Addr).ShouldSucceed()

			startTime := time.Now()
			ginkgo.GinkgoWriter.Printf("Starting timing test at %s", startTime.Format("15:04:05"))

			// Poll for label discovery with short intervals
			// GossipSub should propagate in ~2-5 seconds
			utils.ResetCLIState()
			output := cli.Routing().Search().
				WithSkill("natural_language_processing").
				OnServer(utils.Peer2Addr).
				ShouldEventuallyContain(perfCID, 10*time.Second) // Max 10s timeout

			discoveryTime := time.Since(startTime)
			ginkgo.GinkgoWriter.Printf("✅ Labels discovered in %v", discoveryTime)

			// Verify it's faster than baseline DHT propagation (15s)
			gomega.Expect(discoveryTime).To(gomega.BeNumerically("<", 7*time.Second),
				"GossipSub should propagate faster than DHT-only baseline")

			gomega.Expect(output).To(gomega.ContainSubstring(perfCID))
		})
	})

	ginkgo.Context("GossipSub bulk record propagation", func() {
		var bulkCIDs []string
		var bulkPaths []string

		ginkgo.BeforeAll(func() {
			// Prepare 5 test records for bulk testing
			// Note: Reusing same record content but treating as separate for propagation test
			bulkPaths = make([]string, 5)
			for i := range 5 {
				bulkPaths[i] = filepath.Join(tempDir, fmt.Sprintf("record_v070_gossipsub_bulk_%d_test.json", i))
				_ = os.WriteFile(bulkPaths[i], testdata.ExpectedRecordV070JSON, 0o600)
			}
		})

		ginkgo.It("should push 5 records to peer 1", func() {
			bulkCIDs = make([]string, 5)
			for i, path := range bulkPaths {
				cid := cli.Push(path).WithArgs("--output", "raw").OnServer(utils.Peer1Addr).ShouldSucceed()
				bulkCIDs[i] = cid
				RegisterCIDForCleanup(cid, "gossipsub")
				ginkgo.GinkgoWriter.Printf("Pushed bulk record %d/%d: %s", i+1, 5, cid)
			}
		})

		ginkgo.It("should publish all 5 records sequentially", func() {
			for i, bulkCID := range bulkCIDs {
				cli.Routing().Publish(bulkCID).OnServer(utils.Peer1Addr).ShouldSucceed()
				ginkgo.GinkgoWriter.Printf("Published bulk record %d/%d via GossipSub", i+1, 5)
			}
		})

		ginkgo.It("should propagate all 5 records' labels via GossipSub", func() {
			// Verify all 5 records are discoverable from Peer2
			// Wait at least 10 seconds for GossipSub propagation of all announcements
			utils.ResetCLIState()
			successCount := 0
			for i, bulkCID := range bulkCIDs {
				cli.Routing().Search().
					WithSkill("natural_language_processing").
					WithLimit(10).
					OnServer(utils.Peer2Addr).
					ShouldEventuallyContain(bulkCID, 15*time.Second)

				successCount++
				ginkgo.GinkgoWriter.Printf("✅ Bulk record %d/%d discovered on Peer2", i+1, 5)
				utils.ResetCLIState()
			}

			// All 5 should be discoverable
			gomega.Expect(successCount).To(gomega.Equal(5),
				"All 5 records should be discoverable via GossipSub")

			ginkgo.GinkgoWriter.Printf("✅ SUCCESS: GossipSub propagated all 5 records efficiently")
		})

		ginkgo.It("should verify bulk records are also discoverable from peer 3", func() {
			// Verify propagation to Peer3 as well (proves mesh propagation)
			utils.ResetCLIState()
			successCount := 0
			for i, bulkCID := range bulkCIDs {
				cli.Routing().Search().
					WithSkill("natural_language_processing").
					WithLimit(10).
					OnServer(utils.Peer3Addr).
					ShouldEventuallyContain(bulkCID, 15*time.Second)

				successCount++
				ginkgo.GinkgoWriter.Printf("✅ Bulk record %d/%d discovered on Peer3", i+1, 5)
				utils.ResetCLIState()
			}

			gomega.Expect(successCount).To(gomega.Equal(5),
				"All 5 records should be discoverable on Peer3 via GossipSub")

			ginkgo.GinkgoWriter.Printf("✅ SUCCESS: GossipSub mesh propagated to all peers")
		})
	})

	ginkgo.Context("GossipSub edge cases and validation", func() {
		var edgeCID string

		ginkgo.It("should push edge case test record to peer 1", func() {
			edgePath := filepath.Join(tempDir, "record_v070_gossipsub_edge_test.json")
			_ = os.WriteFile(edgePath, testdata.ExpectedRecordV070JSON, 0o600)

			edgeCID = cli.Push(edgePath).WithArgs("--output", "raw").OnServer(utils.Peer1Addr).ShouldSucceed()
			RegisterCIDForCleanup(edgeCID, "gossipsub")
		})

		ginkgo.It("should handle search with multiple label types via GossipSub", func() {
			// Publish record
			cli.Routing().Publish(edgeCID).OnServer(utils.Peer1Addr).ShouldSucceed()

			// Wait for GossipSub propagation
			time.Sleep(5 * time.Second)

			// Test search with OR logic across multiple label types
			utils.ResetCLIState()
			output := cli.Routing().Search().
				WithSkill("natural_language_processing"). // Should match
				WithDomain("life_science").               // Should match (record has life_science/biotechnology)
				WithMinScore(2).                          // Both should match
				WithLimit(10).
				WithArgs("--output", "json").
				OnServer(utils.Peer2Addr).
				ShouldSucceed()

			gomega.Expect(output).To(gomega.ContainSubstring(edgeCID))
			gomega.Expect(output).To(gomega.ContainSubstring("\"match_score\": 2"))

			ginkgo.GinkgoWriter.Printf("✅ GossipSub propagates all label types correctly")
		})

		ginkgo.It("should verify labels persist across multiple searches", func() {
			// Test that cached labels from GossipSub remain available
			// This ensures the fallback to pull is NOT triggered on subsequent searches

			// First search
			utils.ResetCLIState()
			output1 := cli.Routing().Search().
				WithSkill("natural_language_processing").
				OnServer(utils.Peer2Addr).
				ShouldSucceed()
			gomega.Expect(output1).To(gomega.ContainSubstring(edgeCID))

			// Second search (should use cached labels, not pull again)
			utils.ResetCLIState()
			output2 := cli.Routing().Search().
				WithSkill("natural_language_processing/analytical_reasoning/problem_solving").
				OnServer(utils.Peer2Addr).
				ShouldSucceed()
			gomega.Expect(output2).To(gomega.ContainSubstring(edgeCID))

			// Third search with different peer
			utils.ResetCLIState()
			output3 := cli.Routing().Search().
				WithSkill("natural_language_processing/natural_language_generation").
				OnServer(utils.Peer3Addr).
				ShouldSucceed()
			gomega.Expect(output3).To(gomega.ContainSubstring(edgeCID))

			ginkgo.GinkgoWriter.Printf("✅ Cached labels from GossipSub persist across multiple searches")
		})
	})

	ginkgo.Context("GossipSub comparison with baseline", func() {
		ginkgo.It("should demonstrate faster propagation compared to DHT-only baseline", func() {
			// This test compares against the known baseline from 01_deploy_test.go
			// Baseline: 15 seconds wait for DHT propagation
			// GossipSub: Should work in ~5 seconds

			baselinePath := filepath.Join(tempDir, "record_v070_gossipsub_baseline_test.json")
			_ = os.WriteFile(baselinePath, testdata.ExpectedRecordV070JSON, 0o600)

			baselineCID := cli.Push(baselinePath).WithArgs("--output", "raw").OnServer(utils.Peer1Addr).ShouldSucceed()
			RegisterCIDForCleanup(baselineCID, "gossipsub")

			// Publish and start timing
			cli.Routing().Publish(baselineCID).OnServer(utils.Peer1Addr).ShouldSucceed()
			startTime := time.Now()

			// Poll for discovery with 1-second intervals
			ginkgo.GinkgoWriter.Printf("Polling for label discovery (max 10 seconds)...")
			utils.ResetCLIState()

			found := false
			maxAttempts := 10
			for attempt := 1; attempt <= maxAttempts; attempt++ {
				output, err := cli.Routing().Search().
					WithSkill("natural_language_processing").
					WithLimit(10).
					OnServer(utils.Peer2Addr).
					Execute()

				if err == nil && strings.Contains(output, baselineCID) {
					discoveryTime := time.Since(startTime)
					ginkgo.GinkgoWriter.Printf("✅ Labels discovered in %v (attempt %d/%d)", discoveryTime, attempt, maxAttempts)
					found = true

					// Verify it's faster than DHT baseline
					gomega.Expect(discoveryTime).To(gomega.BeNumerically("<", 7*time.Second),
						"GossipSub should be significantly faster than DHT-only baseline (15s)")

					break
				}

				time.Sleep(1 * time.Second)
				utils.ResetCLIState()
			}

			gomega.Expect(found).To(gomega.BeTrue(), "Labels should be discovered within 10 seconds via GossipSub")

			// CLEANUP: This is the last test in this Describe block
			ginkgo.DeferCleanup(func() {
				CleanupNetworkRecords(gossipsubTestCIDs, "gossipsub tests")
			})
		})
	})
})
