// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"os"
	"path/filepath"

	"github.com/agntcy/dir/tests/e2e/shared/testdata"
	"github.com/agntcy/dir/tests/e2e/shared/utils"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

// Test data for OASF 0.8.0 record (record_080_v4.json):
//   - annotations: {"key": "value"}
//
// The annotation search feature stores annotations in an EAV table and supports
// key:value wildcard matching via the --annotation flag.

var _ = ginkgo.Describe("Annotation-based search", func() {
	ginkgo.BeforeEach(func() {
		utils.ResetCLIState()
	})

	var (
		tempDir   string
		recordCID string
	)

	ginkgo.Context("searching by annotation key:value", ginkgo.Ordered, func() {
		ginkgo.BeforeAll(func() {
			var err error

			tempDir, err = os.MkdirTemp("", "annotation-search-test")
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			recordPath := filepath.Join(tempDir, "record_080_v4.json")
			err = os.WriteFile(recordPath, testdata.ExpectedRecordV080V4JSON, 0o600)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			recordCID = testEnv.CLI.Push(recordPath).WithArgs("--output", "raw").ShouldSucceed()
			gomega.Expect(recordCID).NotTo(gomega.BeEmpty())
		})

		ginkgo.AfterAll(func() {
			if tempDir != "" {
				_ = os.RemoveAll(tempDir)
			}
		})

		// Exact match
		ginkgo.It("finds record by exact annotation key:value", func() {
			output := testEnv.CLI.Search().
				WithAnnotation("key:value").
				ShouldSucceed()
			gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
		})

		// Wildcard on value
		ginkgo.It("finds record with wildcard on annotation value", func() {
			output := testEnv.CLI.Search().
				WithAnnotation("key:val*").
				ShouldSucceed()
			gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
		})

		// Wildcard on value — single char
		ginkgo.It("finds record with ? wildcard on annotation value", func() {
			output := testEnv.CLI.Search().
				WithAnnotation("key:valu?").
				ShouldSucceed()
			gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
		})

		// Key only (no colon) — matches any value for that key
		ginkgo.It("finds record by annotation key only (no value)", func() {
			output := testEnv.CLI.Search().
				WithAnnotation("key").
				ShouldSucceed()
			gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
		})

		// Wildcard on key
		ginkgo.It("finds record with wildcard on annotation key", func() {
			output := testEnv.CLI.Search().
				WithAnnotation("ke*:value").
				ShouldSucceed()
			gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
		})

		// Combined with another filter (AND logic)
		ginkgo.It("finds record when annotation is combined with name filter", func() {
			output := testEnv.CLI.Search().
				WithName("*research-assistant*").
				WithAnnotation("key:value").
				ShouldSucceed()
			gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
		})

		// Negative: wrong value
		ginkgo.It("returns no results for non-matching annotation value", func() {
			output := testEnv.CLI.Search().
				WithAnnotation("key:nonexistent").
				ShouldSucceed()
			gomega.Expect(output).NotTo(gomega.ContainSubstring(recordCID))
		})

		// Negative: wrong key
		ginkgo.It("returns no results for non-matching annotation key", func() {
			output := testEnv.CLI.Search().
				WithAnnotation("nonexistent:value").
				ShouldSucceed()
			gomega.Expect(output).NotTo(gomega.ContainSubstring(recordCID))
		})

		// Repeated filters: one filter's key must not pair with another filter's value
		ginkgo.It("does not cross-match key and value across repeated annotation filters", func() {
			output := testEnv.CLI.Search().
				WithAnnotation("key:nonexistent").
				WithAnnotation("nonexistent:value").
				ShouldSucceed()
			gomega.Expect(output).NotTo(gomega.ContainSubstring(recordCID))
		})

		ginkgo.It("finds record by a key-only filter repeated with a non-matching key:value filter", func() {
			output := testEnv.CLI.Search().
				WithAnnotation("key").
				WithAnnotation("nonexistent:other").
				ShouldSucceed()
			gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
		})

		ginkgo.It("does not cross-match key and value across repeated exclude-annotation filters", func() {
			output := testEnv.CLI.Search().
				WithName("*research-assistant*").
				WithArgs("--exclude-annotation", "key:nonexistent", "--exclude-annotation", "nonexistent:value").
				ShouldSucceed()
			gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
		})

		// Negative: conflicting annotation + name filter
		ginkgo.It("returns no results when annotation matches but name does not", func() {
			output := testEnv.CLI.Search().
				WithName("nonexistent-agent").
				WithAnnotation("key:value").
				ShouldSucceed()
			gomega.Expect(output).NotTo(gomega.ContainSubstring(recordCID))
		})

		// format=record returns full record with annotation data
		ginkgo.It("returns full record data including annotations with format=record", func() {
			output := testEnv.CLI.SearchRecords().
				WithAnnotation("key:value").
				WithArgs("--output", "json").
				ShouldSucceed()
			gomega.Expect(output).To(gomega.ContainSubstring("key"))
			gomega.Expect(output).To(gomega.ContainSubstring("value"))
		})
	})
})
