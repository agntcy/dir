// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:dupl
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
//   - modules: [10201: "core/llm/model"]

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

		// Core exact match searches
		ginkgo.Context("exact match searches", func() {
			ginkgo.It("finds record by name", func() {
				output := cli.Search().
					WithName("directory.agntcy.org/example/research-assistant-v4").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("finds record by version", func() {
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
		})

		// Wildcard pattern searches
		ginkgo.Context("wildcard searches", func() {
			ginkgo.It("finds record with asterisk wildcard", func() {
				output := cli.Search().
					WithName("*research-assistant*").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("finds record with question mark wildcard", func() {
				output := cli.Search().
					WithVersion("v?.0.0").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("finds record with character class", func() {
				output := cli.Search().
					WithVersion("v[0-9].0.0").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("finds record with negated character class", func() {
				output := cli.Search().
					WithVersion("v[^0-3].0.0"). // v4.0.0, 4 is not in [0-3]
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("finds record with mixed wildcards", func() {
				output := cli.Search().
					WithName("*[e]xample/research-assistant-v?").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})
		})

		// Filter logic
		ginkgo.Context("filter logic", func() {
			ginkgo.It("applies AND logic between different fields", func() {
				output := cli.Search().
					WithName("*research-assistant*").
					WithVersion("v4.*").
					WithSkillID("10201").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("returns no results when AND filters conflict", func() {
				output := cli.Search().
					WithName("*research-assistant*").
					WithVersion("v1.*"). // Record has v4.0.0
					ShouldSucceed()
				gomega.Expect(output).NotTo(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("applies OR logic for multiple values of same field", func() {
				output := cli.Search().
					WithVersion("v1.0.0").
					WithVersion("v4.0.0"). // This matches
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})
		})

		// Negative tests
		ginkgo.Context("negative tests", func() {
			ginkgo.It("returns no results for non-matching query", func() {
				output := cli.Search().
					WithName("nonexistent-agent").
					ShouldSucceed()
				gomega.Expect(output).NotTo(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("returns no results for negated class that excludes match", func() {
				output := cli.Search().
					WithVersion("v[^4].0.0"). // v4.0.0, but [^4] excludes 4
					ShouldSucceed()
				gomega.Expect(output).NotTo(gomega.ContainSubstring(recordCID))
			})
		})

		// Pagination
		ginkgo.Context("pagination", func() {
			ginkgo.It("respects limit and offset parameters", func() {
				output := cli.Search().
					WithName("*research-assistant*").
					WithOffset(0).
					WithLimit(10).
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})
		})

		ginkgo.Context("comparison operators", func() {
			ginkgo.It("finds record with version >= v3.0.0", func() {
				output := cli.Search().
					WithVersion(">=v3.0.0").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("finds record with version <= v5.0.0", func() {
				output := cli.Search().
					WithVersion("<=v5.0.0").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("finds record with version > v3.0.0", func() {
				output := cli.Search().
					WithVersion(">v3.0.0").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("does not find record with version < v4.0.0", func() {
				output := cli.Search().
					WithVersion("<v4.0.0").
					ShouldSucceed()
				gomega.Expect(output).NotTo(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("finds record with version =v4.0.0", func() {
				output := cli.Search().
					WithVersion("=v4.0.0").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("finds record with schema-version >= 0.7.0", func() {
				output := cli.Search().
					WithSchemaVersion(">=0.7.0").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("does not find record with schema-version > 0.8.0", func() {
				output := cli.Search().
					WithSchemaVersion(">0.8.0").
					ShouldSucceed()
				gomega.Expect(output).NotTo(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("finds record with created-at >= 2025-01-01", func() {
				output := cli.Search().
					WithCreatedAt(">=2025-01-01").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("does not find record with created-at < 2025-01-01", func() {
				output := cli.Search().
					WithCreatedAt("<2025-01-01").
					ShouldSucceed()
				gomega.Expect(output).NotTo(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("finds record within version range", func() {
				output := cli.Search().
					WithVersion(">=v3.0.0").
					WithVersion("<=v5.0.0").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})
		})
	})

	ginkgo.Context("search records command", ginkgo.Ordered, func() {
		ginkgo.BeforeAll(func() {
			var err error
			tempDir, err = os.MkdirTemp("", "search-records-test")
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

		ginkgo.It("returns full record data with JSON output", func() {
			output := cli.SearchRecords().
				WithName("directory.agntcy.org/example/research-assistant-v4").
				WithArgs("--output", "json").
				ShouldSucceed()

			// Verify record fields are present in output
			gomega.Expect(output).To(gomega.ContainSubstring("research-assistant-v4"))
			gomega.Expect(output).To(gomega.ContainSubstring("v4.0.0"))
			gomega.Expect(output).To(gomega.ContainSubstring("0.8.0"))
			gomega.Expect(output).To(gomega.ContainSubstring("AGNTCY Contributors"))
		})

		ginkgo.It("returns record with skills data", func() {
			output := cli.SearchRecords().
				WithSkillID("10201").
				WithArgs("--output", "json").
				ShouldSucceed()

			gomega.Expect(output).To(gomega.ContainSubstring("text_completion"))
			gomega.Expect(output).To(gomega.ContainSubstring("10201"))
		})

		ginkgo.It("returns record with domain data", func() {
			output := cli.SearchRecords().
				WithDomain("life_science/*").
				WithArgs("--output", "json").
				ShouldSucceed()

			gomega.Expect(output).To(gomega.ContainSubstring("life_science"))
			gomega.Expect(output).To(gomega.ContainSubstring("biotechnology"))
			gomega.Expect(output).To(gomega.ContainSubstring("301"))
		})

		ginkgo.It("returns record with locator data", func() {
			output := cli.SearchRecords().
				WithLocator("*research-assistant").
				WithArgs("--output", "json").
				ShouldSucceed()

			gomega.Expect(output).To(gomega.ContainSubstring("docker_image"))
			gomega.Expect(output).To(gomega.ContainSubstring("ghcr.io/agntcy/research-assistant"))
		})

		ginkgo.It("returns record with module data", func() {
			output := cli.SearchRecords().
				WithModule("core/*").
				WithArgs("--output", "json").
				ShouldSucceed()

			gomega.Expect(output).To(gomega.ContainSubstring("core/llm/model"))
			gomega.Expect(output).To(gomega.ContainSubstring("gpt-4"))
		})

		ginkgo.It("supports wildcards like cids command", func() {
			output := cli.SearchRecords().
				WithName("*research-assistant-v?").
				WithArgs("--output", "json").
				ShouldSucceed()

			gomega.Expect(output).To(gomega.ContainSubstring("research-assistant-v4"))
		})

		ginkgo.It("returns no results for non-matching query", func() {
			output := cli.SearchRecords().
				WithName("nonexistent-agent").
				WithArgs("--output", "json").
				ShouldSucceed()

			// Should not contain our record data
			gomega.Expect(output).NotTo(gomega.ContainSubstring("research-assistant-v4"))
		})
	})
})
