// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"os"
	"path/filepath"
	"time"

	"github.com/agntcy/dir/e2e/config"
	"github.com/agntcy/dir/e2e/utils"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

// Using peer addresses from utils.constants

var _ = ginkgo.Describe("Running dirctl end-to-end tests using a network multi peer deployment", func() {
	var cli *utils.CLI
	var cid string

	// Setup temp record file
	tempDir := os.Getenv("E2E_COMPILE_OUTPUT_DIR")
	if tempDir == "" {
		tempDir = os.TempDir()
	}
	tempPath := filepath.Join(tempDir, "record_v3_network_test.json")

	// Create directory and write V3 record data
	_ = os.MkdirAll(filepath.Dir(tempPath), 0o755)
	_ = os.WriteFile(tempPath, expectedRecordV3JSON, 0o600)

	ginkgo.BeforeEach(func() {
		if cfg.DeploymentMode != config.DeploymentModeNetwork {
			ginkgo.Skip("Skipping test, not in network mode")
		}

		// Initialize CLI helper
		cli = utils.NewCLI()
	})

	ginkgo.It("should push an record to peer 1", func() {
		cid = cli.Push(tempPath).OnServer(utils.Peer1Addr).ShouldSucceed()

		// Validate that the returned CID correctly represents the pushed data
		utils.LoadAndValidateCID(cid, tempPath)
	})

	ginkgo.It("should pull the record from peer 1", func() {
		cli.Pull(cid).OnServer(utils.Peer1Addr).ShouldSucceed()
	})

	ginkgo.It("should fail to pull the record from peer 2", func() {
		_ = cli.Pull(cid).OnServer(utils.Peer2Addr).ShouldFail()
	})

	ginkgo.It("should publish an record to the network on peer 1", func() {
		cli.Publish(cid).OnServer(utils.Peer1Addr).ShouldSucceed()

		// Wait at least 10 seconds to ensure the record is published.
		time.Sleep(15 * time.Second)
	})

	ginkgo.It("should fail publish an record to the network on peer 2 that does not store the record", func() {
		_ = cli.Publish(cid).OnServer(utils.Peer2Addr).ShouldFail()
	})

	ginkgo.It("should list local records correctly (List is local-only)", func() {
		// Reset CLI state to ensure clean test environment
		utils.ResetCLIState()

		// Test that List only returns records on the peer that published them
		// Peer1 published the record, so it should find it locally
		output := cli.List().WithCid(cid).OnServer(utils.Peer1Addr).ShouldSucceed()

		// Should find the local record
		gomega.Expect(output).To(gomega.ContainSubstring(cid))
		gomega.Expect(output).To(gomega.ContainSubstring("Local Record"))

		// Reset CLI state before testing Peer2
		utils.ResetCLIState()

		// Peer2 did NOT publish the record, so List should not find it locally
		// (even though it might be available via DHT/network)
		output2 := cli.List().WithCid(cid).OnServer(utils.Peer2Addr).ShouldSucceed()

		// Should NOT find the record locally on Peer2
		gomega.Expect(output2).To(gomega.ContainSubstring("not found in local records"))
		gomega.Expect(output2).To(gomega.ContainSubstring("Use 'dirctl search' to find providers"))
	})

	ginkgo.It("should list by skill correctly on local vs remote peers", func() {
		// Reset CLI state to ensure clean test environment
		utils.ResetCLIState()

		// Test Peer1 (published the record) - should find it locally
		output1 := cli.List().WithSkill("/skills/Natural Language Processing").OnServer(utils.Peer1Addr).ShouldSucceed()

		// Should find the local record with expected labels
		gomega.Expect(output1).To(gomega.ContainSubstring(cid))
		gomega.Expect(output1).To(gomega.ContainSubstring("Local Record"))
		gomega.Expect(output1).To(gomega.ContainSubstring("/skills/schema.oasf.agntcy.org/skills/contextual_comprehension"))
		gomega.Expect(output1).To(gomega.ContainSubstring("/skills/schema.oasf.agntcy.org/skills/entity_recognition"))

		// Reset CLI state again before testing Peer2
		utils.ResetCLIState()

		// Test Peer2 (did NOT publish the record) - should not find it locally
		output2 := cli.List().WithSkill("/skills/Natural Language Processing").OnServer(utils.Peer2Addr).ShouldSucceed()

		// Should NOT find the record locally, but should show helpful message
		gomega.Expect(output2).NotTo(gomega.ContainSubstring(cid))
		// Note: If no local records match, CLI might show empty results or no records message
	})
})
