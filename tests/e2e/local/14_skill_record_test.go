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

// Duplicated as literals (rather than imported from server/skill) to keep the
// e2e tests module independent of the server module and to assert the
// wire-level contract consumers will rely on.
const (
	skillRecordName = "agntcy.dir/skill"
	skillModuleName = "core/language_model/agentskills"
)

var _ = ginkgo.Describe("DIR self-published SKILL record", func() {
	ginkgo.BeforeEach(func() {
		utils.ResetCLIState()
	})

	ginkgo.It("should be discoverable by name and carry the SKILL.md bytes in its module artifact", func() {
		var cid string

		// Publishing happens asynchronously after Start returns; poll for it.
		// `search --output raw` formats results as `[<cid> <cid> ...]`; strip
		// the brackets so the CID can be passed straight to `pull`.
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

		gomega.Expect(doc.Modules).NotTo(gomega.BeEmpty())
		gomega.Expect(doc.Modules[0].Name).To(gomega.Equal(skillModuleName))

		artifact := doc.Modules[0].Artifact
		gomega.Expect(artifact.MediaType).To(gomega.Equal("text/markdown"))
		gomega.Expect(artifact.Digest).To(gomega.HavePrefix("sha256:"))
		gomega.Expect(artifact.Size).To(gomega.BeNumerically(">", 0))

		decoded, err := base64.StdEncoding.DecodeString(artifact.Data)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(string(decoded)).To(gomega.ContainSubstring("# AGNTCY Directory"))
		gomega.Expect(decoded).To(gomega.HaveLen(artifact.Size))
	})
})
