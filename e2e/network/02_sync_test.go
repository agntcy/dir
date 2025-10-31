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
	var privateKeyPath string
	var tempKeyDir string

	// Setup temp files for CLI commands (CLI needs actual files on disk)
	tempDir := os.Getenv("E2E_COMPILE_OUTPUT_DIR")
	if tempDir == "" {
		tempDir = os.TempDir()
	}
	recordV4Path := filepath.Join(tempDir, "record_070_sync_v4_test.json")
	recordV5Path := filepath.Join(tempDir, "record_070_sync_v5_test.json")

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
		})
	})

	ginkgo.Context("status command", func() {
		ginkgo.It("should accept a sync ID argument and return the sync status", func() {
			output := cli.Sync().Status(syncID).OnServer(utils.Peer1Addr).ShouldSucceed()

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
			cli.Sync().Status(syncID).OnServer(utils.Peer1Addr).ShouldContain("DELETE")
		})
	})

	ginkgo.Context("sync functionality", ginkgo.Ordered, func() {
		var cid string
		var cidV5 string

		// Setup cosign key pair for signing tests
		ginkgo.BeforeAll(func() {
			var err error
			tempKeyDir, err = os.MkdirTemp("", "sync-test-keys")
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Generate cosign key pair
			utils.GenerateCosignKeyPair(tempKeyDir)
			privateKeyPath = filepath.Join(tempKeyDir, "cosign.key")

			// Verify key file was created
			gomega.Expect(privateKeyPath).To(gomega.BeAnExistingFile())

			// Set cosign password for signing
			err = os.Setenv("COSIGN_PASSWORD", utils.TestPassword)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})

		// Cleanup cosign keys after all tests
		ginkgo.AfterAll(func() {
			os.Unsetenv("COSIGN_PASSWORD")
			if tempKeyDir != "" {
				err := os.RemoveAll(tempKeyDir)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
			}
		})

		ginkgo.It("should push record_070_sync_v4.json to peer 1", func() {
			cid = cli.Push(recordV4Path).WithArgs("--output", "raw").OnServer(utils.Peer1Addr).ShouldSucceed()

			// Track CID for cleanup
			RegisterCIDForCleanup(cid, "sync")

			// Validate that the returned CID correctly represents the pushed data
			utils.LoadAndValidateCID(cid, recordV4Path)

			// Sign the record
			output := cli.Sign(cid, privateKeyPath).OnServer(utils.Peer1Addr).ShouldSucceed()
			ginkgo.GinkgoWriter.Printf("Sign output: %s", output)
		})

		ginkgo.It("should publish record_070_sync_v4.json", func() {
			cli.Routing().Publish(cid).OnServer(utils.Peer1Addr).ShouldSucceed()
		})

		ginkgo.It("should push record_070_sync_v5.json to peer 1", func() {
			cidV5 = cli.Push(recordV5Path).WithArgs("--output", "raw").OnServer(utils.Peer1Addr).ShouldSucceed()

			// Track CID for cleanup
			RegisterCIDForCleanup(cidV5, "sync")

			// Validate that the returned CID correctly represents the pushed data
			utils.LoadAndValidateCID(cidV5, recordV5Path)

			// Sign the record
			output := cli.Sign(cidV5, privateKeyPath).OnServer(utils.Peer1Addr).ShouldSucceed()
			ginkgo.GinkgoWriter.Printf("Sign output: %s", output)
		})

		ginkgo.It("should publish record_070_sync_v5.json", func() {
			cli.Routing().Publish(cidV5).OnServer(utils.Peer1Addr).ShouldSucceed()
		})

		ginkgo.It("should fail to pull record_070_sync_v4.json from peer 2", func() {
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
		})

		// Wait for sync to complete
		ginkgo.It("should wait for sync to complete", func() {
			// Poll sync status until it changes from PENDING to IN_PROGRESS
			output := cli.Sync().Status(syncID).OnServer(utils.Peer2Addr).ShouldEventuallyContain("IN_PROGRESS", 120*time.Second)
			ginkgo.GinkgoWriter.Printf("Current sync status: %s", output)

			// Wait for 60 seconds to ensure the sync is complete (reduce flakiness)
			time.Sleep(60 * time.Second)
		})

		ginkgo.It("should succeed to pull record_070_sync_v4.json from peer 2 after sync", func() {
			output := cli.Pull(cid).WithArgs("--output", "json").OnServer(utils.Peer2Addr).ShouldSucceed()

			// Compare the output with the expected JSON
			equal, err := utils.CompareOASFRecords([]byte(output), testdata.ExpectedRecordV070SyncV4JSON)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(equal).To(gomega.BeTrue())
		})

		ginkgo.It("should succeed to search for record_070_sync_v4.json from peer 2 after sync", func() {
			// Search should eventually return the cid in peer 2 (retry until monitor indexes the record)
			output := cli.Search().WithName("directory.agntcy.org/cisco/marketing-strategy-v4").OnServer(utils.Peer2Addr).ShouldEventuallyContain(cid, 240*time.Second)

			ginkgo.GinkgoWriter.Printf("Search found cid: %s", output)
		})

		ginkgo.It("should verify the record_070_sync_v4.json from peer 2 after sync", func() {
			output := cli.Verify(cid).OnServer(utils.Peer2Addr).ShouldSucceed()

			// Verify the output
			gomega.Expect(output).To(gomega.ContainSubstring("Record signature is: trusted"))
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
			ginkgo.GinkgoWriter.Printf("Verifying initial state - peer 3 should not have any records\n")
			_ = cli.Pull(cid).OnServer(utils.Peer3Addr).ShouldFail()   // v4 (NLP) should not exist
			_ = cli.Pull(cidV5).OnServer(utils.Peer3Addr).ShouldFail() // v5 (Audio) should not exist

			ginkgo.GinkgoWriter.Printf("Running routing search for 'audio' skill\n")
			searchOutput := cli.Routing().Search().WithArgs("--skill", "audio").WithArgs("--output", "json").OnServer(utils.Peer3Addr).ShouldSucceed()

			ginkgo.GinkgoWriter.Printf("Routing search output: %s\n", searchOutput)
			gomega.Expect(searchOutput).To(gomega.ContainSubstring(cidV5))

			ginkgo.GinkgoWriter.Printf("Creating sync by tag with 'audio' search output\n")
			output := cli.Sync().CreateFromStdin(searchOutput).OnServer(utils.Peer3Addr).ShouldSucceed()
			gomega.Expect(output).To(gomega.ContainSubstring("Sync IDs created:"))

			// Extract sync ID using simple string methods
			// Find the quoted UUID in the output
			start := strings.Index(output, `[`)
			end := strings.LastIndex(output, `]`)
			gomega.Expect(start).To(gomega.BeNumerically(">", -1), "Expected to find opening quote")
			gomega.Expect(end).To(gomega.BeNumerically(">", start), "Expected to find closing quote")
			syncID = output[start+1 : end]

			ginkgo.GinkgoWriter.Printf("Sync ID: %s", syncID)
		})

		// Wait for sync to complete
		ginkgo.It("should wait for sync to complete", func() {
			_ = cli.Sync().Status(syncID).OnServer(utils.Peer3Addr).ShouldEventuallyContain("IN_PROGRESS", 120*time.Second)

			// Wait for 60 seconds to ensure the sync is complete (reduce flakiness)
			time.Sleep(60 * time.Second)
		})

		ginkgo.It("should succeed to pull record_070_sync_v5.json from peer 3 after sync", func() {
			output := cli.Pull(cidV5).WithArgs("--output", "json").OnServer(utils.Peer3Addr).ShouldSucceed()

			// Compare the output with the expected JSON
			equal, err := utils.CompareOASFRecords([]byte(output), testdata.ExpectedRecordV070SyncV5JSON)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(equal).To(gomega.BeTrue())
		})

		ginkgo.It("should succeed to search for record_070_sync_v5.json from peer 3 after sync", func() {
			// Search should eventually return the cid in peer 2 (retry until monitor indexes the record)
			output := cli.Search().WithName("directory.agntcy.org/cisco/marketing-strategy-v5").OnServer(utils.Peer3Addr).ShouldEventuallyContain(cidV5, 240*time.Second)

			ginkgo.GinkgoWriter.Printf("Search found cid: %s", output)
		})

		ginkgo.It("should verify the record_070_sync_v5.json from peer 3 after sync", func() {
			output := cli.Verify(cidV5).OnServer(utils.Peer3Addr).ShouldSucceed()

			// Verify the output
			gomega.Expect(output).To(gomega.ContainSubstring("Record signature is: trusted"))
		})

		ginkgo.It("should fail to pull record_070_sync_v4.json from peer 3 after sync", func() {
			_ = cli.Pull(cid).OnServer(utils.Peer3Addr).ShouldFail()

			// CLEANUP: This is the last test in the sync functionality Context
			// Clean up sync test records to ensure isolation from subsequent test files
			ginkgo.DeferCleanup(func() {
				CleanupNetworkRecords(syncTestCIDs, "sync tests")
			})
		})
	})
})
