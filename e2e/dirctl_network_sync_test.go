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

// expectedAgentV2JSON is now embedded in dirctl_test.go and reused here

var _ = ginkgo.Describe("Running dirctl end-to-end tests for sync commands", func() {
	var cli *utils.CLI
	var syncID string

	// Setup temp agent files
	tempAgentDir := os.Getenv("E2E_COMPILE_OUTPUT_DIR")
	if tempAgentDir == "" {
		tempAgentDir = os.TempDir()
	}
	tempAgentV2Path := filepath.Join(tempAgentDir, "agent_v2_sync_test.json")
	tempAgentV3Path := filepath.Join(tempAgentDir, "agent_v3_sync_test.json")

	// Create directory and write agent data
	_ = os.MkdirAll(filepath.Dir(tempAgentV2Path), 0o755)
	_ = os.WriteFile(tempAgentV2Path, expectedAgentV2JSON, 0o600)
	_ = os.WriteFile(tempAgentV3Path, expectedAgentV3JSON, 0o600)

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
		var agentCID string

		ginkgo.It("should push agent_v2.json to peer 1", func() {
			agentCID = cli.Push(tempAgentV2Path).OnServer(utils.Peer1Addr).ShouldSucceed()

			// Validate that the returned CID correctly represents the pushed data
			utils.LoadAndValidateCID(agentCID, tempAgentV2Path)
		})

		ginkgo.It("should fail to pull agent_v2.json from peer 2", func() {
			_ = cli.Pull(agentCID).OnServer(utils.Peer2Addr).ShouldFail()
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
		})

		ginkgo.It("should succeed to pull agent_v2.json from peer 2 after sync", func() {
			output := cli.Pull(agentCID).OnServer(utils.Peer2Addr).ShouldSucceed()

			// Compare the output with the expected JSON
			equal, err := utils.CompareOASFRecords([]byte(output), expectedAgentV2JSON)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(equal).To(gomega.BeTrue())
		})

		ginkgo.It("should succeed to search for agent_v2.json from peer 2 after sync", func() {
			// Search should eventually return the agentCID in peer 2 (retry until monitor indexes the record)
			output := cli.Search().WithQuery("name", "directory.agntcy.org/cisco/marketing-strategy-v2").OnServer(utils.Peer2Addr).ShouldEventuallyContain(agentCID, 240*time.Second)

			ginkgo.GinkgoWriter.Printf("Search found agentCID: %s", output)
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

		// Wait a reasonable time to ensure any residual sync processes would have completed
		ginkgo.It("should wait to ensure no sync occurs", func() {
			time.Sleep(120 * time.Second)
		})

		// Push agent_v3.json to peer 1 (this is a NEW agent after sync deletion)
		ginkgo.It("should push agent_v3.json to peer 1", func() {
			agentCID = cli.Push(tempAgentV3Path).OnServer(utils.Peer1Addr).ShouldSucceed()

			// Validate that the returned CID correctly represents the pushed data
			utils.LoadAndValidateCID(agentCID, tempAgentV3Path)
		})

		// Pull agent_v3.json from peer 2
		ginkgo.It("should fail to pull agent_v3.json from peer 2", func() {
			_ = cli.Pull(agentCID).OnServer(utils.Peer2Addr).ShouldFail()
		})
	})
})
