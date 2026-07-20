// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/agntcy/dir/tests/e2e/shared/testdata"
	"github.com/agntcy/dir/tests/e2e/shared/utils"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Extractor enricher import", func() {
	ginkgo.BeforeEach(func() {
		utils.ResetCLIState()
	})

	ginkgo.Context("import with enricher.extractor", ginkgo.Ordered, ginkgo.Serial, func() {
		var (
			tempDir  string
			cfgPath  string
			cardPath string
			cid      string
		)

		ginkgo.BeforeAll(func() {
			// Check for the oasf-sdk extractor manifest written by `dirctl init`.
			// This mirrors extractor.IsProvisioned without importing the internal package.
			home, homeErr := os.UserHomeDir()
			gomega.Expect(homeErr).NotTo(gomega.HaveOccurred())

			manifest := filepath.Join(home, ".agntcy", "oasf-sdk", "extractor", "manifest.json")
			if _, err := os.Stat(manifest); err != nil {
				ginkgo.Skip("OASF extractor not provisioned — run `dirctl init` to enable extractor enricher tests")
			}

			var err error

			tempDir, err = os.MkdirTemp("", "extractor-enricher-e2e-*")
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			cardPath = filepath.Join(tempDir, "agent-card.json")
			gomega.Expect(os.WriteFile(cardPath, testdata.A2AAgentCard, 0o600)).To(gomega.Succeed())

			// Empty extractor block uses the config saved by `dirctl init`
			// (internalextractor.LoadConfigured).
			cfgPath = filepath.Join(tempDir, "import.config.yaml")
			gomega.Expect(os.WriteFile(cfgPath, []byte("enricher:\n  extractor: {}\n"), 0o600)).To(gomega.Succeed())
		})

		ginkgo.AfterAll(func() {
			if cid != "" {
				_ = testEnv.CLI.Delete(cid).ShouldSucceed()
			}

			if tempDir != "" {
				_ = os.RemoveAll(tempDir)
			}
		})

		ginkgo.It("imports an A2A agent card using the extractor enricher", func() {
			cidFile := filepath.Join(tempDir, "imported.cids")

			testEnv.CLI.Import("a2a", cardPath).WithArgs(
				"--config="+cfgPath,
				"--force",
				"--output-cids="+cidFile,
			).ShouldEventuallySucceed(60 * time.Second)

			cidData, err := os.ReadFile(cidFile)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			cid = strings.TrimSpace(string(cidData))
			gomega.Expect(cid).NotTo(gomega.BeEmpty(), "imported CID should not be empty")
		})

		ginkgo.It("enriches the record with at least one skill and domain", func() {
			raw := testEnv.CLI.Pull(cid).WithArgs("--output", "json").ShouldSucceed()

			var doc struct {
				Skills []struct {
					Name string `json:"name"`
					ID   uint32 `json:"id"`
				} `json:"skills"`
				Domains []struct {
					Name string `json:"name"`
					ID   uint32 `json:"id"`
				} `json:"domains"`
			}

			gomega.Expect(json.Unmarshal([]byte(raw), &doc)).To(gomega.Succeed())

			gomega.Expect(doc.Skills).NotTo(gomega.BeEmpty(), "extractor should produce at least one skill")
			gomega.Expect(doc.Domains).NotTo(gomega.BeEmpty(), "extractor should produce at least one domain")

			gomega.Expect(doc.Skills[0].Name).NotTo(gomega.BeEmpty())
			gomega.Expect(doc.Domains[0].Name).NotTo(gomega.BeEmpty())
		})
	})
})
