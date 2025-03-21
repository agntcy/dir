// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	_ "embed"
	"os"
	"path/filepath"

	clicmd "github.com/agntcy/dir/cli/cmd"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

//go:embed testdata/agent.json
var expectedAgentJSON []byte

var _ = ginkgo.Describe("dirctl end-to-end tests", func() {
	// Test params
	var tempAgentPath string

	ginkgo.BeforeEach(func() {
		// Load common config
		tempAgentDir := os.Getenv("E2E_COMPILE_OUTPUT_DIR")
		if tempAgentDir == "" {
			tempAgentDir = os.TempDir()
		}
		tempAgentPath = filepath.Join(tempAgentDir, "agent.json")
	})

	ginkgo.Context("agent compilation", func() {
		ginkgo.It("should compile an agent", func() {
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
		var agentDigest string

		ginkgo.It("should push an agent", func() {
			var outputBuffer bytes.Buffer

			pushCmd := clicmd.RootCmd
			pushCmd.SetOut(&outputBuffer)
			pushCmd.SetArgs([]string{
				"push",
				"--from-file", tempAgentPath,
			})

			err := pushCmd.Execute()
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Retrieve agentID from output
			agentDigest = outputBuffer.String()
		})

		ginkgo.It("should pull an existing agent", func() {
			var outputBuffer bytes.Buffer

			pullCmd := clicmd.RootCmd
			pullCmd.SetOut(&outputBuffer)
			pullCmd.SetArgs([]string{
				"pull",
				"--digest", agentDigest,
			})

			err := pullCmd.Execute()
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})
	})

	ginkgo.Context("agent immutability", func() {
		ginkgo.It("push existing agent again", func() {
			pushCmd := clicmd.RootCmd
			pushCmd.SetArgs([]string{
				"push",
				"--from-file", tempAgentPath,
			})

			err := pushCmd.Execute()
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})
	})
})
