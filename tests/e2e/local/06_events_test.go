// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"os"

	"github.com/agntcy/dir/tests/e2e/shared/testdata"
	"github.com/agntcy/dir/tests/e2e/shared/utils"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

// Event CLI Tests
//
// Testing Strategy:
// - CLI tests (this file): Verify command existence, help text, flag acceptance
// - SDK tests (e2e/client/02_events_test.go): Full streaming event reception tests
//
// Rationale: The 'dirctl events listen' command runs as a long-running streaming process.
// The CLI test framework (utils.CLI) executes commands synchronously and captures output,
// which doesn't support background processes. Therefore:
// - We test CLI command structure here (help, flags, command registration)
// - We test actual event streaming in e2e/client/ using the SDK
//
// This matches the pattern for other streaming commands in the codebase.

var _ = ginkgo.Describe("Events CLI Commands", ginkgo.Serial, ginkgo.Label("events"), func() {
	ginkgo.BeforeEach(func() {
		utils.ResetCLIState()
	})

	ginkgo.Context("events command availability", func() {
		ginkgo.It("should have events command registered", func() {
			// Test that 'dirctl events' command exists
			output, err := testEnv.CLI.Command("events").WithArgs("--help").Execute()
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(output).To(gomega.ContainSubstring("Stream real-time events"))
			gomega.Expect(output).To(gomega.ContainSubstring("listen"))
		})

		ginkgo.It("should have events listen subcommand", func() {
			// Test that 'dirctl events listen' exists
			output, err := testEnv.CLI.Command("events").WithArgs("listen", "--help").Execute()
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(output).To(gomega.ContainSubstring("Listen to real-time system events"))
			gomega.Expect(output).To(gomega.ContainSubstring("--types"))
			gomega.Expect(output).To(gomega.ContainSubstring("--labels"))
			gomega.Expect(output).To(gomega.ContainSubstring("--cids"))
		})
	})

	ginkgo.Context("events listen command flags", func() {
		ginkgo.It("should support --types flag", func() {
			// Verify the --types flag exists in help
			output, err := testEnv.CLI.Command("events").WithArgs("listen", "--help").Execute()
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(output).To(gomega.ContainSubstring("--types"))
			gomega.Expect(output).To(gomega.ContainSubstring("Event types to filter"))
		})

		ginkgo.It("should support --labels flag", func() {
			// Verify the --labels flag exists in help
			output, err := testEnv.CLI.Command("events").WithArgs("listen", "--help").Execute()
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(output).To(gomega.ContainSubstring("--labels"))
			gomega.Expect(output).To(gomega.ContainSubstring("Label filters"))
		})

		ginkgo.It("should support --cids flag", func() {
			// Verify the --cids flag exists in help
			output, err := testEnv.CLI.Command("events").WithArgs("listen", "--help").Execute()
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(output).To(gomega.ContainSubstring("--cids"))
			gomega.Expect(output).To(gomega.ContainSubstring("CID filters"))
		})

		ginkgo.It("should support --output flag", func() {
			// Verify the --output flag exists (from AddOutputFlags)
			output, err := testEnv.CLI.Command("events").WithArgs("listen", "--help").Execute()
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(output).To(gomega.ContainSubstring("--output"))
		})
	})

	ginkgo.Context("event emission during operations", ginkgo.Ordered, func() {
		var pushCID, publishCID string

		ginkgo.It("should emit events during push operations", func() {
			// Push operation emits RECORD_PUSHED event
			// Full streaming reception tested in e2e/client/02_events_test.go
			recordFile, err := os.CreateTemp("", "record-*.json")
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Use V070 to get a different CID from routing test
			err = os.WriteFile(recordFile.Name(), testdata.ExpectedRecordV070JSON, 0o600)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			pushCID = testEnv.CLI.Push(recordFile.Name()).WithArgs("--output", "raw").ShouldSucceed()
			gomega.Expect(pushCID).NotTo(gomega.BeEmpty())
		})

		ginkgo.It("should emit events during publish operations", func() {
			// Push operation emits RECORD_PUSHED event
			// Full streaming reception tested in e2e/client/02_events_test.go
			recordFile, err := os.CreateTemp("", "record-*.json")
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Use V070 to get a different CID from routing test
			err = os.WriteFile(recordFile.Name(), testdata.ExpectedRecordV070JSON, 0o600)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			publishCID = testEnv.CLI.Push(recordFile.Name()).WithArgs("--output", "raw").ShouldSucceed()
			output := testEnv.CLI.Routing().Publish(publishCID).ShouldSucceed()
			gomega.Expect(output).To(gomega.ContainSubstring("Successfully submitted publication request"))
		})

		ginkgo.It("should emit events during delete operations", func() {
			// Delete the record from the first test (different from publish test)
			// Delete operation emits RECORD_DELETED event
			// Full streaming reception tested in e2e/client/02_events_test.go
			testEnv.CLI.Delete(pushCID).ShouldSucceed()

			// Verify delete worked
			_ = testEnv.CLI.Pull(pushCID).ShouldFail()
		})
	})
})
