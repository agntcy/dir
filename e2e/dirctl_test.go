// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	_ "embed"
	"os"
	"path/filepath"
	"time"

	"github.com/agntcy/dir/e2e/config"
	"github.com/agntcy/dir/e2e/utils"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

//go:embed testdata/agent.json
var expectedAgentJSON []byte

var _ = ginkgo.Describe("Running dirctl end-to-end tests using a local single node deployment", func() {
	var cli *utils.CLI
	ginkgo.BeforeEach(func() {
		if cfg.DeploymentMode != config.DeploymentModeLocal {
			ginkgo.Skip("Skipping test, not in local mode")
		}

		utils.ResetCLIState()
		// Initialize CLI helper
		cli = utils.NewCLI()
	})

	// Test params
	var tempAgentCID string

	tempAgentDir := os.Getenv("E2E_COMPILE_OUTPUT_DIR")
	if tempAgentDir == "" {
		tempAgentDir = os.TempDir()
	}
	tempAgentPath := filepath.Join(tempAgentDir, "agent.json")

	// Setup: Copy test agent to temp location for push/pull tests
	ginkgo.BeforeEach(func() {
		err := os.MkdirAll(filepath.Dir(tempAgentPath), 0o755)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		err = os.WriteFile(tempAgentPath, expectedAgentJSON, 0o600)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	ginkgo.Context("agent push and pull", func() {
		ginkgo.It("should successfully push an agent", func() {
			tempAgentCID = cli.Push(tempAgentPath).ShouldSucceed()

			// Validate that the returned CID correctly represents the pushed data
			utils.LoadAndValidateCID(tempAgentCID, tempAgentPath)
		})

		ginkgo.It("should successfully pull an existing agent", func() {
			cli.Pull(tempAgentCID).ShouldSucceed()
		})

		ginkgo.It("should push the same agent again and return the same digest", func() {
			cli.Push(tempAgentPath).ShouldReturn(tempAgentCID)
		})

		ginkgo.It("should push two different agents and return different digests", func() {
			// Modify the agent file to create a different agent
			tempAgentPath2 := filepath.Join(tempAgentDir, "agent2.json")
			err := os.WriteFile(tempAgentPath2, []byte(`{"name": "different-agent", "signature": {}}`), 0o600)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Push second agent
			newAgentCID := cli.Push(tempAgentPath2).ShouldSucceed()

			// Ensure the CIDs are different
			gomega.Expect(newAgentCID).NotTo(gomega.Equal(tempAgentCID))
		})

		ginkgo.It("should pull a non-existent agent and return an error", func() {
			_ = cli.Pull("non-existent-CID").ShouldFail()
		})
	})

	ginkgo.Context("agent search", func() {
		ginkgo.It("should search for records with every filter and return their CID", func() {
			cli.Search().
				WithLimit(10).
				WithOffset(0).
				WithQuery("name", "directory.agntcy.org/cisco/marketing-strategy").
				WithQuery("version", "v1.0.0").
				WithQuery("skill-id", "10201").
				WithQuery("skill-name", "Natural Language Processing/Text Completion").
				WithQuery("locator", "docker-image:https://ghcr.io/agntcy/marketing-strategy").
				WithQuery("extension", "schema.oasf.agntcy.org/features/runtime/framework:v0.0.0").
				ShouldReturn(tempAgentCID)
		})
	})

	ginkgo.Context("agent delete", func() {
		ginkgo.It("should successfully delete an agent", func() {
			cli.Delete(tempAgentCID).ShouldSucceed()
		})

		ginkgo.It("should fail to pull a deleted agent", func() {
			// Add a small delay to ensure delete operation is fully processed
			time.Sleep(100 * time.Millisecond)

			_ = cli.Pull(tempAgentCID).ShouldFail()
		})
	})
})
