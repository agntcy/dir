// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package local

import (
	_ "embed"
	"os"
	"path/filepath"

	"github.com/agntcy/dir/e2e/shared/config"
	"github.com/agntcy/dir/e2e/shared/testdata"
	"github.com/agntcy/dir/e2e/shared/utils"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Running dirctl validation tests with different OASF validation configurations", func() {
	var cli *utils.CLI

	ginkgo.BeforeEach(func() {
		if cfg.DeploymentMode != config.DeploymentModeLocal {
			ginkgo.Skip("Skipping test, not in local mode")
		}

		utils.ResetCLIState()
		// Initialize CLI helper
		cli = utils.NewCLI()
	})

	// Setup temp directory for all tests
	tempDir := os.Getenv("E2E_COMPILE_OUTPUT_DIR")
	if tempDir == "" {
		tempDir = os.TempDir()
	}

	// Test cases for validation scenarios
	ginkgo.Context("Validation with strict mode disabled", ginkgo.Ordered, ginkgo.Serial, ginkgo.Label("validation-non-strict"), func() {
		// These tests should succeed when strict validation is disabled
		// because record_warnings_080.json contains warnings that would fail in strict mode

		var record080Path, recordWarnings080Path string

		ginkgo.BeforeAll(func() {
			// Create temporary files for test records
			record080Path = filepath.Join(tempDir, "record_080_validation_test.json")
			recordWarnings080Path = filepath.Join(tempDir, "record_warnings_080_validation_test.json")

			_ = os.MkdirAll(filepath.Dir(record080Path), 0o755)
			_ = os.WriteFile(record080Path, testdata.ExpectedRecordV080JSON, 0o600)
			_ = os.WriteFile(recordWarnings080Path, testdata.ExpectedRecordWarningsV080JSON, 0o600)
		})

		ginkgo.It("should successfully push record_080.json when strict validation is disabled", func() {
			cid := cli.Push(record080Path).WithArgs("--output", "raw").ShouldSucceed()
			gomega.Expect(cid).NotTo(gomega.BeEmpty(), "Push should return a valid CID")

			// Validate that the returned CID correctly represents the pushed data
			utils.LoadAndValidateCID(cid, record080Path)
		})

		ginkgo.It("should successfully push record_warnings_080.json when strict validation is disabled", func() {
			// This record contains warnings that would fail in strict mode
			// but should succeed when strict mode is disabled
			cid := cli.Push(recordWarnings080Path).WithArgs("--output", "raw").ShouldSucceed()
			gomega.Expect(cid).NotTo(gomega.BeEmpty(), "Push should return a valid CID even with warnings")

			// Validate that the returned CID correctly represents the pushed data
			utils.LoadAndValidateCID(cid, recordWarnings080Path)
		})
	})

	ginkgo.Context("Validation with API validation disabled", ginkgo.Ordered, ginkgo.Serial, ginkgo.Label("validation-api-disabled"), func() {
		// These tests should succeed when API validation is disabled
		// because embedded schema validation is used instead

		var record080Path, record070Path, record031Path, recordWarnings080Path string

		ginkgo.BeforeAll(func() {
			// Create temporary files for test records
			record080Path = filepath.Join(tempDir, "record_080_api_disabled_test.json")
			record070Path = filepath.Join(tempDir, "record_070_api_disabled_test.json")
			record031Path = filepath.Join(tempDir, "record_031_api_disabled_test.json")
			recordWarnings080Path = filepath.Join(tempDir, "record_warnings_080_api_disabled_test.json")

			_ = os.MkdirAll(filepath.Dir(record080Path), 0o755)
			_ = os.WriteFile(record080Path, testdata.ExpectedRecordV080JSON, 0o600)
			_ = os.WriteFile(record070Path, testdata.ExpectedRecordV070JSON, 0o600)
			_ = os.WriteFile(record031Path, testdata.ExpectedRecordV031JSON, 0o600)
			_ = os.WriteFile(recordWarnings080Path, testdata.ExpectedRecordWarningsV080JSON, 0o600)
		})

		ginkgo.It("should successfully push record_080.json when API validation is disabled", func() {
			cid := cli.Push(record080Path).WithArgs("--output", "raw").ShouldSucceed()
			gomega.Expect(cid).NotTo(gomega.BeEmpty(), "Push should return a valid CID")

			// Validate that the returned CID correctly represents the pushed data
			utils.LoadAndValidateCID(cid, record080Path)
		})

		ginkgo.It("should successfully push record_070.json when API validation is disabled", func() {
			cid := cli.Push(record070Path).WithArgs("--output", "raw").ShouldSucceed()
			gomega.Expect(cid).NotTo(gomega.BeEmpty(), "Push should return a valid CID")

			// Validate that the returned CID correctly represents the pushed data
			utils.LoadAndValidateCID(cid, record070Path)
		})

		ginkgo.It("should successfully push record_031.json when API validation is disabled", func() {
			cid := cli.Push(record031Path).WithArgs("--output", "raw").ShouldSucceed()
			gomega.Expect(cid).NotTo(gomega.BeEmpty(), "Push should return a valid CID")

			// Validate that the returned CID correctly represents the pushed data
			utils.LoadAndValidateCID(cid, record031Path)
		})

		ginkgo.It("should successfully push record_warnings_080.json when API validation is disabled", func() {
			// This record contains warnings that would fail in strict mode
			// but should succeed when API validation is disabled (using embedded validation)
			cid := cli.Push(recordWarnings080Path).WithArgs("--output", "raw").ShouldSucceed()
			gomega.Expect(cid).NotTo(gomega.BeEmpty(), "Push should return a valid CID even with warnings")

			// Validate that the returned CID correctly represents the pushed data
			utils.LoadAndValidateCID(cid, recordWarnings080Path)
		})

		ginkgo.It("should successfully pull records after pushing when API validation is disabled", func() {
			// Test pulling the records we just pushed
			cid080 := cli.Push(record080Path).WithArgs("--output", "raw").ShouldSucceed()
			cli.Pull(cid080).ShouldSucceed()

			cid070 := cli.Push(record070Path).WithArgs("--output", "raw").ShouldSucceed()
			cli.Pull(cid070).ShouldSucceed()

			cid031 := cli.Push(record031Path).WithArgs("--output", "raw").ShouldSucceed()
			cli.Pull(cid031).ShouldSucceed()

			cidWarnings080 := cli.Push(recordWarnings080Path).WithArgs("--output", "raw").ShouldSucceed()
			cli.Pull(cidWarnings080).ShouldSucceed()
		})
	})
})
