// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/agntcy/dir/importer"
	importerconfig "github.com/agntcy/dir/importer/config"
	"github.com/agntcy/dir/importer/enricher"
	"github.com/agntcy/dir/importer/factory"
	"github.com/agntcy/dir/importer/types"
	"github.com/agntcy/dir/tests/e2e/shared/config"
	"github.com/agntcy/dir/tests/e2e/shared/testdata"
	"github.com/agntcy/dir/tests/e2e/shared/utils"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

// importerWithStaticEnricher wraps importer.New, injecting a StaticEnricher so that
// e2e tests can import records without an LLM.
func importerWithStaticEnricher(ctx context.Context, client importerconfig.ClientInterface, cfg importerconfig.Config) (types.Importer, error) {
	cfg.EnricherOverride = enricher.NewStaticEnricher()

	return importer.New(ctx, client, cfg) //nolint:wrapcheck
}

var _ = ginkgo.Describe("Running dirctl end-to-end tests for the export command", func() {
	var cli *utils.CLI

	ginkgo.BeforeEach(func() {
		if cfg.DeploymentMode != config.DeploymentModeLocal {
			ginkgo.Skip("Skipping test, not in local mode")
		}

		utils.ResetCLIState()

		cli = utils.NewCLI()
	})

	ginkgo.Context("Export with default oasf format", ginkgo.Ordered, ginkgo.Serial, func() {
		var cid string

		tempDir, err := os.MkdirTemp("", "export-e2e-*")
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		ginkgo.AfterAll(func() {
			os.RemoveAll(tempDir)
		})

		ginkgo.It("should push a record to set up test data", func() {
			pushPath := filepath.Join(tempDir, "push_record.json")
			err := os.WriteFile(pushPath, testdata.ExpectedRecordV100JSON, 0o600)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			cid = cli.Push(pushPath).WithArgs("--output", "raw").ShouldSucceed()
			gomega.Expect(cid).NotTo(gomega.BeEmpty())
		})

		ginkgo.It("should export a record to stdout by CID", func() {
			output := cli.Export(cid).ShouldSucceed()
			gomega.Expect(output).NotTo(gomega.BeEmpty())

			var parsed map[string]any

			err := json.Unmarshal([]byte(output), &parsed)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "stdout output should be valid JSON")
			gomega.Expect(parsed).To(gomega.HaveKey("name"))
		})

		ginkgo.It("should export a record to a file with explicit extension", func() {
			outPath := filepath.Join(tempDir, "exported_record.json")

			cli.Export(cid).WithArgs("--output-file", outPath).ShouldSucceed()

			data, err := os.ReadFile(outPath)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(data).NotTo(gomega.BeEmpty())

			var parsed map[string]any

			err = json.Unmarshal(data, &parsed)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "file content should be valid JSON")
			gomega.Expect(parsed).To(gomega.HaveKey("name"))
		})

		ginkgo.It("should auto-append file extension when omitted", func() {
			outPath := filepath.Join(tempDir, "no_ext_record")
			expectedPath := outPath + ".json"

			cli.Export(cid).WithArgs("--output-file", outPath).ShouldSucceed()

			_, err := os.Stat(expectedPath)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(),
				"file with auto-appended .json extension should exist")

			data, err := os.ReadFile(expectedPath)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			var parsed map[string]any

			err = json.Unmarshal(data, &parsed)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "file content should be valid JSON")
		})

		ginkgo.It("should fail with an unsupported format", func() {
			err := cli.Export(cid).WithArgs("--format", "nonexistent").ShouldFail()
			gomega.Expect(err.Error()).To(gomega.ContainSubstring("unsupported export format"))
		})

		ginkgo.It("should fail for a non-existent CID", func() {
			_ = cli.Export("non-existent-CID").ShouldFail()
		})

		ginkgo.It("should clean up the test record", func() {
			cli.Delete(cid).ShouldSucceed()
		})
	})

	ginkgo.Context("Export with a2a format", ginkgo.Ordered, ginkgo.Serial, func() {
		var cid string

		tempDir, err := os.MkdirTemp("", "export-a2a-e2e-*")
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		ginkgo.BeforeAll(func() {
			factory.Replace(importerconfig.ImportTypeA2A, importerWithStaticEnricher)
		})

		ginkgo.AfterAll(func() {
			os.RemoveAll(tempDir)
			factory.Replace(importerconfig.ImportTypeA2A, importer.New)
		})

		ginkgo.It("should import an A2A agent card to set up test data", func() {
			cardPath := filepath.Join(tempDir, "agent-card.json")
			gomega.Expect(os.WriteFile(cardPath, testdata.A2AAgentCard, 0o600)).To(gomega.Succeed())

			// Dummy config satisfies enricher validation; the actual enricher is replaced via factory.
			enrichCfg := filepath.Join(tempDir, "mcphost.json")
			gomega.Expect(os.WriteFile(enrichCfg, []byte(`{}`), 0o600)).To(gomega.Succeed())

			cidFile := filepath.Join(tempDir, "imported.cids")

			cli.Import("a2a", cardPath).WithArgs("--enrich-config="+enrichCfg, "--output-cids="+cidFile).ShouldEventuallySucceed(60 * time.Second)

			cidData, err := os.ReadFile(cidFile)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			cid = strings.TrimSpace(string(cidData))
			gomega.Expect(cid).NotTo(gomega.BeEmpty(), "imported CID should not be empty")
		})

		ginkgo.It("should export the record as A2A AgentCard to stdout", func() {
			output := cli.Export(cid).WithArgs("--format", "a2a").ShouldSucceed()
			gomega.Expect(output).NotTo(gomega.BeEmpty())

			var exported map[string]any

			err := json.Unmarshal([]byte(output), &exported)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "stdout output should be valid JSON")

			var original map[string]any
			gomega.Expect(json.Unmarshal(testdata.A2AAgentCard, &original)).To(gomega.Succeed())

			gomega.Expect(exported).To(gomega.Equal(original), "exported A2A card should match the original input")
		})

		ginkgo.It("should export the record as A2A AgentCard to a file", func() {
			outPath := filepath.Join(tempDir, "agent-card.json")

			cli.Export(cid).WithArgs("--format", "a2a", "--output-file", outPath).ShouldSucceed()

			data, err := os.ReadFile(outPath)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			var parsed map[string]any

			err = json.Unmarshal(data, &parsed)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "file content should be valid JSON")
			gomega.Expect(parsed["name"]).To(gomega.Equal("Code Review Agent"))
		})

		ginkgo.It("should auto-append .json extension when omitted", func() {
			outPath := filepath.Join(tempDir, "a2a_no_ext")
			expectedPath := outPath + ".json"

			cli.Export(cid).WithArgs("--format", "a2a", "--output-file", outPath).ShouldSucceed()

			_, err := os.Stat(expectedPath)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(),
				"file with auto-appended .json extension should exist")

			data, err := os.ReadFile(expectedPath)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			var parsed map[string]any

			err = json.Unmarshal(data, &parsed)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "file content should be valid JSON")
		})

		ginkgo.It("should clean up the test record", func() {
			cli.Delete(cid).ShouldSucceed()
		})
	})

	ginkgo.Context("Export with mcp-ghcopilot format", ginkgo.Ordered, ginkgo.Serial, func() {
		var cid string

		tempDir, err := os.MkdirTemp("", "export-mcp-ghcopilot-e2e-*")
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		ginkgo.BeforeAll(func() {
			factory.Replace(importerconfig.ImportTypeMCP, importerWithStaticEnricher)
		})

		ginkgo.AfterAll(func() {
			os.RemoveAll(tempDir)
			factory.Replace(importerconfig.ImportTypeMCP, importer.New)
		})

		ginkgo.It("should import an MCP server descriptor to set up test data", func() {
			serverPath := filepath.Join(tempDir, "mcp-server.json")
			gomega.Expect(os.WriteFile(serverPath, testdata.MCPServer, 0o600)).To(gomega.Succeed())

			enrichCfg := filepath.Join(tempDir, "mcphost.json")
			gomega.Expect(os.WriteFile(enrichCfg, []byte(`{}`), 0o600)).To(gomega.Succeed())

			cidFile := filepath.Join(tempDir, "imported.cids")

			cli.Import("mcp", serverPath).WithArgs("--enrich-config="+enrichCfg, "--output-cids="+cidFile).ShouldEventuallySucceed(60 * time.Second)

			cidData, err := os.ReadFile(cidFile)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			cid = strings.TrimSpace(string(cidData))
			gomega.Expect(cid).NotTo(gomega.BeEmpty(), "imported CID should not be empty")
		})

		ginkgo.It("should export the record as GitHub Copilot MCP config to stdout", func() {
			output := cli.Export(cid).WithArgs("--format", "mcp-ghcopilot").ShouldSucceed()
			gomega.Expect(output).NotTo(gomega.BeEmpty())

			var exported map[string]any

			err := json.Unmarshal([]byte(output), &exported)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "stdout output should be valid JSON")

			gomega.Expect(exported).To(gomega.HaveKey("servers"), "output should have a 'servers' key")
			gomega.Expect(exported).To(gomega.HaveKey("inputs"), "output should have an 'inputs' key")

			servers, ok := exported["servers"].(map[string]any)
			gomega.Expect(ok).To(gomega.BeTrue())
			gomega.Expect(servers).NotTo(gomega.BeEmpty(), "servers map should contain at least one entry")
		})

		ginkgo.It("should export the record as GitHub Copilot MCP config to a file", func() {
			outPath := filepath.Join(tempDir, "ghcopilot.json")

			cli.Export(cid).WithArgs("--format", "mcp-ghcopilot", "--output-file", outPath).ShouldSucceed()

			data, err := os.ReadFile(outPath)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			var parsed map[string]any

			err = json.Unmarshal(data, &parsed)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "file content should be valid JSON")
			gomega.Expect(parsed).To(gomega.HaveKey("servers"))
		})

		ginkgo.It("should auto-append .json extension when omitted", func() {
			outPath := filepath.Join(tempDir, "ghcopilot_no_ext")
			expectedPath := outPath + ".json"

			cli.Export(cid).WithArgs("--format", "mcp-ghcopilot", "--output-file", outPath).ShouldSucceed()

			_, err := os.Stat(expectedPath)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(),
				"file with auto-appended .json extension should exist")

			data, err := os.ReadFile(expectedPath)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			var parsed map[string]any

			err = json.Unmarshal(data, &parsed)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "file content should be valid JSON")
		})

		ginkgo.It("should clean up the test record", func() {
			cli.Delete(cid).ShouldSucceed()
		})
	})

	ginkgo.Context("Export with skill format", ginkgo.Ordered, ginkgo.Serial, func() {
		var cid string

		tempDir, err := os.MkdirTemp("", "export-skill-e2e-*")
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		ginkgo.BeforeAll(func() {
			factory.Replace(importerconfig.ImportTypeAgentSkill, importerWithStaticEnricher)
		})

		ginkgo.AfterAll(func() {
			os.RemoveAll(tempDir)
			factory.Replace(importerconfig.ImportTypeAgentSkill, importer.New)
		})

		ginkgo.It("should import a SKILL.md to set up test data", func() {
			skillDir := filepath.Join(tempDir, "code-review")
			gomega.Expect(os.MkdirAll(skillDir, 0o755)).To(gomega.Succeed())
			gomega.Expect(os.WriteFile(filepath.Join(skillDir, "SKILL.md"), testdata.SkillMarkdown, 0o600)).To(gomega.Succeed())

			// Dummy config satisfies enricher validation; the actual enricher is replaced via factory.
			enrichCfg := filepath.Join(tempDir, "mcphost.json")
			gomega.Expect(os.WriteFile(enrichCfg, []byte(`{}`), 0o600)).To(gomega.Succeed())

			cidFile := filepath.Join(tempDir, "imported.cids")

			cli.Import("agent-skill", skillDir).WithArgs("--enrich-config="+enrichCfg, "--output-cids="+cidFile).ShouldSucceed()

			cidData, err := os.ReadFile(cidFile)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			cid = strings.TrimSpace(string(cidData))
			gomega.Expect(cid).NotTo(gomega.BeEmpty(), "imported CID should not be empty")
		})

		ginkgo.It("should export the record as SKILL.md to stdout", func() {
			output := cli.Export(cid).WithArgs("--format", "agent-skill").ShouldSucceed()
			gomega.Expect(output).NotTo(gomega.BeEmpty())
			gomega.Expect(output).To(gomega.Equal(strings.TrimSpace(string(testdata.SkillMarkdown))),
				"exported SKILL.md should match the original input")
		})

		ginkgo.It("should export the record as SKILL.md to a file", func() {
			outPath := filepath.Join(tempDir, "SKILL.md")

			cli.Export(cid).WithArgs("--format", "agent-skill", "--output-file", outPath).ShouldSucceed()

			data, err := os.ReadFile(outPath)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(string(data)).To(gomega.ContainSubstring("name: code-review"))
		})

		ginkgo.It("should auto-append .md extension when omitted", func() {
			outPath := filepath.Join(tempDir, "skill_no_ext")
			expectedPath := outPath + ".md"

			cli.Export(cid).WithArgs("--format", "agent-skill", "--output-file", outPath).ShouldSucceed()

			_, err := os.Stat(expectedPath)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(),
				"file with auto-appended .md extension should exist")

			data, err := os.ReadFile(expectedPath)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(string(data)).To(gomega.ContainSubstring("name: code-review"))
		})

		ginkgo.It("should clean up the test record", func() {
			cli.Delete(cid).ShouldSucceed()
		})
	})
})
