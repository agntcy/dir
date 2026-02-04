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

var _ = ginkgo.Describe("Running dirctl end-to-end tests for name resolution across nodes", func() {
	var cli *utils.CLI
	var syncID string

	// Setup temp files for CLI commands
	tempDir := os.Getenv("E2E_COMPILE_OUTPUT_DIR")
	if tempDir == "" {
		tempDir = os.TempDir()
	}
	recordPath := filepath.Join(tempDir, "record_070_name_resolution_test.json")

	// Create directory and write record data
	_ = os.MkdirAll(filepath.Dir(recordPath), 0o755)
	_ = os.WriteFile(recordPath, testdata.ExpectedRecordV070SyncV4JSON, 0o600)

	ginkgo.BeforeEach(func() {
		if cfg.DeploymentMode != config.DeploymentModeNetwork {
			ginkgo.Skip("Skipping test, not in network mode")
		}

		utils.ResetCLIState()
		cli = utils.NewCLI()
	})

	ginkgo.Context("name resolution after sync", ginkgo.Ordered, func() {
		var cid string
		const recordName = "directory.agntcy.org/cisco/marketing-strategy-v4"
		const recordVersion = "v4.0.0"

		ginkgo.It("should push record to peer 1", func() {
			cid = cli.Push(recordPath).WithArgs("--output", "raw").OnServer(utils.Peer1Addr).ShouldSucceed()

			// Track CID for cleanup
			RegisterCIDForCleanup(cid, "name_resolution")

			// Validate CID
			utils.LoadAndValidateCID(cid, recordPath)
		})

		ginkgo.It("should fail to resolve by name from peer 2 before sync", func() {
			// Name resolution should fail because peer2 doesn't have the record
			_ = cli.Info(recordName).OnServer(utils.Peer2Addr).ShouldFail()
		})

		ginkgo.It("should fail to pull by name from peer 2 before sync", func() {
			// Pull by name should fail because peer2 doesn't have the record indexed
			_ = cli.Pull(recordName).OnServer(utils.Peer2Addr).ShouldFail()
		})

		ginkgo.It("should create sync from peer 1 to peer 2", func() {
			output := cli.Sync().Create(utils.Peer1InternalAddr).OnServer(utils.Peer2Addr).ShouldSucceed()

			gomega.Expect(output).To(gomega.ContainSubstring("Sync created with ID: "))
			syncID = strings.TrimPrefix(output, "Sync created with ID: ")
		})

		ginkgo.It("should wait for sync to complete", func() {
			// Poll sync status until it changes to IN_PROGRESS
			output := cli.Sync().Status(syncID).OnServer(utils.Peer2Addr).ShouldEventuallyContain("IN_PROGRESS", 120*time.Second)
			ginkgo.GinkgoWriter.Printf("Current sync status: %s\n", output)

			// Wait for sync to fully complete
			time.Sleep(60 * time.Second)
		})

		ginkgo.It("should resolve by name from peer 2 after sync", func() {
			// Info by name should work - this tests the naming resolution
			output := cli.Info(recordName).WithArgs("--output", "json").OnServer(utils.Peer2Addr).ShouldEventuallySucceed(120 * time.Second)

			// Verify the output contains the expected CID
			gomega.Expect(output).To(gomega.ContainSubstring(cid))
		})

		ginkgo.It("should resolve by name:version from peer 2 after sync", func() {
			// Info by name:version should work
			output := cli.Info(recordName+":"+recordVersion).WithArgs("--output", "json").OnServer(utils.Peer2Addr).ShouldSucceed()

			// Verify the output contains the expected CID
			gomega.Expect(output).To(gomega.ContainSubstring(cid))
		})

		ginkgo.It("should pull by name from peer 2 after sync", func() {
			// Pull by name should now work
			output := cli.Pull(recordName).WithArgs("--output", "json").OnServer(utils.Peer2Addr).ShouldSucceed()

			// Compare the output with the expected JSON
			equal, err := utils.CompareOASFRecords([]byte(output), testdata.ExpectedRecordV070SyncV4JSON)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(equal).To(gomega.BeTrue())
		})

		ginkgo.It("should pull by name:version from peer 2 after sync", func() {
			// Pull by name:version should work
			output := cli.Pull(recordName+":"+recordVersion).WithArgs("--output", "json").OnServer(utils.Peer2Addr).ShouldSucceed()

			// Compare the output with the expected JSON
			equal, err := utils.CompareOASFRecords([]byte(output), testdata.ExpectedRecordV070SyncV4JSON)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(equal).To(gomega.BeTrue())
		})

		ginkgo.It("should pull by name@cid from peer 2 after sync with hash verification", func() {
			// Pull by name@cid should work and verify the hash
			output := cli.Pull(recordName+"@"+cid).WithArgs("--output", "json").OnServer(utils.Peer2Addr).ShouldSucceed()

			// Compare the output with the expected JSON
			equal, err := utils.CompareOASFRecords([]byte(output), testdata.ExpectedRecordV070SyncV4JSON)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(equal).To(gomega.BeTrue())
		})

		ginkgo.It("should delete sync from peer 2", func() {
			cli.Sync().Delete(syncID).OnServer(utils.Peer2Addr).ShouldSucceed()
		})

		ginkgo.It("should wait for delete to complete", func() {
			output := cli.Sync().Status(syncID).OnServer(utils.Peer2Addr).ShouldEventuallyContain("DELETED", 120*time.Second)
			ginkgo.GinkgoWriter.Printf("Current sync status: %s\n", output)

			// Cleanup
			ginkgo.DeferCleanup(func() {
				CleanupNetworkRecords(nameResolutionTestCIDs, "name resolution tests")
			})
		})
	})
})
