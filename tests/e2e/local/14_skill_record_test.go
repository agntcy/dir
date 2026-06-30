// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"github.com/agntcy/dir/tests/e2e/shared/utils"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

// Hard-coded (not imported from server/skill) to keep the e2e module
// independent and to assert the wire-level contract consumers rely on.
const (
	skillRecordName       = "org.agntcy/directory"
	skillModuleName       = "core/language_model/agentskills"
	mcpModuleName         = "integration/mcp"
	skillArtifactMediaTyp = "application/agent-skills+md"
)

var _ = ginkgo.Describe("DIR self-published SKILL record", func() {
	ginkgo.BeforeEach(func() {
		utils.ResetCLIState()
	})

	ginkgo.It("should be discoverable by name and carry the SKILL.md bytes plus an MCP module", func() {
		var cid string

		// Publish runs asynchronously after Start; poll for it.
		// `search --output raw` returns `[<cid> ...]`; strip the brackets
		// so the value is a bare CID `pull` can accept.
		gomega.Eventually(func(g gomega.Gomega) {
			out := testEnv.CLI.Search().
				WithName(skillRecordName).
				WithLimit(1).
				WithArgs("--output", "raw").
				ShouldSucceed()

			cid = strings.Trim(strings.TrimSpace(out), "[]")
			g.Expect(cid).NotTo(gomega.BeEmpty(),
				"DIR skill record should be searchable by name %q", skillRecordName)
			g.Expect(cid).NotTo(gomega.ContainSubstring(" "),
				"search returned multiple CIDs; expected exactly one")
		}).WithTimeout(30 * time.Second).WithPolling(time.Second).Should(gomega.Succeed())

		raw := testEnv.CLI.Pull(cid).WithArgs("--output", "json").ShouldSucceed()

		var doc struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Modules     []struct {
				Name     string `json:"name"`
				Artifact struct {
					MediaType string `json:"media_type"`
					Data      string `json:"data"`
					Digest    string `json:"digest"`
					Size      int    `json:"size"`
				} `json:"artifact"`
			} `json:"modules"`
		}

		gomega.Expect(json.Unmarshal([]byte(raw), &doc)).To(gomega.Succeed())

		gomega.Expect(doc.Name).To(gomega.Equal(skillRecordName))
		gomega.Expect(doc.Description).NotTo(gomega.BeEmpty())

		// Both modules must live on the same record.
		moduleNames := make([]string, 0, len(doc.Modules))
		for _, m := range doc.Modules {
			moduleNames = append(moduleNames, m.Name)
		}

		gomega.Expect(moduleNames).To(gomega.ContainElement(skillModuleName))
		gomega.Expect(moduleNames).To(gomega.ContainElement(mcpModuleName))

		skillArtifact := struct {
			MediaType string `json:"media_type"`
			Data      string `json:"data"`
			Digest    string `json:"digest"`
			Size      int    `json:"size"`
		}{}

		for _, m := range doc.Modules {
			if m.Name == skillModuleName {
				skillArtifact = m.Artifact

				break
			}
		}

		gomega.Expect(skillArtifact.MediaType).To(gomega.Equal(skillArtifactMediaTyp))
		gomega.Expect(skillArtifact.Digest).To(gomega.HavePrefix("sha256:"))
		gomega.Expect(skillArtifact.Size).To(gomega.BeNumerically(">", 0))

		decoded, err := base64.StdEncoding.DecodeString(skillArtifact.Data)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(string(decoded)).To(gomega.ContainSubstring("# AGNTCY Directory"))
		gomega.Expect(decoded).To(gomega.HaveLen(skillArtifact.Size))
	})
})
