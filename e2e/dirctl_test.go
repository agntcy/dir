// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	_ "embed"
	"os"
	"path/filepath"

	clicmd "github.com/agntcy/dir/cli/cmd"
	"github.com/agntcy/dir/e2e/config"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/opencontainers/go-digest"
)

//go:embed testdata/agent.json
var expectedAgentJSON []byte

var _ = ginkgo.Describe("Running dirctl end-to-end tests using a local single node deployment", func() {
	ginkgo.BeforeEach(func() {
		if cfg.DeploymentMode != config.DeploymentModeLocal {
			ginkgo.Skip("Skipping test, not in local mode")
		}
	})

	// Test params
	var tempAgentDigest string

	tempAgentDir := os.Getenv("E2E_COMPILE_OUTPUT_DIR")
	if tempAgentDir == "" {
		tempAgentDir = os.TempDir()
	}
	tempAgentPath := filepath.Join(tempAgentDir, "agent.json")

	ginkgo.Context("agent build", func() {
		ginkgo.It("should build an agent", func() {
			var outputBuffer bytes.Buffer

			compileCmd := clicmd.RootCmd
			compileCmd.SetOut(&outputBuffer)
			compileCmd.SetArgs([]string{
				"build",
				"testdata",
			})

			err := compileCmd.Execute()
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			err = os.MkdirAll(filepath.Dir(tempAgentPath), 0o755)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			err = os.WriteFile(tempAgentPath, outputBuffer.Bytes(), 0o600)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Compare the output with the expected JSON
			equal, err := compareJSONAgents(outputBuffer.Bytes(), expectedAgentJSON)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(equal).To(gomega.BeTrue())
		})
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

			tempAgentDigest = outputBuffer.String()

			// Ensure the digest is valid
			_, err = digest.Parse(tempAgentDigest)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})

		ginkgo.It("should successfully pull an existing agent", func() {
			var outputBuffer bytes.Buffer

			pullCmd := clicmd.RootCmd
			pullCmd.SetOut(&outputBuffer)
			pullCmd.SetArgs([]string{
				"pull",
				tempAgentDigest,
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

			// Ensure the digests are the same
			newAgentDigest := outputBuffer.String()
			gomega.Expect(newAgentDigest).To(gomega.Equal(tempAgentDigest))
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
			newAgentDigest := outputBuffer.String()

			// Ensure the digests are different
			gomega.Expect(newAgentDigest).NotTo(gomega.Equal(tempAgentDigest))
		})

		ginkgo.It("should pull a non-existent agent and return an error", func() {
			var outputBuffer bytes.Buffer

			pullCmd := clicmd.RootCmd
			pullCmd.SetOut(&outputBuffer)
			pullCmd.SetArgs([]string{
				"pull",
				"non-existent-digest",
			})

			err := pullCmd.Execute()
			gomega.Expect(err).To(gomega.HaveOccurred())
		})
	})

	ginkgo.Context("agent delete", func() {
		ginkgo.It("should successfully delete an agent", func() {
			var outputBuffer bytes.Buffer

			deleteCmd := clicmd.RootCmd
			deleteCmd.SetOut(&outputBuffer)
			deleteCmd.SetArgs([]string{
				"delete",
				tempAgentDigest,
			})

			err := deleteCmd.Execute()
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})

		ginkgo.It("should fail to pull a deleted agent", func() {
			var outputBuffer bytes.Buffer

			pullCmd := clicmd.RootCmd
			pullCmd.SetOut(&outputBuffer)
			pullCmd.SetArgs([]string{
				"pull",
				tempAgentDigest,
			})

			err := pullCmd.Execute()
			gomega.Expect(err).To(gomega.HaveOccurred())
		})
	})
})
