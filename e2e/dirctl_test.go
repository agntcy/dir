// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	_ "embed"
	"os"
	"path/filepath"
	"strings"
	"time"

	clicmd "github.com/agntcy/dir/cli/cmd"
	"github.com/agntcy/dir/e2e/config"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

//go:embed testdata/agent.json
var expectedAgentJSON []byte

var _ = ginkgo.Describe("Running dirctl end-to-end tests using a local single node deployment", func() {
	ginkgo.BeforeEach(func() {
		if cfg.DeploymentMode != config.DeploymentModeLocal {
			ginkgo.Skip("Skipping test, not in local mode")
		}

		// Reset CLI command state to ensure clean state between tests
		ResetCLIState()
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
			var outputBuffer bytes.Buffer

			pushCmd := clicmd.RootCmd
			pushCmd.SetOut(&outputBuffer)
			pushCmd.SetArgs([]string{
				"push",
				tempAgentPath,
			})

			err := pushCmd.Execute()
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			tempAgentCID = strings.TrimSpace(outputBuffer.String())

			// CID is not empty
			gomega.Expect(tempAgentCID).NotTo(gomega.BeEmpty())

			// TODO: validate the CID is valid
		})

		ginkgo.It("should successfully pull an existing agent", func() {
			var outputBuffer bytes.Buffer

			pullCmd := clicmd.RootCmd
			pullCmd.SetOut(&outputBuffer)
			pullCmd.SetArgs([]string{
				"pull",
				tempAgentCID,
			})

			err := pullCmd.Execute()
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})

		ginkgo.It("should push the same agent again and return the same digest", func() {
			var outputBuffer bytes.Buffer

			pushCmd := clicmd.RootCmd
			pushCmd.SetOut(&outputBuffer)
			pushCmd.SetArgs([]string{
				"push",
				tempAgentPath,
			})

			err := pushCmd.Execute()
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Ensure the CIDs are the same
			newAgentCID := outputBuffer.String()
			gomega.Expect(newAgentCID).To(gomega.Equal(tempAgentCID))
		})

		ginkgo.It("should push two different agents and return different digests", func() {
			var outputBuffer bytes.Buffer

			// Modify the agent file to create a different agent
			tempAgentPath2 := filepath.Join(tempAgentDir, "agent2.json")
			err := os.WriteFile(tempAgentPath2, []byte(`{"name": "different-agent", "signature": {}}`), 0o600)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Push second agent
			pushCmd2 := clicmd.RootCmd
			pushCmd2.SetOut(&outputBuffer)
			pushCmd2.SetArgs([]string{
				"push",
				tempAgentPath2,
			})

			err = pushCmd2.Execute()
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			newAgentCID := outputBuffer.String()

			// Ensure the CIDs are different
			gomega.Expect(newAgentCID).NotTo(gomega.Equal(tempAgentCID))
		})

		ginkgo.It("should pull a non-existent agent and return an error", func() {
			var outputBuffer bytes.Buffer

			pullCmd := clicmd.RootCmd
			pullCmd.SetOut(&outputBuffer)
			pullCmd.SetArgs([]string{
				"pull",
				"non-existent-CID",
			})

			err := pullCmd.Execute()
			gomega.Expect(err).To(gomega.HaveOccurred())
		})
	})

	ginkgo.Context("agent search", func() {
		ginkgo.It("should search for records with every filter and return their CID", func() {
			var outputBuffer bytes.Buffer

			searchCmd := clicmd.RootCmd
			searchCmd.SetOut(&outputBuffer)
			searchCmd.SetArgs([]string{
				"search",
				"--limit",
				"10",
				"--offset",
				"0",
				"--query",
				"name=directory.agntcy.org/cisco/marketing-strategy",
				"--query",
				"version=v1.0.0",
				"--query",
				"skill-id=10201",
				"--query",
				"skill-name=Natural Language Processing/Text Completion",
				"--query",
				"locator=docker-image:https://ghcr.io/agntcy/marketing-strategy",
				"--query",
				"extension=schema.oasf.agntcy.org/features/runtime/framework:v0.0.0",
			})

			err := searchCmd.Execute()
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Check if the output contains the expected CID (trim newline from search output)
			searchOutput := strings.TrimSpace(outputBuffer.String())
			gomega.Expect(searchOutput).To(gomega.Equal(tempAgentCID))
		})
	})

	ginkgo.Context("agent delete", func() {
		ginkgo.It("should successfully delete an agent", func() {
			var outputBuffer bytes.Buffer

			deleteCmd := clicmd.RootCmd
			deleteCmd.SetOut(&outputBuffer)
			deleteCmd.SetArgs([]string{
				"delete",
				tempAgentCID,
			})

			err := deleteCmd.Execute()
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})

		ginkgo.It("should fail to pull a deleted agent", func() {
			// Add a small delay to ensure delete operation is fully processed
			time.Sleep(100 * time.Millisecond)

			var outputBuffer bytes.Buffer

			pullCmd := clicmd.RootCmd
			pullCmd.SetOut(&outputBuffer)
			pullCmd.SetArgs([]string{
				"pull",
				tempAgentCID,
			})

			err := pullCmd.Execute()
			gomega.Expect(err).To(gomega.HaveOccurred())
		})
	})
})
