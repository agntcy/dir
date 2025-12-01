// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"os"
	"path/filepath"

	"github.com/agntcy/dir/e2e/shared/config"
	"github.com/agntcy/dir/e2e/shared/testdata"
	"github.com/agntcy/dir/e2e/shared/utils"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

// Test data for OASF 0.8.0 record:
//   - name: "directory.agntcy.org/example/research-assistant-v4"
//   - version: "v4.0.0"
//   - schema_version: "0.8.0"
//   - authors: ["AGNTCY Contributors"]
//   - created_at: "2025-03-19T17:06:37Z"
//   - skills: [10201: "natural_language_processing/.../text_completion", 10702: ".../problem_solving"]
//   - locators: [docker_image: "https://ghcr.io/agntcy/research-assistant"]
//   - domains: [301: "life_science/biotechnology"]
//   - modules: ["license", "runtime/framework", "runtime/language"]

var _ = ginkgo.Describe("Search functionality for OASF 0.8.0 records", func() {
	var cli *utils.CLI

	ginkgo.BeforeEach(func() {
		if cfg.DeploymentMode != config.DeploymentModeLocal {
			ginkgo.Skip("Skipping test, not in local mode")
		}

		utils.ResetCLIState()
		cli = utils.NewCLI()
	})

	var (
		tempDir    string
		recordPath string
		recordCID  string
	)

	ginkgo.Context("search cids command", ginkgo.Ordered, func() {
		ginkgo.BeforeAll(func() {
			var err error
			tempDir, err = os.MkdirTemp("", "search-test")
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			recordPath = filepath.Join(tempDir, "record_080.json")
			err = os.WriteFile(recordPath, testdata.ExpectedRecordV080JSON, 0o600)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			recordCID = cli.Push(recordPath).WithArgs("--output", "raw").ShouldSucceed()
			gomega.Expect(recordCID).NotTo(gomega.BeEmpty())
		})

		ginkgo.AfterAll(func() {
			if tempDir != "" {
				_ = os.RemoveAll(tempDir)
			}
		})

		// Basic exact match searches
		ginkgo.Context("exact match searches", func() {
			ginkgo.It("finds record by exact name", func() {
				output := cli.Search().
					WithName("directory.agntcy.org/example/research-assistant-v4").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("finds record by exact version", func() {
				output := cli.Search().
					WithVersion("v4.0.0").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("finds record by skill ID", func() {
				output := cli.Search().
					WithSkillID("10201").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("finds record by skill name", func() {
				output := cli.Search().
					WithSkillName("natural_language_processing/natural_language_generation/text_completion").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("finds record by locator", func() {
				output := cli.Search().
					WithLocator("docker_image:https://ghcr.io/agntcy/research-assistant").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("finds record by domain name", func() {
				output := cli.Search().
					WithDomain("life_science/biotechnology").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("finds record by domain ID", func() {
				output := cli.Search().
					WithDomainID("301").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("finds record by module name", func() {
				output := cli.Search().
					WithModule("runtime/framework").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("finds record by author", func() {
				output := cli.Search().
					WithAuthor("AGNTCY Contributors").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("finds record by schema version", func() {
				output := cli.Search().
					WithSchemaVersion("0.8.0").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("finds record by created_at", func() {
				output := cli.Search().
					WithCreatedAt("2025-03-19T17:06:37Z").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})
		})

		// Wildcard searches with * pattern
		ginkgo.Context("asterisk wildcard searches", func() {
			ginkgo.It("finds record with name suffix wildcard", func() {
				output := cli.Search().
					WithName("*research-assistant-v4").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("finds record with name prefix wildcard", func() {
				output := cli.Search().
					WithName("directory.agntcy.org/*").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("finds record with name middle wildcard", func() {
				output := cli.Search().
					WithName("*example*assistant*").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("finds record with version wildcard", func() {
				output := cli.Search().
					WithVersion("v4.*").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("finds record with skill name wildcard", func() {
				output := cli.Search().
					WithSkillName("*text_completion").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("finds record with locator wildcard", func() {
				output := cli.Search().
					WithLocator("*research-assistant").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("finds record with domain wildcard", func() {
				output := cli.Search().
					WithDomain("life_science/*").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("finds record with author wildcard", func() {
				output := cli.Search().
					WithAuthor("*Contributors").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("finds record with schema version wildcard", func() {
				output := cli.Search().
					WithSchemaVersion("0.8.*").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})
		})

		// Wildcard searches with ? pattern
		ginkgo.Context("question mark wildcard searches", func() {
			ginkgo.It("finds record with single char version wildcard", func() {
				output := cli.Search().
					WithVersion("v?.0.0").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("finds record with question mark in name", func() {
				output := cli.Search().
					WithName("directory.agntcy.org/example/research-assistant-v?").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("finds record with mixed wildcards", func() {
				output := cli.Search().
					WithName("*research-assistant-v?").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})
		})

		// Wildcard searches with [] character class
		ginkgo.Context("character class wildcard searches", func() {
			ginkgo.It("finds record with numeric range in version", func() {
				output := cli.Search().
					WithVersion("v[0-9].0.0").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("finds record with character list in name", func() {
				output := cli.Search().
					WithName("directory.agntcy.org/[e]xample/research-assistant-v4").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("finds record with negated character class", func() {
				output := cli.Search().
					WithVersion("v[^0-3].0.0"). // v4.0.0, 4 is not in [0-3]
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("finds record with all wildcard types combined", func() {
				output := cli.Search().
					WithName("*[e]xample/research-assistant-v?").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})
		})

		// Multiple filter types (AND logic between different fields)
		ginkgo.Context("combined filters (AND logic)", func() {
			ginkgo.It("finds record with name and version", func() {
				output := cli.Search().
					WithName("*research-assistant*").
					WithVersion("v4.*").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("finds record with skill and domain", func() {
				output := cli.Search().
					WithSkillID("10201").
					WithDomain("*biotechnology").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("finds record with multiple filter types", func() {
				output := cli.Search().
					WithName("*research-assistant*").
					WithVersion("v4.*").
					WithSkillName("*text_completion").
					WithDomain("life_science/*").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("returns no results when filters conflict", func() {
				output := cli.Search().
					WithName("*research-assistant*").
					WithVersion("v1.*"). // Record has v4.0.0
					ShouldSucceed()
				gomega.Expect(output).NotTo(gomega.ContainSubstring(recordCID))
			})
		})

		// Multiple values for same field (OR logic)
		ginkgo.Context("multiple values for same field (OR logic)", func() {
			ginkgo.It("finds record when one of multiple versions matches", func() {
				output := cli.Search().
					WithVersion("v1.0.0").
					WithVersion("v4.0.0"). // This matches
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("finds record when one of multiple names matches", func() {
				output := cli.Search().
					WithName("nonexistent-agent").
					WithName("*research-assistant*"). // This matches
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("finds record when one of multiple skill IDs matches", func() {
				output := cli.Search().
					WithSkillID("99999").
					WithSkillID("10201"). // This matches
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})
		})

		// Negative tests
		ginkgo.Context("negative tests", func() {
			ginkgo.It("returns no results for non-matching name", func() {
				output := cli.Search().
					WithName("nonexistent-agent").
					ShouldSucceed()
				gomega.Expect(output).NotTo(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("returns no results for non-matching version", func() {
				output := cli.Search().
					WithVersion("v99.0.0").
					ShouldSucceed()
				gomega.Expect(output).NotTo(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("returns no results for non-matching wildcard", func() {
				output := cli.Search().
					WithName("*nonexistent*").
					ShouldSucceed()
				gomega.Expect(output).NotTo(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("returns no results for negated character class that excludes match", func() {
				output := cli.Search().
					WithVersion("v[^4].0.0"). // v4.0.0, but [^4] excludes 4
					ShouldSucceed()
				gomega.Expect(output).NotTo(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("returns no results for non-matching domain", func() {
				output := cli.Search().
					WithDomain("*healthcare*"). // Record has life_science/biotechnology
					ShouldSucceed()
				gomega.Expect(output).NotTo(gomega.ContainSubstring(recordCID))
			})
		})

		// Pagination
		ginkgo.Context("pagination", func() {
			ginkgo.It("respects limit parameter", func() {
				output := cli.Search().
					WithName("*research-assistant*").
					WithLimit(10).
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("respects offset parameter", func() {
				// With offset 0, we should find the record
				output := cli.Search().
					WithName("*research-assistant*").
					WithOffset(0).
					WithLimit(10).
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})
		})
	})
})
