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

// Using the shared record data from embed.go
//
//nolint:dupl
var _ = ginkgo.Describe("Running dirctl end-to-end tests to check search functionality", func() {
	var cli *utils.CLI

	ginkgo.BeforeEach(func() {
		if cfg.DeploymentMode != config.DeploymentModeLocal {
			ginkgo.Skip("Skipping test, not in local mode")
		}

		utils.ResetCLIState()
		// Initialize CLI helper
		cli = utils.NewCLI()
	})

	// Test params
	var (
		tempDir    string
		recordPath string
		recordCID  string
	)

	ginkgo.Context("wildcard search functionality", ginkgo.Ordered, func() {
		// Setup: Create temporary directory and push a test record
		ginkgo.BeforeAll(func() {
			var err error
			tempDir, err = os.MkdirTemp("", "search-test")
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			recordPath = filepath.Join(tempDir, "record.json")

			// Write test record to temp location
			err = os.WriteFile(recordPath, testdata.ExpectedRecordV070JSON, 0o600)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Push the record to the store for searching
			recordCID = cli.Push(recordPath).WithArgs("--raw").ShouldSucceed()
			gomega.Expect(recordCID).NotTo(gomega.BeEmpty())
		})

		// Cleanup: Remove temporary directory after all tests
		ginkgo.AfterAll(func() {
			if tempDir != "" {
				err := os.RemoveAll(tempDir)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
			}
		})

		ginkgo.Context("exact match searches (no wildcards)", func() {
			ginkgo.It("should find record by exact name match", func() {
				output := cli.Search().
					WithQuery("name", "directory.agntcy.org/cisco/marketing-strategy-v3").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("should find record by exact version match", func() {
				output := cli.Search().
					WithQuery("version", "v3.0.0").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("should find record by exact skill name match", func() {
				output := cli.Search().
					WithQuery("skill-name", "natural_language_processing/natural_language_generation/text_completion").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("should find record by exact skill ID match", func() {
				output := cli.Search().
					WithQuery("skill-id", "10201").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("should find record by exact locator match", func() {
				output := cli.Search().
					WithQuery("locator", "docker_image:https://ghcr.io/agntcy/marketing-strategy").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("should find record by exact module name match", func() {
				output := cli.Search().
					WithQuery("module", "license").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})
		})

		ginkgo.Context("wildcard searches with * pattern", func() {
			ginkgo.Context("name field wildcards", func() {
				ginkgo.It("should find record with name prefix wildcard", func() {
					output := cli.Search().
						WithQuery("name", "directory.agntcy.org/cisco/*").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})

				ginkgo.It("should find record with name suffix wildcard", func() {
					output := cli.Search().
						WithQuery("name", "*marketing-strategy-v3").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})

				ginkgo.It("should find record with name middle wildcard", func() {
					output := cli.Search().
						WithQuery("name", "directory.agntcy.org/*/marketing-strategy-v3").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})

				ginkgo.It("should find record with multiple wildcards in name", func() {
					output := cli.Search().
						WithQuery("name", "*cisco*strategy*").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})
			})

			ginkgo.Context("version field wildcards", func() {
				ginkgo.It("should find record with version prefix wildcard", func() {
					output := cli.Search().
						WithQuery("version", "v3.*").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})

				ginkgo.It("should find record with version suffix wildcard", func() {
					output := cli.Search().
						WithQuery("version", "*.0.0").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})

				ginkgo.It("should find record with version middle wildcard", func() {
					output := cli.Search().
						WithQuery("version", "v*0.0").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})
			})

			ginkgo.Context("skill name wildcards", func() {
				ginkgo.It("should find record with skill name prefix wildcard", func() {
					output := cli.Search().
						WithQuery("skill-name", "natural_language*").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})

				ginkgo.It("should find record with skill name suffix wildcard", func() {
					output := cli.Search().
						WithQuery("skill-name", "*Completion").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})

				ginkgo.It("should find record with skill name middle wildcard", func() {
					output := cli.Search().
						WithQuery("skill-name", "Natural*Processing*Text*").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})

				ginkgo.It("should find record with different skill using wildcard", func() {
					output := cli.Search().
						WithQuery("skill-name", "*problem_solving").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})
			})

			ginkgo.Context("locator wildcards", func() {
				ginkgo.It("should find record with locator prefix wildcard", func() {
					output := cli.Search().
						WithQuery("locator", "docker_image:*").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})

				ginkgo.It("should find record with locator suffix wildcard", func() {
					output := cli.Search().
						WithQuery("locator", "*marketing-strategy").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})

				ginkgo.It("should find record with locator middle wildcard", func() {
					output := cli.Search().
						WithQuery("locator", "docker_image:*ghcr.io*marketing-strategy").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})

				ginkgo.It("should find record with protocol wildcard", func() {
					output := cli.Search().
						WithQuery("locator", "*://ghcr.io/agntcy/marketing-strategy").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})
			})

			ginkgo.Context("module wildcards", func() {
				ginkgo.It("should find record with module name prefix wildcard", func() {
					output := cli.Search().
						WithQuery("module", "license*").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})

				ginkgo.It("should find record with module name suffix wildcard", func() {
					output := cli.Search().
						WithQuery("module", "*framework*").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})

				ginkgo.It("should find record with schema module wildcard", func() {
					output := cli.Search().
						WithQuery("module", "*runtime*").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})

				ginkgo.It("should find record with module wildcard", func() {
					output := cli.Search().
						WithQuery("module", "*").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})
			})
		})

		ginkgo.Context("wildcard searches with ? pattern", func() {
			ginkgo.Context("version field question mark wildcards", func() {
				ginkgo.It("should find record with single character version wildcard", func() {
					output := cli.Search().
						WithQuery("version", "v?.0.0").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})

				ginkgo.It("should find record with multiple question mark wildcards in version", func() {
					output := cli.Search().
						WithQuery("version", "v?.?.?").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})

				ginkgo.It("should find record with question mark in middle of version", func() {
					output := cli.Search().
						WithQuery("version", "v3.?.0").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})
			})

			ginkgo.Context("name field question mark wildcards", func() {
				ginkgo.It("should find record with question mark in name", func() {
					output := cli.Search().
						WithQuery("name", "directory.agntcy.org/cisco/marketing-strategy-v?").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})

				ginkgo.It("should find record with multiple question marks in name", func() {
					output := cli.Search().
						WithQuery("name", "directory.agntcy.org/????o/marketing-strategy-v3").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})

				ginkgo.It("should find record with question mark at beginning of name segment", func() {
					output := cli.Search().
						WithQuery("name", "directory.agntcy.org/?isco/marketing-strategy-v3").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})
			})

			ginkgo.Context("skill name question mark wildcards", func() {
				ginkgo.It("should find record with question mark in skill name", func() {
					output := cli.Search().
						WithQuery("skill-name", "natural_language_processing/natural_language_generation/text_completio?").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})

				ginkgo.It("should find record with question mark replacing single word character", func() {
					output := cli.Search().
						WithQuery("skill-name", "?atural_language_processing/natural_language_generation/text_completion").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})

				ginkgo.It("should find record with multiple question marks in skill name", func() {
					output := cli.Search().
						WithQuery("skill-name", "natural_langua??_processing/natural_language_generation/text_completion").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})
			})

			ginkgo.Context("locator question mark wildcards", func() {
				ginkgo.It("should find record with question mark in protocol", func() {
					output := cli.Search().
						WithQuery("locator", "docker_image:http?://ghcr.io/agntcy/marketing-strategy").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})

				ginkgo.It("should find record with question mark in domain", func() {
					output := cli.Search().
						WithQuery("locator", "docker_image:https://ghcr.i?/agntcy/marketing-strategy").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})

				ginkgo.It("should find record with question mark in path", func() {
					output := cli.Search().
						WithQuery("locator", "docker_image:https://ghcr.io/agntcy/marketing-strateg?").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})
			})

			ginkgo.Context("module question mark wildcards", func() {
				ginkgo.It("should find record with question mark in module name", func() {
					output := cli.Search().
						WithQuery("module", "licens?").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})
			})

			ginkgo.Context("mixed ? and * wildcard patterns", func() {
				ginkgo.It("should find record with both wildcards in version", func() {
					output := cli.Search().
						WithQuery("version", "v?.*").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})

				ginkgo.It("should find record with both wildcards in name", func() {
					output := cli.Search().
						WithQuery("name", "*cisco/marketing-strategy-v?").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})

				ginkgo.It("should find record with both wildcards in skill name", func() {
					output := cli.Search().
						WithQuery("skill-name", "natural*processing/natural_language_generation/text_completio?").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})

				ginkgo.It("should find record with complex wildcard combination", func() {
					output := cli.Search().
						WithQuery("locator", "*://ghcr.i?/*/marketing-strateg?").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})
			})
		})

		ginkgo.Context("wildcard searches with [] list patterns", func() {
			ginkgo.Context("version field list wildcards", func() {
				ginkgo.It("should find record with numeric range in version", func() {
					output := cli.Search().
						WithQuery("version", "v[0-9].0.0").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})

				ginkgo.It("should find record with specific digit list in version", func() {
					output := cli.Search().
						WithQuery("version", "v[123].0.0").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})

				ginkgo.It("should find record with negated character class in version", func() {
					output := cli.Search().
						WithQuery("version", "v[^0-2].0.0").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})
			})

			ginkgo.Context("name field list wildcards", func() {
				ginkgo.It("should find record with character list in name", func() {
					output := cli.Search().
						WithQuery("name", "directory.agntcy.org/[abc]isco/marketing-strategy-v3").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})

				ginkgo.It("should find record with alphabetic range in name", func() {
					output := cli.Search().
						WithQuery("name", "directory.agntcy.org/[a-z]isco/marketing-strategy-v3").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})

				ginkgo.It("should find record with negated range in name", func() {
					output := cli.Search().
						WithQuery("name", "directory.agntcy.org/[^xyz]isco/marketing-strategy-v3").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})
			})

			ginkgo.Context("skill name list wildcards", func() {
				ginkgo.It("should find record with character list in skill name", func() {
					output := cli.Search().
						WithQuery("skill-name", "[mn]atural_language_processing/natural_language_generation/text_completion").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})

				ginkgo.It("should find record with alphabetic range in skill name", func() {
					output := cli.Search().
						WithQuery("skill-name", "[A-Z]atural_language_processing/natural_language_generation/text_completion").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})

				ginkgo.It("should find record with negated character class in skill name", func() {
					output := cli.Search().
						WithQuery("skill-name", "natural_language_processing/natural_language_generation/text_[^D-Z]ompletion").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})
			})

			ginkgo.Context("locator list wildcards", func() {
				ginkgo.It("should find record with character list in protocol", func() {
					output := cli.Search().
						WithQuery("locator", "docker_image:[ht]ttps://ghcr.io/agntcy/marketing-strategy").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})

				ginkgo.It("should find record with alphabetic range in domain", func() {
					output := cli.Search().
						WithQuery("locator", "docker_image:https://[a-z]hcr.io/agntcy/marketing-strategy").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})

				ginkgo.It("should find record with negated range in path", func() {
					output := cli.Search().
						WithQuery("locator", "docker_image:https://ghcr.io/agntcy/marketing-strateg[^0-9]").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})
			})

			ginkgo.Context("module list wildcards", func() {
				ginkgo.It("should find record with character list in module name", func() {
					output := cli.Search().
						WithQuery("module", "[l]icense").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})

				ginkgo.It("should find record with alphabetic range in module name", func() {
					output := cli.Search().
						WithQuery("module", "[a-z]icense").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})
			})

			ginkgo.Context("mixed list wildcards with other patterns", func() {
				ginkgo.It("should find record with list and asterisk wildcards", func() {
					output := cli.Search().
						WithQuery("name", "*[c]isco*").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})

				ginkgo.It("should find record with list and question mark wildcards", func() {
					output := cli.Search().
						WithQuery("version", "v[0-9].?.0").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})

				ginkgo.It("should find record with all wildcard types combined", func() {
					output := cli.Search().
						WithQuery("name", "*[c]isco/marketing-strategy-v?").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})

				ginkgo.It("should find record with multiple list wildcards", func() {
					output := cli.Search().
						WithQuery("locator", "docker_image:https://[g]hcr.io/agntcy/marketing-strateg[y]").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})
			})

			ginkgo.Context("complex list wildcard patterns", func() {
				ginkgo.It("should find record with alphanumeric range", func() {
					output := cli.Search().
						WithQuery("name", "directory.agntcy.org/[a-zA-Z0-9]isco/marketing-strategy-v3").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})

				ginkgo.It("should find record with mixed character classes", func() {
					output := cli.Search().
						WithQuery("skill-name", "[A-Z]atural_[A-Z]anguage_[A-Z]rocessing/natural_language_generation/text_[A-Z]ompletion").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})

				ginkgo.It("should find record with complex negated pattern", func() {
					output := cli.Search().
						WithQuery("locator", "docker_image:https://ghcr.io/agntcy/marketing-strateg[^0-9xz]").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})
			})
		})

		ginkgo.Context("complex wildcard combinations", func() {
			ginkgo.It("should find record with multiple filter types using wildcards", func() {
				output := cli.Search().
					WithQuery("name", "*cisco*").
					WithQuery("version", "v3.*").
					WithQuery("skill-name", "*language*").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("should find record mixing exact and wildcard filters", func() {
				output := cli.Search().
					WithQuery("skill-id", "10201").
					WithQuery("name", "*marketing-strategy*").
					WithQuery("locator", "docker_image:*").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("should handle search with limit and wildcard", func() {
				output := cli.Search().
					WithQuery("name", "*cisco*").
					WithLimit(5).
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("should handle search with offset and wildcard", func() {
				output := cli.Search().
					WithQuery("version", "v*").
					WithOffset(0).
					WithLimit(10).
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("should find record with question mark and asterisk wildcards combined", func() {
				output := cli.Search().
					WithQuery("name", "*cisco*").
					WithQuery("version", "v?.0.0").
					WithQuery("skill-name", "Natural*Completio?").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("should find record mixing exact, asterisk and question mark filters", func() {
				output := cli.Search().
					WithQuery("skill-id", "10201").
					WithQuery("name", "*marketing-strategy-v?").
					WithQuery("locator", "docker_image:http?://*").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("should find record with all wildcard types combined", func() {
				output := cli.Search().
					WithQuery("name", "*[c]isco*").
					WithQuery("version", "v[0-9].?.0").
					WithQuery("skill-name", "[A-Z]atural*processing*").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("should find record mixing exact and list wildcard filters", func() {
				output := cli.Search().
					WithQuery("skill-id", "10201").
					WithQuery("name", "*marketing-strategy-v[0-9]").
					WithQuery("locator", "docker_image:https://[a-z]hcr.io/*").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})
		})

		ginkgo.Context("negative wildcard tests", func() {
			ginkgo.It("should return no results for non-matching wildcard pattern", func() {
				output := cli.Search().
					WithQuery("name", "nonexistent*pattern").
					ShouldSucceed()
				gomega.Expect(output).NotTo(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("should return no results for wildcard with no matches", func() {
				output := cli.Search().
					WithQuery("version", "v99.*").
					ShouldSucceed()
				gomega.Expect(output).NotTo(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("should return no results when combining conflicting filters", func() {
				output := cli.Search().
					WithQuery("name", "*cisco*").
					WithQuery("version", "v1.*"). // Record has v3.0.0
					ShouldSucceed()
				gomega.Expect(output).NotTo(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("should return no results for non-matching question mark pattern", func() {
				output := cli.Search().
					WithQuery("version", "v?.9.9"). // Record has v3.0.0
					ShouldSucceed()
				gomega.Expect(output).NotTo(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("should return no results for question mark requiring exact length", func() {
				output := cli.Search().
					WithQuery("name", "directory.agntcy.org/cisco/marketing-strategy-v??"). // v3 is only 1 char
					ShouldSucceed()
				gomega.Expect(output).NotTo(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("should return no results for conflicting question mark and exact filters", func() {
				output := cli.Search().
					WithQuery("version", "v?.0.0").
					WithQuery("version", "v2.0.0"). // Record has v3.0.0, not v2.0.0
					ShouldSucceed()
				gomega.Expect(output).NotTo(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("should return no results for non-matching list wildcard pattern", func() {
				output := cli.Search().
					WithQuery("version", "v[0-2].0.0"). // Record has v3.0.0, 3 is not in [0-2]
					ShouldSucceed()
				gomega.Expect(output).NotTo(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("should return no results for negated character class that excludes match", func() {
				output := cli.Search().
					WithQuery("version", "v[^3].0.0"). // Record has v3.0.0, but [^3] excludes 3
					ShouldSucceed()
				gomega.Expect(output).NotTo(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("should return no results for list wildcard with wrong character set", func() {
				output := cli.Search().
					WithQuery("name", "directory.agntcy.org/[xyz]isco/marketing-strategy-v3"). // 'c' not in [xyz]
					ShouldSucceed()
				gomega.Expect(output).NotTo(gomega.ContainSubstring(recordCID))
			})
		})

		ginkgo.Context("edge cases and special characters", func() {
			ginkgo.It("should handle wildcard at the beginning and end", func() {
				output := cli.Search().
					WithQuery("name", "*marketing-strategy-v3*").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("should handle single wildcard matching everything", func() {
				output := cli.Search().
					WithQuery("name", "*").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("should handle wildcards with special characters in URL", func() {
				output := cli.Search().
					WithQuery("locator", "*://ghcr.io/*").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("should handle wildcards with dots and slashes", func() {
				output := cli.Search().
					WithQuery("module", "runtime/*").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("should handle question mark with dots in version", func() {
				output := cli.Search().
					WithQuery("version", "v3.?.0").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("should handle question mark with special characters in URLs", func() {
				output := cli.Search().
					WithQuery("locator", "docker_image:https://ghcr.i?/agntcy/marketing-strategy").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("should handle single question mark in various positions", func() {
				output := cli.Search().
					WithQuery("name", "directory.agntcy.org/cisco/marketing-strategy-v?").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("should handle mixed wildcards with complex patterns", func() {
				output := cli.Search().
					WithQuery("locator", "*://ghcr.i?/*/marketing-strateg?").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("should handle list wildcards with slashes", func() {
				output := cli.Search().
					WithQuery("module", "runtime/framework").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("should handle list wildcards with special URL characters", func() {
				output := cli.Search().
					WithQuery("locator", "docker_image:https://[a-z]hcr.io/agntcy/marketing-strategy").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("should handle multiple list wildcards in single pattern", func() {
				output := cli.Search().
					WithQuery("name", "directory.agntcy.org/[c]isco/marketing-strategy-v[0-9]").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("should handle list wildcards with all wildcard types", func() {
				output := cli.Search().
					WithQuery("locator", "*://[a-z]hcr.i?/*/marketing-strateg[y]").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})
		})
	})
})
