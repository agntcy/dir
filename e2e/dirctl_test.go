// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
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
				"--config=testdata/build.config.yaml",
			})

			err := compileCmd.Execute()
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			err = os.MkdirAll(filepath.Dir(tempAgentPath), 0o755)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			err = os.WriteFile(tempAgentPath, outputBuffer.Bytes(), 0o600)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Compare the output with the expected JSON
			equal, err := compareJSON(outputBuffer.Bytes(), expectedAgentJSON)
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

func compareJSON(json1, json2 []byte) (bool, error) {
	var agent1, agent2 coretypes.Agent

	// Convert to JSON
	if err := json.Unmarshal(json1, &agent1); err != nil {
		return false, fmt.Errorf("failed to unmarshal json: %w", err)
	}

	if err := json.Unmarshal(json2, &agent2); err != nil {
		return false, fmt.Errorf("failed to unmarshal json: %w", err)
	}

	// Overwrite CreatedAt
	agent1.CreatedAt = agent2.GetCreatedAt()

	// Convert back to JSON
	json1, err := json.Marshal(&agent1)
	if err != nil {
		return false, fmt.Errorf("failed to marshal json: %w", err)
	}

	json2, err = json.Marshal(&agent2)
	if err != nil {
		return false, fmt.Errorf("failed to marshal json: %w", err)
	}

	return reflect.DeepEqual(json1, json2), nil //nolint:govet
}
