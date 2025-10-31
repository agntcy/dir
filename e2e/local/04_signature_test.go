// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package local

import (
	_ "embed"
	"os"
	"path/filepath"
	"time"

	"github.com/agntcy/dir/e2e/shared/config"
	"github.com/agntcy/dir/e2e/shared/testdata"
	"github.com/agntcy/dir/e2e/shared/utils"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

// Using the shared record data from embed.go

// Test constants.
const (
	signTempDirPrefix = "sign-test"
)

// Test file paths helper.
type signTestPaths struct {
	tempDir         string
	record          string
	privateKey      string
	publicKey       string
	signature       string
	signatureOutput string
}

func setupSignTestPaths() *signTestPaths {
	tempDir, err := os.MkdirTemp("", signTempDirPrefix)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	return &signTestPaths{
		tempDir:         tempDir,
		record:          filepath.Join(tempDir, "record.json"),
		signature:       filepath.Join(tempDir, "signature.json"),
		signatureOutput: filepath.Join(tempDir, "signature-output.json"),
		privateKey:      filepath.Join(tempDir, "cosign.key"),
		publicKey:       filepath.Join(tempDir, "cosign.pub"),
	}
}

var _ = ginkgo.Describe("Running dirctl end-to-end tests to check signature support", func() {
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
		paths *signTestPaths
		cid   string
	)

	ginkgo.Context("signature workflow", ginkgo.Ordered, func() {
		// Setup: Create temporary directory and files for the entire workflow
		ginkgo.BeforeAll(func() {
			var err error
			paths = setupSignTestPaths()
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Write test record to temp location
			err = os.WriteFile(paths.record, testdata.ExpectedRecordV070JSON, 0o600)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Generate cosign key pair for all tests
			utils.GenerateCosignKeyPair(paths.tempDir)

			// Verify key files were created
			gomega.Expect(paths.privateKey).To(gomega.BeAnExistingFile())
			gomega.Expect(paths.publicKey).To(gomega.BeAnExistingFile())

			// Set cosign password for all tests
			err = os.Setenv("COSIGN_PASSWORD", utils.TestPassword)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})

		// Cleanup: Remove temporary directory after all workflow tests
		ginkgo.AfterAll(func() {
			// Unset cosign password for all tests
			os.Unsetenv("COSIGN_PASSWORD")

			if paths != nil && paths.tempDir != "" {
				err := os.RemoveAll(paths.tempDir)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
			}
		})

		ginkgo.It("should create keys for signing", func() {
			// Keys are already created in BeforeAll, just verify they exist
			gomega.Expect(paths.privateKey).To(gomega.BeAnExistingFile())
			gomega.Expect(paths.publicKey).To(gomega.BeAnExistingFile())
		})

		ginkgo.It("should push a record to the store", func() {
			cid = cli.Push(paths.record).WithArgs("--output", "raw").ShouldSucceed()

			// Validate that the returned CID correctly represents the pushed data
			utils.LoadAndValidateCID(cid, paths.record)
		})

		ginkgo.It("should sign a record with a key pair", func() {
			_ = cli.Sign(cid, paths.privateKey).ShouldSucceed()

			time.Sleep(10 * time.Second)
		})

		ginkgo.It("should verify a signature with a public key on server side", func() {
			cli.Command("verify").
				WithArgs(cid).
				ShouldContain("Record signature is: trusted")
		})

		ginkgo.It("should pull a signature from the store", func() {
			cli.Command("pull").
				WithArgs(cid, "--signature").
				WithArgs("--output", "json").
				ShouldContain("\"signature\":")
		})

		ginkgo.It("should pull a public key from the store", func() {
			cli.Command("pull").
				WithArgs(cid, "--public-key").
				ShouldContain("-----BEGIN PUBLIC KEY-----")
		})
	})
})
