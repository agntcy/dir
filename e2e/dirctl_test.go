// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	"context"
	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/cli/util"
	"github.com/agntcy/dir/client"
	"os"
	"path/filepath"

	buildcmd "github.com/agntcy/dir/cli/cmd/build"
	pullcmd "github.com/agntcy/dir/cli/cmd/pull"
	pushcmd "github.com/agntcy/dir/cli/cmd/push"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("dirctl end-to-end tests", func() {
	var (
		// Test params
		marketingStrategyPath string
		tempAgentPath         string
		testCtx               context.Context
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

		// Create client
		client, err := client.New(client.WithDefaultConfig())
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		// Set context for CLI client
		testCtx = util.SetClientForContext(context.Background(), client)
	})

	ginkgo.Context("agent compilation", func() {
		ginkgo.It("should compile an agent", func() {
			var outputBuffer bytes.Buffer

			compileCmd := buildcmd.Command
			compileCmd.SetOut(&outputBuffer)
			compileCmd.SetContext(testCtx)
			compileCmd.SetArgs([]string{
				"--name=marketing-strategy",
				"--version=v1.0.0",
				"--artifact-type=LOCATOR_TYPE_PYTHON_PACKAGE",
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

			pushCmd := pushcmd.Command
			pushCmd.SetOut(&outputBuffer)
			pushCmd.SetContext(testCtx)
			pushCmd.SetArgs([]string{
				"--from-file", tempAgentPath,
			})

			err := pushCmd.Execute()
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Retrieve agentID from output
			err = agentDigest.FromString(outputBuffer.String())
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})

		ginkgo.It("should pull an existing agent", func() {
			var outputBuffer bytes.Buffer

			pullCmd := pullcmd.Command
			pullCmd.SetOut(&outputBuffer)
			pullCmd.SetContext(testCtx)
			pullCmd.SetArgs([]string{
				"--digest", agentDigest.ToString(),
			})

			err := pullCmd.Execute()
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})
	})

	ginkgo.Context("agent immutability", func() {
		ginkgo.It("push existing agent again", func() {
			pushCmd := pushcmd.Command
			pushCmd.SetContext(testCtx)
			pushCmd.SetArgs([]string{
				"--from-file", tempAgentPath,
			})

			err := pushCmd.Execute()
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})
	})
})
