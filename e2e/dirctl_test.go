// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	"os"
	"path/filepath"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"

	clicmd "github.com/agntcy/dir/cli/cmd"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("dirctl end-to-end tests", func() {
	var (
		// Test params
		marketingStrategyPath string
		tempAgentPath         string
	)

	ginkgo.BeforeEach(func() {
		// Load common config
		examplesDir := "testdata/"
		marketingStrategyPath = filepath.Join(examplesDir, "marketing-strategy")
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
				"--name=marketing-strategy",
				"--version=v1.0.0",
				"--artifact-type=python-package",
				"--artifact-url=http://ghcr.io/cisco-agents/marketing-strategy",
				"--author=author1",
				"--author=author2",
				marketingStrategyPath,
			})

			err := compileCmd.Execute()
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			err = os.MkdirAll(filepath.Dir(tempAgentPath), 0755)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			err = os.WriteFile(tempAgentPath, outputBuffer.Bytes(), 0644)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})
	})

	ginkgo.Context("agent push and pull", func() {
		var agentDigest coretypes.Digest

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
			err = agentDigest.Decode(outputBuffer.String())
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})

		ginkgo.It("should pull an existing agent", func() {
			var outputBuffer bytes.Buffer

			pullCmd := clicmd.RootCmd
			pullCmd.SetOut(&outputBuffer)
			pullCmd.SetArgs([]string{
				"pull",
				"--digest", agentDigest.Encode(),
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
