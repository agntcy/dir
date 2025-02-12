// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	"os"
	"path/filepath"

	buildcmd "github.com/agntcy/dir/cli/cmd/build"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("dirctl end-to-end tests", func() {
	var (
		// Test params
		marketingStrategyPath string
		tempAgentPath         string
	)

	ginkgo.BeforeEach(func() {
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

			compileCmd := buildcmd.Command
			compileCmd.SetOut(&outputBuffer)
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

	// ginkgo.Context("agent push and pull", func() {
	// 	ginkgo.It("should push an agent", func() {
	// 		var outputBuffer bytes.Buffer

	// 		pushCmd := pushcmd.Command
	// 		pushCmd.SetOut(&outputBuffer)
	// 		pushCmd.SetArgs([]string{
	// 			"--from-file", tempAgentPath,
	// 		})

	// 		err := pushCmd.Execute()
	// 		gomega.Expect(err).NotTo(gomega.HaveOccurred())

	// 		// Retrieve agentID from output
	// 		var agent types.Agent
	// 		err = json.Unmarshal(outputBuffer.Bytes(), &agent)
	// 		gomega.Expect(err).NotTo(gomega.HaveOccurred())

	// 		agentID = agent.Id
	// 	})

	// 	ginkgo.It("should pull an existing agent", func() {
	// 		var outputBuffer bytes.Buffer

	// 		pullCmd := pullcmd.Command
	// 		pullCmd.SetOut(&outputBuffer)
	// 		pullCmd.SetArgs([]string{
	// 			"--id", agentID,
	// 		})

	// 		err := pullCmd.Execute()
	// 		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	// 	})
	// })

	// ginkgo.Context("agent immutability", func() {
	// 	ginkgo.It("push existing agent again", func() {
	// 		pushCmd := pushcmd.Command
	// 		pushCmd.SetArgs([]string{
	// 			"--from-file", tempAgentPath,
	// 		})

	// 		err := pushCmd.Execute()
	// 		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	// 	})
	// })

	// ginkgo.Context("agent search", func() {

	// 	type searchResult struct {
	// 		ID string `json:"id"`
	// 	}

	// 	ginkgo.It("should push example agents", func() {
	// 		agentPaths := []string{
	// 			linkedinAgentPath,
	// 			readerAgentPath,
	// 		}

	// 		for _, agentPath := range agentPaths {
	// 			pushCmd := pushcmd.Command
	// 			pushCmd.SetArgs([]string{
	// 				"--from-file", agentPath,
	// 			})

	// 			err := pushCmd.Execute()
	// 			gomega.Expect(err).NotTo(gomega.HaveOccurred())
	// 		}
	// 	})

	// 	ginkgo.It("should search for LinkedIn agents", func() {
	// 		var outputBuffer bytes.Buffer

	// 		searchCmd := searchcmd.Command
	// 		searchCmd.SetOut(&outputBuffer)
	// 		searchCmd.SetArgs([]string{
	// 			"--query", "linkedin poster",
	// 		})

	// 		err := searchCmd.Execute()
	// 		gomega.Expect(err).NotTo(gomega.HaveOccurred())

	// 		var res []searchResult
	// 		err = json.Unmarshal(outputBuffer.Bytes(), &res)
	// 		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	// 		gomega.Expect(len(res)).To(gomega.BeNumerically(">", 0), "No LinkedIn agents found")

	// 		agentID := res[0].ID

	// 		pullCmd := pullcmd.Command
	// 		pullCmd.SetOut(&outputBuffer)
	// 		pullCmd.SetArgs([]string{
	// 			"--id", agentID,
	// 			"--verify",
	// 		})

	// 		err = pullCmd.Execute()
	// 		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	// 	})

	// 	ginkgo.It("should search for all agents", func() {
	// 		var outputBuffer bytes.Buffer

	// 		searchCmd := searchcmd.Command
	// 		searchCmd.SetOut(&outputBuffer)
	// 		searchCmd.SetArgs([]string{
	// 			"--query", "*",
	// 		})

	// 		err := searchCmd.Execute()
	// 		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	// 	})
	// })
})
