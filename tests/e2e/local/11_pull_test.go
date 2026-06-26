// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/agntcy/dir/tests/e2e/shared/testdata"
	"github.com/agntcy/dir/tests/e2e/shared/utils"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Running dirctl end-to-end tests for the pull command", func() {
	ginkgo.BeforeEach(func() {
		utils.ResetCLIState()
	})

	ginkgo.Context("Pull output destinations", ginkgo.Ordered, ginkgo.Serial, func() {
		var cid string

		tempDir, err := os.MkdirTemp("", "pull-e2e-*")
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		ginkgo.AfterAll(func() {
			os.RemoveAll(tempDir)
		})

		ginkgo.It("should push a record to set up test data", func() {
			pushPath := filepath.Join(tempDir, "push_record.json")
			gomega.Expect(os.WriteFile(pushPath, testdata.ExpectedRecordV100JSON, 0o600)).To(gomega.Succeed())

			cid = testEnv.CLI.Push(pushPath).WithArgs("--output", "raw").ShouldSucceed()
			gomega.Expect(cid).NotTo(gomega.BeEmpty())
		})

		ginkgo.It("should pull a record to stdout as JSON by default", func() {
			output := testEnv.CLI.Pull(cid).ShouldSucceed()
			gomega.Expect(output).NotTo(gomega.BeEmpty())

			var parsed map[string]any

			err := json.Unmarshal([]byte(output), &parsed)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "default stdout output should be valid JSON")
			gomega.Expect(parsed).To(gomega.HaveKey("name"))
		})

		ginkgo.It("should pull a record to a file with --output-file", func() {
			outPath := filepath.Join(tempDir, "pulled_record.json")

			testEnv.CLI.Pull(cid).WithArgs("--output-file", outPath).ShouldSucceed()

			data, err := os.ReadFile(outPath)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(data).NotTo(gomega.BeEmpty())

			var parsed map[string]any

			err = json.Unmarshal(data, &parsed)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "file content should be valid JSON")
			gomega.Expect(parsed).To(gomega.HaveKey("name"))
		})

		ginkgo.It("should batch pull matching records to a directory", func() {
			outDir := filepath.Join(tempDir, "batch")

			testEnv.CLI.Command("pull").WithArgs(
				"--output-dir", outDir,
				"--module", "integration/a2a",
			).ShouldSucceed()

			entries, err := os.ReadDir(outDir)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(entries).NotTo(gomega.BeEmpty(), "output directory should contain pulled records")

			for _, entry := range entries {
				gomega.Expect(strings.HasSuffix(entry.Name(), ".json")).To(gomega.BeTrue(),
					"each pulled record should be a .json file")
			}
		})

		ginkgo.It("should fail when --output-dir and positional arg are both provided", func() {
			outDir := filepath.Join(tempDir, "fail-both")

			err := testEnv.CLI.Pull(cid).WithArgs("--output-dir", outDir, "--module", "integration/a2a").ShouldFail()
			gomega.Expect(err.Error()).To(gomega.ContainSubstring("mutually exclusive"))
		})

		ginkgo.It("should fail when --output-dir is used without search filters", func() {
			outDir := filepath.Join(tempDir, "fail-no-filter")

			err := testEnv.CLI.Command("pull").WithArgs("--output-dir", outDir).ShouldFail()
			gomega.Expect(err.Error()).To(gomega.ContainSubstring("at least one search filter"))
		})

		ginkgo.It("should clean up the test record", func() {
			testEnv.CLI.Delete(cid).ShouldSucceed()
		})
	})
})
