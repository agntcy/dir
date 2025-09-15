// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	_ "embed"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/agntcy/dir/e2e/config"
	"github.com/agntcy/dir/e2e/utils"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

// Using peer addresses from utils.constants

var _ = ginkgo.Describe("Running dirctl end-to-end tests for sync commands", func() {
	var cli *utils.CLI
	var syncID string

	// Setup temp files
	tempDir := os.Getenv("E2E_COMPILE_OUTPUT_DIR")
	if tempDir == "" {
		tempDir = os.TempDir()
	}
	tempPath := filepath.Join(tempDir, "record_sync_test.json")

	// Create directory and write record data
	_ = os.MkdirAll(filepath.Dir(tempPath), 0o755)
	_ = os.WriteFile(tempPath, expectedRecordV070SyncJSON, 0o600)

	ginkgo.BeforeEach(func() {
		if cfg.DeploymentMode != config.DeploymentModeNetwork {
			ginkgo.Skip("Skipping test, not in network mode")
		}

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

		ginkgo.It("should push record_v070_sync.json to peer 1", func() {
			cid = cli.Push(tempPath).OnServer(utils.Peer1Addr).ShouldSucceed()

			// Validate that the returned CID correctly represents the pushed data
			utils.LoadAndValidateCID(cid, tempPath)
		})

		ginkgo.It("should fail to pull record_v070_sync.json from peer 2", func() {
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

		ginkgo.It("should succeed to pull record_v070_sync.json from peer 2 after sync", func() {
			output := cli.Pull(cid).OnServer(utils.Peer2Addr).ShouldSucceed()

			// Compare the output with the expected JSON
			equal, err := utils.CompareOASFRecords([]byte(output), expectedRecordV070SyncJSON)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(equal).To(gomega.BeTrue())
		})

		ginkgo.It("should succeed to search for record_v070_sync.json from peer 2 after sync", func() {
			// Search should eventually return the cid in peer 2 (retry until monitor indexes the record)
			output := cli.Search().WithQuery("name", "directory.agntcy.org/cisco/marketing-strategy-v4").OnServer(utils.Peer2Addr).ShouldEventuallyContain(cid, 240*time.Second)

			ginkgo.GinkgoWriter.Printf("Search found cid: %s", output)
		})

		ginkgo.It("should verify the record_v070_sync.json from peer 2 after sync", func() {
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
	})
})
