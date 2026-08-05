// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/agntcy/dir/tests/e2e/shared/testdata"
	"github.com/agntcy/dir/tests/e2e/shared/utils"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

// pushUniqueRecord pushes a copy of the sample record renamed with suffix, so
// that every record in this suite has a CID that no other suite touches.
func pushUniqueRecord(tempDir, suffix string) string {
	var record map[string]any

	gomega.Expect(json.Unmarshal(testdata.ExpectedRecordV100JSON, &record)).To(gomega.Succeed())

	record["name"] = fmt.Sprintf("delete_e2e_%s_agent", suffix)

	data, err := json.Marshal(record)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	path := filepath.Join(tempDir, "record_"+suffix+".json")
	gomega.Expect(os.WriteFile(path, data, 0o600)).To(gomega.Succeed())

	cid := testEnv.CLI.Push(path).WithArgs("--output", "raw").ShouldSucceed()
	gomega.Expect(cid).NotTo(gomega.BeEmpty())

	return cid
}

var _ = ginkgo.Describe("Running dirctl end-to-end tests for the delete command", func() {
	ginkgo.BeforeEach(func() {
		utils.ResetCLIState()
	})

	ginkgo.Context("Batch delete", ginkgo.Ordered, ginkgo.Serial, func() {
		var (
			tempDir string
			cids    map[string]string
		)

		ginkgo.BeforeAll(func() {
			var err error

			tempDir, err = os.MkdirTemp("", "delete-e2e-*")
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			cids = map[string]string{}
			for _, suffix := range []string{"single", "args_a", "args_b", "json_a", "json_b", "lines_a", "lines_b"} {
				cids[suffix] = pushUniqueRecord(tempDir, suffix)
			}
		})

		ginkgo.AfterAll(func() {
			for _, cid := range cids {
				_, _ = testEnv.CLI.Delete(cid).SuppressStderr().Execute()
			}

			if tempDir != "" {
				_ = os.RemoveAll(tempDir)
			}
		})

		ginkgo.It("should delete a single record and echo its CID in raw output", func() {
			output := testEnv.CLI.Delete(cids["single"]).WithArgs("--output", "raw").ShouldSucceed()
			gomega.Expect(strings.TrimSpace(output)).To(gomega.Equal(cids["single"]))

			_ = testEnv.CLI.Pull(cids["single"]).ShouldFail()
		})

		ginkgo.It("should delete multiple records passed as arguments", func() {
			output := testEnv.CLI.Delete(cids["args_a"], cids["args_b"]).ShouldSucceed()
			gomega.Expect(output).To(gomega.ContainSubstring(cids["args_a"]))
			gomega.Expect(output).To(gomega.ContainSubstring(cids["args_b"]))

			_ = testEnv.CLI.Pull(cids["args_a"]).ShouldFail()
			_ = testEnv.CLI.Pull(cids["args_b"]).ShouldFail()
		})

		ginkgo.It("should delete records from a JSON array on stdin", func() {
			input, err := json.Marshal([]string{cids["json_a"], cids["json_b"]})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			testEnv.CLI.DeleteFromStdin(string(input)).ShouldSucceed()

			_ = testEnv.CLI.Pull(cids["json_a"]).ShouldFail()
			_ = testEnv.CLI.Pull(cids["json_b"]).ShouldFail()
		})

		ginkgo.It("should delete records from line-delimited CIDs on stdin", func() {
			input := cids["lines_a"] + "\n" + cids["lines_b"] + "\n"

			testEnv.CLI.DeleteFromStdin(input).ShouldSucceed()

			_ = testEnv.CLI.Pull(cids["lines_a"]).ShouldFail()
			_ = testEnv.CLI.Pull(cids["lines_b"]).ShouldFail()
		})

		ginkgo.It("should fail when no CID is provided", func() {
			err := testEnv.CLI.Command("delete").ShouldFail()
			gomega.Expect(err.Error()).To(gomega.ContainSubstring("requires at least 1 arg"))
		})

		ginkgo.It("should fail when arguments are combined with --stdin", func() {
			err := testEnv.CLI.Delete(cids["single"]).WithArgs("--stdin").ShouldFail()
			gomega.Expect(err.Error()).To(gomega.ContainSubstring("accepts at most 0 arg"))
		})
	})
})
