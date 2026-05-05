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

var _ = ginkgo.Describe("Running dirctl end-to-end tests for name resolution across nodes", func() {
	var syncID string

	// Setup temp files for CLI commands
	recordPath := filepath.Join(os.TempDir(), "record_070_name_resolution_test.json")

	// Create directory and write record data
	_ = os.MkdirAll(filepath.Dir(recordPath), 0o755)
	_ = os.WriteFile(recordPath, testdata.ExpectedRecordV070NameResolutionJSON, 0o600)

	ginkgo.BeforeEach(func() {
		utils.ResetCLIState()
	})

	ginkgo.Context("name resolution after sync", ginkgo.Ordered, func() {
		var cid string

		const (
			recordName    = "directory.agntcy.org/cisco/name-resolution-test"
			recordVersion = "v1.0.0"
		)

		ginkgo.It("should push record to peer 1", func() {
			cid = testEnv.Peer1.Push(recordPath).WithArgs("--output", "raw").ShouldSucceed()

			// Track CID for cleanup
			RegisterCIDForCleanup(cid, "name_resolution")

			// Validate CID
			utils.LoadAndValidateCID(cid, recordPath)
		})

		ginkgo.It("should fail to resolve by name from peer 2 before sync", func() {
			// Name resolution should fail because peer2 doesn't have the record
			_ = testEnv.Peer2.Info(recordName).ShouldFail()
		})

		ginkgo.It("should fail to pull by name from peer 2 before sync", func() {
			// Pull by name should fail because peer2 doesn't have the record indexed
			_ = testEnv.Peer2.Pull(recordName).ShouldFail()
		})

		ginkgo.It("should create sync from peer 1 to peer 2", func() {
			output := testEnv.Peer2.Sync().Create(testEnv.Config.Peer1InternalServerAddress).ShouldEventuallyContain("Sync created with ID: ", 45*time.Second)

			syncID = strings.TrimPrefix(output, "Sync created with ID: ")
		})

		ginkgo.It("should wait for sync to complete", func() {
			output := testEnv.Peer2.Sync().Status(syncID).ShouldEventuallyContain("COMPLETED", 240*time.Second)
			ginkgo.GinkgoWriter.Printf("Current sync status: %s\n", output)
		})

		ginkgo.It("should resolve by name from peer 2 after sync", func() {
			// Info by name should work - this tests the naming resolution
			output := testEnv.Peer2.Info(recordName).WithArgs("--output", "json").ShouldEventuallySucceed(120 * time.Second)

			// Verify the output contains the expected CID
			gomega.Expect(output).To(gomega.ContainSubstring(cid))
		})

		ginkgo.It("should resolve by name:version from peer 2 after sync", func() {
			// Info by name:version should work
			output := testEnv.Peer2.Info(recordName+":"+recordVersion).WithArgs("--output", "json").ShouldSucceed()

			// Verify the output contains the expected CID
			gomega.Expect(output).To(gomega.ContainSubstring(cid))
		})

		ginkgo.It("should pull by name from peer 2 after sync", func() {
			// Pull by name should now work
			output := testEnv.Peer2.Pull(recordName).WithArgs("--output", "json").ShouldSucceed()

			// Compare the output with the expected JSON
			equal, err := utils.CompareOASFRecords([]byte(output), testdata.ExpectedRecordV070NameResolutionJSON)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(equal).To(gomega.BeTrue())
		})

		ginkgo.It("should pull by name:version from peer 2 after sync", func() {
			// Pull by name:version should work
			output := testEnv.Peer2.Pull(recordName+":"+recordVersion).WithArgs("--output", "json").ShouldSucceed()

			// Compare the output with the expected JSON
			equal, err := utils.CompareOASFRecords([]byte(output), testdata.ExpectedRecordV070NameResolutionJSON)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(equal).To(gomega.BeTrue())
		})

		ginkgo.It("should pull by name@cid from peer 2 after sync with hash verification", func() {
			// Pull by name@cid should work and verify the hash
			output := testEnv.Peer2.Pull(recordName+"@"+cid).WithArgs("--output", "json").ShouldSucceed()

			// Compare the output with the expected JSON
			equal, err := utils.CompareOASFRecords([]byte(output), testdata.ExpectedRecordV070NameResolutionJSON)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(equal).To(gomega.BeTrue())
		})

		ginkgo.It("should cleanup network publications", func() {
			// Cleanup
			ginkgo.DeferCleanup(func() {
				CleanupNetworkRecords(nameResolutionTestCIDs, "name resolution tests", testEnv.PeerCLIs())
			})
		})
	})
})
