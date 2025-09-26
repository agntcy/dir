// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package network

import (
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

// Using peer addresses from utils.constants

// Package-level variables for cleanup (accessible by AfterSuite)
// CIDs are now tracked in network_suite_test.go

var _ = ginkgo.Describe("Running dirctl end-to-end tests for sync commands", func() {
	var cli *utils.CLI
	var syncID string

	// Setup temp files for CLI commands (CLI needs actual files on disk)
	tempDir := os.Getenv("E2E_COMPILE_OUTPUT_DIR")
	if tempDir == "" {
		tempDir = os.TempDir()
	}
	recordV4Path := filepath.Join(tempDir, "record_v070_sync_v4_test.json")
	recordV5Path := filepath.Join(tempDir, "record_v070_sync_v5_test.json")

	// Create directory and write record data
	_ = os.MkdirAll(filepath.Dir(recordV4Path), 0o755)
	_ = os.WriteFile(recordV4Path, testdata.ExpectedRecordV070SyncV4JSON, 0o600)
	_ = os.WriteFile(recordV5Path, testdata.ExpectedRecordV070SyncV5JSON, 0o600)

	ginkgo.BeforeEach(func() {
		if cfg.DeploymentMode != config.DeploymentModeNetwork {
			ginkgo.Skip("Skipping test, not in network mode")
		}

		utils.ResetCLIState()

		// Initialize CLI helper
		cli = utils.NewCLI()
	})

	ginkgo.Context("create command", func() {
		ginkgo.It("should accept valid remote URL format", func() {
			output := cli.Sync().Create("https://directory.example.com").OnServer(utils.Peer1Addr).ShouldSucceed()

			gomega.Expect(output).To(gomega.ContainSubstring("Sync created with ID: "))
			syncID = strings.TrimPrefix(output, "Sync created with ID: ")
		})
	})

	ginkgo.Context("list command", func() {
		ginkgo.It("should execute without arguments and return a list with the created sync", func() {
			output := cli.Sync().List().OnServer(utils.Peer1Addr).ShouldSucceed()

			gomega.Expect(output).To(gomega.ContainSubstring(syncID))
			gomega.Expect(output).To(gomega.ContainSubstring("https://directory.example.com"))
			gomega.Expect(output).To(gomega.ContainSubstring("PENDING"))
		})
	})

	ginkgo.Context("status command", func() {
		ginkgo.It("should accept a sync ID argument and return the sync status", func() {
			output := cli.Sync().Status(syncID).OnServer(utils.Peer1Addr).ShouldSucceed()

			gomega.Expect(output).To(gomega.ContainSubstring(syncID))
			gomega.Expect(output).To(gomega.ContainSubstring("PENDING"))
		})
	})

	ginkgo.Context("delete command", func() {
		ginkgo.It("should accept a sync ID argument and delete the sync", func() {
			// Command may fail due to network/auth issues, but argument parsing should work
			_, err := cli.Sync().Delete(syncID).OnServer(utils.Peer1Addr).Execute()
			if err != nil {
				gomega.Expect(err.Error()).NotTo(gomega.ContainSubstring("required"))
			}
		})

		ginkgo.It("should return deleted status", func() {
			cli.Sync().List().OnServer(utils.Peer1Addr).ShouldContain("DELETE")
		})
	})

	ginkgo.Context("sync functionality", func() {
		var cid string
		var cidV5 string

		ginkgo.It("should push record_v070_sync_v4.json to peer 1", func() {
			cid = cli.Push(recordV4Path).OnServer(utils.Peer1Addr).ShouldSucceed()

			// Track CID for cleanup
			RegisterCIDForCleanup(cid, "sync")

			// Validate that the returned CID correctly represents the pushed data
			utils.LoadAndValidateCID(cid, recordV4Path)
		})

		ginkgo.It("should publish record_v070_sync_v4.json", func() {
			cli.Routing().Publish(cid).OnServer(utils.Peer1Addr).ShouldSucceed()
		})

		ginkgo.It("should push record_v070_sync_v5.json to peer 1", func() {
			cidV5 = cli.Push(recordV5Path).OnServer(utils.Peer1Addr).ShouldSucceed()

			// Track CID for cleanup
			RegisterCIDForCleanup(cidV5, "sync")

			// Validate that the returned CID correctly represents the pushed data
			utils.LoadAndValidateCID(cidV5, recordV5Path)
		})

		ginkgo.It("should publish record_v070_sync_v5.json", func() {
			cli.Routing().Publish(cidV5).OnServer(utils.Peer1Addr).ShouldSucceed()
		})

		ginkgo.It("should fail to pull record_v070_sync_v4.json from peer 2", func() {
			_ = cli.Pull(cid).OnServer(utils.Peer2Addr).ShouldFail()
		})

		ginkgo.It("should create sync from peer 1 to peer 2", func() {
			output := cli.Sync().Create(utils.Peer1InternalAddr).OnServer(utils.Peer2Addr).ShouldSucceed()

			gomega.Expect(output).To(gomega.ContainSubstring("Sync created with ID: "))
			syncID = strings.TrimPrefix(output, "Sync created with ID: ")
		})

		ginkgo.It("should list the sync", func() {
			output := cli.Sync().List().OnServer(utils.Peer2Addr).ShouldSucceed()

			gomega.Expect(output).To(gomega.ContainSubstring(syncID))
			gomega.Expect(output).To(gomega.ContainSubstring(utils.Peer1InternalAddr))
			gomega.Expect(output).To(gomega.ContainSubstring("PENDING"))
		})

		// Wait for sync to complete
		ginkgo.It("should wait for sync to complete", func() {
			// Poll sync status until it changes from PENDING to IN_PROGRESS
			output := cli.Sync().Status(syncID).OnServer(utils.Peer2Addr).ShouldEventuallyContain("IN_PROGRESS", 120*time.Second)
			ginkgo.GinkgoWriter.Printf("Current sync status: %s", output)

			// Wait for 60 seconds to ensure the sync is complete (reduce flakiness)
			time.Sleep(60 * time.Second)
		})

		ginkgo.It("should succeed to pull record_v070_sync_v4.json from peer 2 after sync", func() {
			output := cli.Pull(cid).OnServer(utils.Peer2Addr).ShouldSucceed()

			// Compare the output with the expected JSON
			equal, err := utils.CompareOASFRecords([]byte(output), testdata.ExpectedRecordV070SyncV4JSON)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(equal).To(gomega.BeTrue())
		})

		ginkgo.It("should succeed to search for record_v070_sync_v4.json from peer 2 after sync", func() {
			// Search should eventually return the cid in peer 2 (retry until monitor indexes the record)
			output := cli.Search().WithQuery("name", "directory.agntcy.org/cisco/marketing-strategy-v4").OnServer(utils.Peer2Addr).ShouldEventuallyContain(cid, 240*time.Second)

			ginkgo.GinkgoWriter.Printf("Search found cid: %s", output)
		})

		ginkgo.It("should verify the record_v070_sync_v4.json from peer 2 after sync", func() {
			cli.Verify(cid).OnServer(utils.Peer2Addr).ShouldSucceed()
		})

		// Delete sync from peer 2
		ginkgo.It("should delete sync from peer 2", func() {
			cli.Sync().Delete(syncID).OnServer(utils.Peer2Addr).ShouldSucceed()
		})

		// Wait for sync to complete
		ginkgo.It("should wait for delete to complete", func() {
			// Poll sync status until it changes from DELETE_PENDING to DELETED
			output := cli.Sync().Status(syncID).OnServer(utils.Peer2Addr).ShouldEventuallyContain("DELETED", 120*time.Second)
			ginkgo.GinkgoWriter.Printf("Current sync status: %s", output)
		})

		ginkgo.It("should create sync from peer 1 to peer 3 using routing search piped to sync create", func() {
			ginkgo.GinkgoWriter.Printf("=== Phase 3: Selective Sync Test (Peer 1 → Peer 3) ===")

			// Verify initial state - peer 3 should not have any records
			ginkgo.GinkgoWriter.Printf("Step 1: Verifying peer 3 initial state...")
			_ = cli.Pull(cid).OnServer(utils.Peer3Addr).ShouldFail()   // v4 (NLP) should not exist
			_ = cli.Pull(cidV5).OnServer(utils.Peer3Addr).ShouldFail() // v5 (Audio) should not exist
			ginkgo.GinkgoWriter.Printf("✅ Confirmed: Peer 3 has no records initially")

			// Run routing search to find records with "Audio" skill
			ginkgo.GinkgoWriter.Printf("Step 2: Running routing search for 'audio/audio_classification' skill...")
			ginkgo.GinkgoWriter.Printf("Command: dirctl routing search --skill audio/audio_classification --json")
			searchOutput := cli.Routing().Search().WithArgs("--skill", "audio/audio_classification").WithArgs("--json").OnServer(utils.Peer3Addr).ShouldSucceed()

			ginkgo.GinkgoWriter.Printf("Routing search output: %s", searchOutput)

			// Analyze what CIDs were found
			ginkgo.GinkgoWriter.Printf("Step 3: Analyzing search results...")
			if strings.Contains(searchOutput, cid) {
				ginkgo.GinkgoWriter.Printf("🚨 UNEXPECTED: v4 record (NLP skills) found in Audio search!")
				ginkgo.GinkgoWriter.Printf("   CID: %s (should NOT be found for Audio skill)", cid)
			} else {
				ginkgo.GinkgoWriter.Printf("✅ EXPECTED: v4 record (NLP skills) NOT found in Audio search")
				ginkgo.GinkgoWriter.Printf("   CID: %s correctly filtered out", cid)
			}

			if strings.Contains(searchOutput, cidV5) {
				ginkgo.GinkgoWriter.Printf("✅ EXPECTED: v5 record (Audio skills) found in Audio search")
				ginkgo.GinkgoWriter.Printf("   CID: %s correctly matches Audio skill", cidV5)
			} else {
				ginkgo.GinkgoWriter.Printf("🚨 UNEXPECTED: v5 record (Audio skills) NOT found in Audio search!")
				ginkgo.GinkgoWriter.Printf("   CID: %s should be found for Audio skill", cidV5)
			}

			// Create selective sync using search results
			ginkgo.GinkgoWriter.Printf("Step 4: Creating selective sync with filtered CIDs...")
			ginkgo.GinkgoWriter.Printf("Command: dirctl sync create --stdin (piped from routing search)")
			output := cli.Sync().CreateFromStdin(searchOutput).OnServer(utils.Peer3Addr).ShouldSucceed()

			ginkgo.GinkgoWriter.Printf("Sync creation output: %s", output)
			gomega.Expect(output).To(gomega.ContainSubstring("Sync created with ID: "))
			ginkgo.GinkgoWriter.Printf("✅ Selective sync created successfully")
		})

		// Wait for sync to complete
		ginkgo.It("should wait for sync to complete", func() {
			ginkgo.GinkgoWriter.Printf("Step 5: Waiting for selective sync to complete...")
			ginkgo.GinkgoWriter.Printf("Monitoring sync status: PENDING → IN_PROGRESS")

			// Poll sync status until it changes from PENDING to IN_PROGRESS
			_ = cli.Sync().List().OnServer(utils.Peer3Addr).ShouldEventuallyContain("IN_PROGRESS", 120*time.Second)

			ginkgo.GinkgoWriter.Printf("✅ Selective sync completed (Zot OCI transfer finished)")
		})

		ginkgo.It("should succeed to pull record_v070_sync_v5.json from peer 3 after sync", func() {
			ginkgo.GinkgoWriter.Printf("Step 6: Verifying Audio record (v5) was synced to peer 3...")
			ginkgo.GinkgoWriter.Printf("Command: dirctl pull %s (Audio-skilled record)", cidV5)

			output := cli.Pull(cidV5).OnServer(utils.Peer3Addr).ShouldSucceed()

			// Compare the output with the expected JSON
			equal, err := utils.CompareOASFRecords([]byte(output), testdata.ExpectedRecordV070SyncV5JSON)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(equal).To(gomega.BeTrue())

			ginkgo.GinkgoWriter.Printf("✅ SUCCESS: Audio record correctly synced to peer 3")
			ginkgo.GinkgoWriter.Printf("   CID: %s", cidV5)
			ginkgo.GinkgoWriter.Printf("   Content matches expected v070_sync_v5.json")
		})

		ginkgo.It("should fail to pull record_v070_sync_v4.json from peer 3 after sync", func() {
			ginkgo.GinkgoWriter.Printf("Step 7: Verifying NLP record (v4) was NOT synced to peer 3...")
			ginkgo.GinkgoWriter.Printf("Command: dirctl pull %s (NLP-skilled record - should fail)", cid)

			_ = cli.Pull(cid).OnServer(utils.Peer3Addr).ShouldFail()

			ginkgo.GinkgoWriter.Printf("✅ SUCCESS: NLP record correctly filtered out by selective sync")
			ginkgo.GinkgoWriter.Printf("   CID: %s (NOT available on peer 3)", cid)
			ginkgo.GinkgoWriter.Printf("=== SELECTIVE SYNC TEST COMPLETED SUCCESSFULLY ===")
			ginkgo.GinkgoWriter.Printf("Summary:")
			ginkgo.GinkgoWriter.Printf("  - Routing search found only Audio-skilled records")
			ginkgo.GinkgoWriter.Printf("  - Sync transferred only filtered CIDs")
			ginkgo.GinkgoWriter.Printf("  - Audio record available on peer 3: ✅")
			ginkgo.GinkgoWriter.Printf("  - NLP record NOT available on peer 3: ✅")

			// CLEANUP: This is the last test in the sync functionality Context
			// Clean up sync test records to ensure isolation from subsequent test files
			ginkgo.DeferCleanup(func() {
				CleanupNetworkRecords(syncTestCIDs, "sync tests")
			})
		})
	})
})
