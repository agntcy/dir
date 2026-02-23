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

var _ = ginkgo.Describe("Name resolution - pull by name", func() {
	var cli *utils.CLI

	ginkgo.BeforeEach(func() {
		if cfg.DeploymentMode != config.DeploymentModeLocal {
			ginkgo.Skip("Skipping test, not in local mode")
		}

		utils.ResetCLIState()

		cli = utils.NewCLI()
	})

	// Test pulling by name using existing test records
	ginkgo.Context("Pull by name", ginkgo.Ordered, func() {
		var (
			tempDir   string
			recordCID string
		)

		// Record name from record_080.json: "http://dns-validation-http/example/research-assistant-v4"
		const recordName = "http://dns-validation-http/example/research-assistant-v4"

		ginkgo.BeforeAll(func() {
			var err error

			tempDir, err = os.MkdirTemp("", "resolve-test")
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Write and push record_080.json
			recordPath := filepath.Join(tempDir, "record_080.json")
			err = os.WriteFile(recordPath, testdata.ExpectedRecordV080V4JSON, 0o600)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			recordCID = cli.Push(recordPath).WithArgs("--output", "raw").ShouldSucceed()
			gomega.Expect(recordCID).NotTo(gomega.BeEmpty())
		})

		ginkgo.AfterAll(func() {
			// Clean up pushed record
			if recordCID != "" {
				_, _ = cli.Delete(recordCID).Execute()
			}

			if tempDir != "" {
				_ = os.RemoveAll(tempDir)
			}
		})

		ginkgo.It("should pull record by full name", func() {
			output := cli.Pull(recordName).
				WithArgs("--output", "json").
				ShouldSucceed()

			gomega.Expect(output).To(gomega.ContainSubstring(recordName))
			gomega.Expect(output).To(gomega.ContainSubstring("v4.0.0"))
		})

		ginkgo.It("should pull record by name with version", func() {
			output := cli.Pull(recordName+":v4.0.0").
				WithArgs("--output", "json").
				ShouldSucceed()

			gomega.Expect(output).To(gomega.ContainSubstring(recordName))
			gomega.Expect(output).To(gomega.ContainSubstring("v4.0.0"))
		})

		ginkgo.It("should pull by CID directly", func() {
			output := cli.Pull(recordCID).
				WithArgs("--output", "json").
				ShouldSucceed()

			gomega.Expect(output).To(gomega.ContainSubstring(recordName))
		})

		ginkgo.It("should pull with hash verification (name@digest)", func() {
			output := cli.Pull(recordName+"@"+recordCID).
				WithArgs("--output", "json").
				ShouldSucceed()

			gomega.Expect(output).To(gomega.ContainSubstring(recordName))
		})

		ginkgo.It("should pull with hash verification (name:version@digest)", func() {
			output := cli.Pull(recordName+":v4.0.0@"+recordCID).
				WithArgs("--output", "json").
				ShouldSucceed()

			gomega.Expect(output).To(gomega.ContainSubstring(recordName))
		})

		ginkgo.It("should fail hash verification with wrong digest", func() {
			// Use a fake CID that won't match
			wrongDigest := "bafyreiaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
			_ = cli.Pull(recordName + "@" + wrongDigest).ShouldFail()
		})

		ginkgo.It("should fail for non-existent name", func() {
			_ = cli.Pull("nonexistent.example.com/agent").ShouldFail()
		})

		ginkgo.It("should fail for non-existent version", func() {
			_ = cli.Pull(recordName + ":v99.0.0").ShouldFail()
		})
	})

	// Test version resolution with multiple versions of the same record
	ginkgo.Context("Version resolution - latest by created_at", ginkgo.Ordered, func() {
		var (
			tempDir string
			cidV4   string
			cidV5   string
		)

		// Both records have the same name but different versions
		const recordName = "http://dns-validation-http/example/research-assistant-v4"

		ginkgo.BeforeAll(func() {
			var err error

			tempDir, err = os.MkdirTemp("", "version-resolve-test")
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Push v4.0.0 (from record_080.json)
			recordV4Path := filepath.Join(tempDir, "record_v4.json")
			err = os.WriteFile(recordV4Path, testdata.ExpectedRecordV080V4JSON, 0o600)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			cidV4 = cli.Push(recordV4Path).WithArgs("--output", "raw").ShouldSucceed()
			gomega.Expect(cidV4).NotTo(gomega.BeEmpty())

			// Push v5.0.0 (from record_080_v5.json)
			recordV5Path := filepath.Join(tempDir, "record_v5.json")
			err = os.WriteFile(recordV5Path, testdata.ExpectedRecordV080V5JSON, 0o600)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			cidV5 = cli.Push(recordV5Path).WithArgs("--output", "raw").ShouldSucceed()
			gomega.Expect(cidV5).NotTo(gomega.BeEmpty())

			// Ensure different CIDs (different content)
			gomega.Expect(cidV4).NotTo(gomega.Equal(cidV5))
		})

		ginkgo.AfterAll(func() {
			// Clean up pushed records
			if cidV4 != "" {
				_, _ = cli.Delete(cidV4).Execute()
			}

			if cidV5 != "" {
				_, _ = cli.Delete(cidV5).Execute()
			}

			if tempDir != "" {
				_ = os.RemoveAll(tempDir)
			}
		})

		ginkgo.It("should pull latest version (v5.0.0) when no version specified", func() {
			output := cli.Pull(recordName).
				WithArgs("--output", "json").
				ShouldSucceed()

			gomega.Expect(output).To(gomega.ContainSubstring("v5.0.0"))
		})

		ginkgo.It("should pull specific version v4.0.0 with name:version", func() {
			output := cli.Pull(recordName+":v4.0.0").
				WithArgs("--output", "json").
				ShouldSucceed()

			gomega.Expect(output).To(gomega.ContainSubstring("v4.0.0"))
		})

		ginkgo.It("should pull specific version v5.0.0 with name:version", func() {
			output := cli.Pull(recordName+":v5.0.0").
				WithArgs("--output", "json").
				ShouldSucceed()

			gomega.Expect(output).To(gomega.ContainSubstring("v5.0.0"))
		})

		ginkgo.It("should pull v4 by CID directly", func() {
			output := cli.Pull(cidV4).
				WithArgs("--output", "json").
				ShouldSucceed()

			gomega.Expect(output).To(gomega.ContainSubstring("v4.0.0"))
		})

		ginkgo.It("should pull v5 by CID directly", func() {
			output := cli.Pull(cidV5).
				WithArgs("--output", "json").
				ShouldSucceed()

			gomega.Expect(output).To(gomega.ContainSubstring("v5.0.0"))
		})

		ginkgo.It("should pull v4 with hash verification", func() {
			output := cli.Pull(recordName+":v4.0.0@"+cidV4).
				WithArgs("--output", "json").
				ShouldSucceed()

			gomega.Expect(output).To(gomega.ContainSubstring("v4.0.0"))
		})

		ginkgo.It("should pull v5 with hash verification", func() {
			output := cli.Pull(recordName+":v5.0.0@"+cidV5).
				WithArgs("--output", "json").
				ShouldSucceed()

			gomega.Expect(output).To(gomega.ContainSubstring("v5.0.0"))
		})

		ginkgo.It("should fail hash verification when version and digest mismatch", func() {
			// Try to pull v4 but provide v5's CID - should fail
			_ = cli.Pull(recordName + ":v4.0.0@" + cidV5).ShouldFail()
		})
	})

	// Test info command with name resolution
	ginkgo.Context("Info by name", ginkgo.Ordered, func() {
		var (
			tempDir   string
			recordCID string
		)

		const recordName = "http://dns-validation-http/example/research-assistant-v4"

		ginkgo.BeforeAll(func() {
			var err error

			tempDir, err = os.MkdirTemp("", "info-resolve-test")
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Write and push record
			recordPath := filepath.Join(tempDir, "record.json")
			err = os.WriteFile(recordPath, testdata.ExpectedRecordV080V4JSON, 0o600)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			recordCID = cli.Push(recordPath).WithArgs("--output", "raw").ShouldSucceed()
			gomega.Expect(recordCID).NotTo(gomega.BeEmpty())
		})

		ginkgo.AfterAll(func() {
			// Clean up pushed record
			if recordCID != "" {
				_, _ = cli.Delete(recordCID).Execute()
			}

			if tempDir != "" {
				_ = os.RemoveAll(tempDir)
			}
		})

		ginkgo.It("should get info by CID", func() {
			output := cli.Info(recordCID).
				WithArgs("--output", "json").
				ShouldSucceed()

			gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
		})

		ginkgo.It("should get info by name", func() {
			output := cli.Info(recordName).
				WithArgs("--output", "json").
				ShouldSucceed()

			gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
		})

		ginkgo.It("should get info by name with version", func() {
			output := cli.Info(recordName+":v4.0.0").
				WithArgs("--output", "json").
				ShouldSucceed()

			gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
		})

		ginkgo.It("should get info with hash verification", func() {
			output := cli.Info(recordName+"@"+recordCID).
				WithArgs("--output", "json").
				ShouldSucceed()

			gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
		})

		ginkgo.It("should fail info for non-existent name", func() {
			_ = cli.Info("nonexistent.example.com/agent").ShouldFail()
		})
	})
})
