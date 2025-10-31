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

// Using peer addresses from utils.constants

// Package-level variables for cleanup (accessible by AfterSuite)
// CIDs are now tracked in network_suite_test.go

var _ = ginkgo.Describe("Running dirctl end-to-end tests using a network multi peer deployment", ginkgo.Ordered, func() {
	var cli *utils.CLI
	var cid string

	// Setup temp record file
	tempDir := os.Getenv("E2E_COMPILE_OUTPUT_DIR")
	if tempDir == "" {
		tempDir = os.TempDir()
	}
	tempPath := filepath.Join(tempDir, "record_070_network_test.json")

	// Create directory and write record data
	_ = os.MkdirAll(filepath.Dir(tempPath), 0o755)
	_ = os.WriteFile(tempPath, testdata.ExpectedRecordV070JSON, 0o600)

	ginkgo.BeforeEach(func() {
		if cfg.DeploymentMode != config.DeploymentModeNetwork {
			ginkgo.Skip("Skipping test, not in network mode")
		}

		// Initialize CLI helper
		cli = utils.NewCLI()
	})

	ginkgo.It("should push record_070.json to peer 1", func() {
		cid = cli.Push(tempPath).WithArgs("--output", "raw").OnServer(utils.Peer1Addr).ShouldSucceed()

		// Track CID for cleanup
		RegisterCIDForCleanup(cid, "deploy")

		// Validate that the returned CID correctly represents the pushed data
		utils.LoadAndValidateCID(cid, tempPath)
	})

	ginkgo.It("should pull record_070.json from peer 1", func() {
		cli.Pull(cid).OnServer(utils.Peer1Addr).ShouldSucceed()
	})

	ginkgo.It("should fail to pull record_070.json from peer 2", func() {
		_ = cli.Pull(cid).OnServer(utils.Peer2Addr).ShouldFail()
	})

	ginkgo.It("should publish record_070.json to the network on peer 1", func() {
		cli.Routing().Publish(cid).OnServer(utils.Peer1Addr).ShouldSucceed()

		// Wait at least 10 seconds to ensure the record is published.
		time.Sleep(15 * time.Second)
	})

	ginkgo.It("should fail publish record_070.json to the network on peer 2 that does not store the record", func() {
		_ = cli.Routing().Publish(cid).OnServer(utils.Peer2Addr).ShouldFail()
	})

	ginkgo.It("should list local records correctly (List is local-only)", func() {
		// Reset CLI state to ensure clean test environment
		utils.ResetCLIState()

		// Test that List only returns records on the peer that published them
		// Peer1 published the record, so it should find it locally
		output := cli.Routing().List().WithCid(cid).OnServer(utils.Peer1Addr).ShouldSucceed()

		// Should find the local record
		gomega.Expect(output).To(gomega.ContainSubstring(cid))
		gomega.Expect(output).To(gomega.ContainSubstring("Local records"))

		// Reset CLI state before testing Peer2
		utils.ResetCLIState()

		// Peer2 did NOT publish the record, so List should not find it locally
		// (even though it might be available via DHT/network)
		output2 := cli.Routing().List().WithCid(cid).OnServer(utils.Peer2Addr).ShouldSucceed()

		// Should NOT find the record locally on Peer2
		gomega.Expect(output2).To(gomega.ContainSubstring("No local records found"))
	})

	ginkgo.It("should list by skill correctly on local vs remote peers", func() {
		// Reset CLI state to ensure clean test environment
		utils.ResetCLIState()

		// Test Peer1 (published the record) - should find it locally
		output1 := cli.Routing().List().WithSkill("natural_language_processing").OnServer(utils.Peer1Addr).ShouldSucceed()

		// Should find the local record with expected labels
		gomega.Expect(output1).To(gomega.ContainSubstring(cid))
		gomega.Expect(output1).To(gomega.ContainSubstring("Local records"))
		gomega.Expect(output1).To(gomega.ContainSubstring("/skills/natural_language_processing/natural_language_generation/text_completion"))
		gomega.Expect(output1).To(gomega.ContainSubstring("/skills/natural_language_processing/analytical_reasoning/problem_solving"))

		// Reset CLI state again before testing Peer2
		utils.ResetCLIState()

		// Test Peer2 (did NOT publish the record) - should not find it locally
		output2 := cli.Routing().List().WithSkill("natural_language_processing").OnServer(utils.Peer2Addr).ShouldSucceed()

		// Should NOT find the record locally, but should show helpful message
		gomega.Expect(output2).NotTo(gomega.ContainSubstring(cid))
		// Note: If no local records match, CLI might show empty results or no records message
	})

	ginkgo.It("should show routing info statistics", func() {
		// Reset CLI state to ensure clean test environment
		utils.ResetCLIState()

		// Test routing info on Peer1 (has published records)
		output1 := cli.Routing().Info().OnServer(utils.Peer1Addr).ShouldSucceed()

		// Should show local routing statistics
		gomega.Expect(output1).To(gomega.ContainSubstring("Local Routing Summary"))
		gomega.Expect(output1).To(gomega.ContainSubstring("Total Records:"))
		gomega.Expect(output1).To(gomega.ContainSubstring("Skills Distribution"))

		// Reset CLI state before testing Peer2
		utils.ResetCLIState()

		// Test routing info on Peer2 (no published records)
		output2 := cli.Routing().Info().OnServer(utils.Peer2Addr).ShouldSucceed()

		// Should show empty statistics or no records message
		gomega.Expect(output2).To(gomega.ContainSubstring("Local Routing Summary"))
		// Peer2 might have 0 records or show "No local records found"
	})

	ginkgo.It("should discover remote records via routing search", func() {
		// Reset CLI state to ensure clean test environment
		utils.ResetCLIState()

		// Test routing search from Peer2 to discover records published by Peer1
		// This tests whether DHT propagation is working in the e2e environment
		output := cli.Routing().Search().
			WithSkill("natural_language_processing").
			WithLimit(10).
			OnServer(utils.Peer2Addr).ShouldEventuallyContain(cid, 60*time.Second)

		ginkgo.GinkgoWriter.Printf("=== DHT DISCOVERY TEST OUTPUT ===\n%s", output)

		// CLEANUP: This is the last test in this Describe block
		// Clean up deploy test records to ensure isolation from subsequent test files
		ginkgo.DeferCleanup(func() {
			CleanupNetworkRecords(deployTestCIDs, "deploy tests")
		})
	})
})
