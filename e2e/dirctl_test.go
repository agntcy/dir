// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"

	clicmd "github.com/agntcy/dir/cli/cmd"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

//go:embed testdata/agent.json
var expectedAgentJSON []byte

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
				"--artifact=python-package:http://ghcr.io/cisco-agents/marketing-strategy",
				"--artifact=docker-image:http://ghcr.io/cisco-agents/marketing-strategy",
				"--author=author1",
				"--author=author2",
				"--category=category1",
				"--category=category2",
				"--config-file=testdata/build.config.yaml",
				marketingStrategyPath,
			})

			err := compileCmd.Execute()
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			err = os.MkdirAll(filepath.Dir(tempAgentPath), 0755)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			err = os.WriteFile(tempAgentPath, outputBuffer.Bytes(), 0644)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Compare the output with the expected JSON
			equal, err := compareJSON(outputBuffer.Bytes(), expectedAgentJSON)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(equal).To(gomega.BeTrue())
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

func compareJSON(json1, json2 []byte) (bool, error) {
	var agent1, agent2 coretypes.Agent

	err := json.Unmarshal(json1, &agent1)
	if err != nil {
		return false, err
	}

	err = json.Unmarshal(json2, &agent2)
	if err != nil {
		return false, err
	}

	// Overwrite fields
	agent1.CreatedAt = agent2.CreatedAt

	// Sort the authors slices
	sort.Strings(agent1.Authors)
	sort.Strings(agent2.Authors)

	// Sort the locators slices by type
	sort.Slice(agent1.Locators, func(i, j int) bool {
		return agent1.Locators[i].Type < agent1.Locators[j].Type
	})
	sort.Slice(agent2.Locators, func(i, j int) bool {
		return agent2.Locators[i].Type < agent2.Locators[j].Type
	})

	// Sort the extensions slices
	sort.Slice(agent1.Extensions, func(i, j int) bool {
		return agent1.Extensions[i].Name < agent1.Extensions[j].Name
	})
	sort.Slice(agent2.Extensions, func(i, j int) bool {
		return agent2.Extensions[i].Name < agent2.Extensions[j].Name
	})

	return reflect.DeepEqual(agent1, agent2), nil //nolint:govet
}
