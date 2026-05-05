// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package network

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/agntcy/dir/tests/e2e/shared/testdata"
	"github.com/agntcy/dir/tests/e2e/shared/utils"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

// Using peer addresses from utils.constants

// Package-level variables for cleanup (accessible by AfterSuite)
// CIDs are now tracked in network_suite_test.go

var _ = ginkgo.Describe("Running dirctl end-to-end tests for sync commands", func() {
	var (
		syncID         string
		deleteSyncID   string
		privateKeyPath string
		tempKeyDir     string
	)

	// Setup temp files for CLI commands (CLI needs actual files on disk)
	recordV4Path := filepath.Join(os.TempDir(), "record_070_sync_v4_test.json")
	recordV5Path := filepath.Join(os.TempDir(), "record_070_sync_v5_test.json")

	// Create directory and write record data
	_ = os.MkdirAll(filepath.Dir(recordV4Path), 0o755)
	_ = os.WriteFile(recordV4Path, testdata.ExpectedRecordV070SyncV4JSON, 0o600)
	_ = os.WriteFile(recordV5Path, testdata.ExpectedRecordV070SyncV5JSON, 0o600)

	ginkgo.BeforeEach(func() {
		utils.ResetCLIState()
	})

	ginkgo.Context("create command", func() {
		ginkgo.It("should accept valid remote URL format", func() {
			output := testEnv.Peer1.Sync().Create("https://directory.example.com").ShouldSucceed()

			gomega.Expect(output).To(gomega.ContainSubstring("Sync created with ID: "))
			syncID = strings.TrimPrefix(output, "Sync created with ID: ")
		})
	})

	ginkgo.Context("list command", func() {
		ginkgo.It("should execute without arguments and return a list with the created sync", func() {
			output := testEnv.Peer1.Sync().List().ShouldSucceed()

			gomega.Expect(output).To(gomega.ContainSubstring(syncID))
			gomega.Expect(output).To(gomega.ContainSubstring("https://directory.example.com"))
		})
	})

	ginkgo.Context("status command", func() {
		ginkgo.It("should accept a sync ID argument and return the sync status", func() {
			output := testEnv.Peer1.Sync().Status(syncID).ShouldSucceed()

			gomega.Expect(output).To(gomega.ContainSubstring("PENDING"))
		})
	})

	ginkgo.Context("delete command", func() {
		ginkgo.It("should accept a sync ID argument and delete the sync", func() {
			// Command may fail due to network/auth issues, but argument parsing should work
			_, err := testEnv.Peer1.Sync().Delete(syncID).Execute()
			if err != nil {
				gomega.Expect(err.Error()).NotTo(gomega.ContainSubstring("required"))
			}
		})

		ginkgo.It("should return deleted status", func() {
			testEnv.Peer1.Sync().Status(syncID).ShouldContain("DELETE")
		})
	})

	ginkgo.Context("sync functionality", ginkgo.Ordered, func() {
		var (
			cid   string
			cidV5 string
		)

		// Setup cosign key pair for signing tests

		ginkgo.BeforeAll(func() {
			var err error

			tempKeyDir, err = os.MkdirTemp("", "sync-test-keys")
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Generate cosign key pair
			cosignPassword := "testpassword"
			utils.GenerateCosignKeyPair(tempKeyDir, cosignPassword)
			privateKeyPath = filepath.Join(tempKeyDir, "cosign.key")

			// Verify key file was created
			gomega.Expect(privateKeyPath).To(gomega.BeAnExistingFile())

			// Set cosign password for signing
			err = os.Setenv("COSIGN_PASSWORD", cosignPassword)
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
			cid = testEnv.Peer1.Push(recordV4Path).WithArgs("--output", "raw").ShouldSucceed()

			// Track CID for cleanup
			RegisterCIDForCleanup(cid, "sync")

			// Validate that the returned CID correctly represents the pushed data
			utils.LoadAndValidateCID(cid, recordV4Path)

			// Sign the record
			output := testEnv.Peer1.Sign(cid, privateKeyPath).ShouldSucceed()
			ginkgo.GinkgoWriter.Printf("Sign output: %s", output)
		})

		ginkgo.It("should publish record_070_sync_v4.json", func() {
			testEnv.Peer1.Routing().Publish(cid).ShouldSucceed()
		})

		ginkgo.It("should push record_070_sync_v5.json to peer 1", func() {
			cidV5 = testEnv.Peer1.Push(recordV5Path).WithArgs("--output", "raw").ShouldSucceed()

			// Track CID for cleanup
			RegisterCIDForCleanup(cidV5, "sync")

			// Validate that the returned CID correctly represents the pushed data
			utils.LoadAndValidateCID(cidV5, recordV5Path)

			// Sign the record
			output := testEnv.Peer1.Sign(cidV5, privateKeyPath).ShouldSucceed()
			ginkgo.GinkgoWriter.Printf("Sign output: %s", output)
		})

		ginkgo.It("should publish record_070_sync_v5.json", func() {
			testEnv.Peer1.Routing().Publish(cidV5).ShouldSucceed()
		})

		ginkgo.It("should fail to pull record_070_sync_v4.json from peer 2", func() {
			_ = testEnv.Peer2.Pull(cid).ShouldFail()
		})

		ginkgo.It("should create sync from peer 1 to peer 2", func() {
			output := testEnv.Peer2.Sync().Create(testEnv.Config.Peer1InternalServerAddress).ShouldSucceed()

			gomega.Expect(output).To(gomega.ContainSubstring("Sync created with ID: "))
			syncID = strings.TrimPrefix(output, "Sync created with ID: ")
		})

		ginkgo.It("should list the sync", func() {
			output := testEnv.Peer2.Sync().List().ShouldSucceed()

			gomega.Expect(output).To(gomega.ContainSubstring(syncID))
			gomega.Expect(output).To(gomega.ContainSubstring(testEnv.Config.Peer1InternalServerAddress))
		})

		// Wait for sync to complete
		ginkgo.It("should wait for sync to complete", func() {
			output := testEnv.Peer2.Sync().Status(syncID).ShouldEventuallyContain("COMPLETED", 240*time.Second)
			ginkgo.GinkgoWriter.Printf("Current sync status: %s", output)
		})

		ginkgo.It("should succeed to pull record_070_sync_v4.json from peer 2 after sync", func() {
			output := testEnv.Peer2.Pull(cid).WithArgs("--output", "json").ShouldSucceed()

			// Compare the output with the expected JSON
			equal, err := utils.CompareOASFRecords([]byte(output), testdata.ExpectedRecordV070SyncV4JSON)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(equal).To(gomega.BeTrue())
		})

		ginkgo.It("should succeed to search for record_070_sync_v4.json from peer 2 after sync", func() {
			// Search should eventually return the cid in peer 2 (retry until monitor indexes the record)
			output := testEnv.Peer2.Search().WithName("directory.agntcy.org/cisco/marketing-strategy-v4").ShouldEventuallyContain(cid, 240*time.Second)

			ginkgo.GinkgoWriter.Printf("Search found cid: %s", output)
		})

		ginkgo.It("should verify the record_070_sync_v4.json from peer 2 after sync", func() {
			// Verification should eventually contain "trusted" (reconciler signature task).
			_ = testEnv.Peer2.Verify(cid).WithArgs("--from-server").ShouldEventuallyContain("Record signature is: trusted", 90*time.Second)
		})

		// Delete sync from peer 2
		ginkgo.It("should delete sync from peer 2", func() {
			output := testEnv.Peer2.Sync().Create(testEnv.Config.Peer1InternalServerAddress).ShouldEventuallySucceed(60 * time.Second)

			gomega.Expect(output).To(gomega.ContainSubstring("Sync created with ID: "))
			deleteSyncID = strings.TrimPrefix(output, "Sync created with ID: ")

			testEnv.Peer2.Sync().Delete(deleteSyncID).ShouldEventuallySucceed(60 * time.Second)
		})

		// Wait for sync to complete
		ginkgo.It("should wait for delete to complete", func() {
			// Poll sync status until it changes from DELETE_PENDING to DELETED
			output := testEnv.Peer2.Sync().Status(deleteSyncID).ShouldEventuallyContain("DELETED", 240*time.Second)
			ginkgo.GinkgoWriter.Printf("Current sync status: %s", output)
		})

		ginkgo.It("should create sync from peer 1 to peer 3 using routing search piped to sync create", func() {
			ginkgo.GinkgoWriter.Printf("Verifying initial state - peer 3 should not have any records\n")

			_ = testEnv.Peer3.Pull(cid).ShouldFail()   // v4 (NLP) should not exist
			_ = testEnv.Peer3.Pull(cidV5).ShouldFail() // v5 (Audio) should not exist

			ginkgo.GinkgoWriter.Printf("Running routing search for 'audio' skill\n")

			searchOutput := testEnv.Peer3.Routing().Search().WithArgs("--skill", "audio").WithArgs("--output", "json").ShouldEventuallyContain(cidV5, 240*time.Second)

			ginkgo.GinkgoWriter.Printf("Routing search output: %s\n", searchOutput)
			gomega.Expect(searchOutput).To(gomega.ContainSubstring(cidV5))

			ginkgo.GinkgoWriter.Printf("Creating sync by tag with 'audio' search output\n")

			output := testEnv.Peer3.Sync().CreateFromStdin(searchOutput).ShouldEventuallyContain("Sync IDs created: ", 240*time.Second)

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
			_ = testEnv.Peer3.Sync().Status(syncID).ShouldEventuallyContain("COMPLETED", 240*time.Second)
		})

		ginkgo.It("should succeed to pull record_070_sync_v5.json from peer 3 after sync", func() {
			output := testEnv.Peer3.Pull(cidV5).WithArgs("--output", "json").ShouldSucceed()

			// Compare the output with the expected JSON
			equal, err := utils.CompareOASFRecords([]byte(output), testdata.ExpectedRecordV070SyncV5JSON)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(equal).To(gomega.BeTrue())
		})

		ginkgo.It("should succeed to search for record_070_sync_v5.json from peer 3 after sync", func() {
			// Search should eventually return the cid in peer 2 (retry until monitor indexes the record)
			output := testEnv.Peer3.Search().WithName("directory.agntcy.org/cisco/marketing-strategy-v5").ShouldEventuallyContain(cidV5, 240*time.Second)

			ginkgo.GinkgoWriter.Printf("Search found cid: %s", output)
		})

		ginkgo.It("should verify the record_070_sync_v5.json from peer 3 after sync", func() {
			// Verification should eventually contain "trusted" (reconciler signature task).
			_ = testEnv.Peer3.Verify(cidV5).WithArgs("--from-server").ShouldEventuallyContain("Record signature is: trusted", 90*time.Second)
		})

		ginkgo.It("should fail to pull record_070_sync_v4.json from peer 3 after sync", func() {
			_ = testEnv.Peer3.Pull(cid).ShouldFail()

			// CLEANUP: This is the last test in the sync functionality Context
			// Clean up sync test records to ensure isolation from subsequent test files
			ginkgo.DeferCleanup(func() {
				CleanupNetworkRecords(syncTestCIDs, "sync tests", testEnv.PeerCLIs())
			})
		})
	})
})
