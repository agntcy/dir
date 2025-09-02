// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"os"
	"path/filepath"

	"github.com/agntcy/dir/e2e/config"
	"github.com/agntcy/dir/e2e/utils"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

// Using the shared record V3 data from embed.go

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
			err = os.WriteFile(recordPath, expectedRecordV3JSON, 0o600)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Push the record to the store for searching
			recordCID = cli.Push(recordPath).ShouldSucceed()
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
					WithQuery("skill-name", "Natural Language Processing/Text Completion").
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
					WithQuery("locator", "docker-image:https://ghcr.io/agntcy/marketing-strategy").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("should find record by exact extension name match", func() {
				output := cli.Search().
					WithQuery("extension", "license:v1.0.0").
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
						WithQuery("skill-name", "Natural Language*").
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
						WithQuery("skill-name", "*Problem Solving").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})
			})

			ginkgo.Context("locator wildcards", func() {
				ginkgo.It("should find record with locator prefix wildcard", func() {
					output := cli.Search().
						WithQuery("locator", "docker-image:*").
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
						WithQuery("locator", "docker-image:*ghcr.io*marketing-strategy").
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

			ginkgo.Context("extension wildcards", func() {
				ginkgo.It("should find record with extension name prefix wildcard", func() {
					output := cli.Search().
						WithQuery("extension", "license*").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})

				ginkgo.It("should find record with extension name suffix wildcard", func() {
					output := cli.Search().
						WithQuery("extension", "*framework*").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})

				ginkgo.It("should find record with schema extension wildcard", func() {
					output := cli.Search().
						WithQuery("extension", "schema.oasf.agntcy.org*").
						ShouldSucceed()
					gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
				})

				ginkgo.It("should find record with extension version wildcard", func() {
					output := cli.Search().
						WithQuery("extension", "*:v1.*").
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
					WithQuery("skill-name", "*Language*").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})

			ginkgo.It("should find record mixing exact and wildcard filters", func() {
				output := cli.Search().
					WithQuery("skill-id", "10201").
					WithQuery("name", "*marketing-strategy*").
					WithQuery("locator", "docker-image:*").
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
					WithQuery("extension", "schema.oasf.agntcy.org/features/runtime/*").
					ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			})
		})
	})
})
