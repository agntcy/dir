// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	_ "embed"
	"encoding/json"
	"os"
	"path/filepath"

	signv1 "github.com/agntcy/dir/api/sign/v1"
	"github.com/agntcy/dir/e2e/config"
	"github.com/agntcy/dir/e2e/utils"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

//go:embed testdata/agent_v3.json
var signTestAgentJSON []byte

// Test constants.
const (
	tempDirPrefix = "sign-test"
)

// Test file paths helper.
type testPaths struct {
	tempDir         string
	record          string
	privateKey      string
	publicKey       string
	signature       string
	signatureOutput string
}

func setupTestPaths() *testPaths {
	tempDir, err := os.MkdirTemp("", tempDirPrefix)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	return &testPaths{
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

		// Initialize CLI helper
		cli = utils.NewCLI()
	})

	// Test params
	var (
		paths         *testPaths
		tempAgentCID  string
		signatureData string
	)

	ginkgo.Context("signature workflow", ginkgo.Ordered, func() {
		// Setup: Create temporary directory and files for the entire workflow
		ginkgo.BeforeAll(func() {
			var err error
			paths = setupTestPaths()
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Write test agent to temp location
			err = os.WriteFile(paths.record, signTestAgentJSON, 0o600)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Generate cosign key pair for all tests
			utils.GenerateCosignKeyPair(paths.tempDir)

			// Verify key files were created
			gomega.Expect(paths.privateKey).To(gomega.BeAnExistingFile())
			gomega.Expect(paths.publicKey).To(gomega.BeAnExistingFile())

			// Create signature ONCE for all tests to ensure consistency
			// (Signatures are non-deterministic, so we need the same instance for push/pull)
			err = os.Setenv("COSIGN_PASSWORD", utils.TestPassword)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			signatureData = cli.Sign(paths.record, paths.privateKey).ShouldSucceed()

			// Save signature for all tests
			err = os.WriteFile(paths.signature, []byte(signatureData), 0o600)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			os.Unsetenv("COSIGN_PASSWORD")
		})

		// Cleanup: Remove temporary directory after all workflow tests
		ginkgo.AfterAll(func() {
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

		ginkgo.It("should sign a record with a key pair", func() {
			// Signature was already created in BeforeAll, just verify it exists and is valid
			gomega.Expect(signatureData).NotTo(gomega.BeEmpty(), "Signature should have been created in BeforeAll")
			gomega.Expect(paths.signature).To(gomega.BeAnExistingFile(), "Signature file should exist")

			// Verify the signature can be parsed as valid JSON
			var signature signv1.Signature
			err := json.Unmarshal([]byte(signatureData), &signature)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Signature should be valid JSON")
			gomega.Expect(signature.GetAlgorithm()).NotTo(gomega.BeEmpty(), "Signature should have an algorithm")
			gomega.Expect(signature.GetSignature()).NotTo(gomega.BeEmpty(), "Signature should have signature data")
		})

		ginkgo.It("should push a record to the store with a signature", func() {
			tempAgentCID = cli.Push(paths.record).WithArgs("--signature", paths.signature).ShouldSucceed()

			// Validate that the returned CID correctly represents the pushed data
			utils.LoadAndValidateCID(tempAgentCID, paths.record)
		})

		ginkgo.It("should pull a record from the store with a signature", func() {
			output := cli.Pull(tempAgentCID).WithArgs("--include-signature").ShouldSucceed()

			// Extract signature from output
			signatureOutput := utils.ExtractSignatureFromCombinedOutput(output)
			gomega.Expect(signatureOutput).NotTo(gomega.BeEmpty())

			// Compare with expected signature
			utils.CompareSignatures(signatureData, signatureOutput)

			// Save output signature to file
			err := os.WriteFile(paths.signatureOutput, []byte(signatureOutput), 0o600)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})

		ginkgo.It("should verify a signature with a public key", func() {
			// Ensure we have a signature file
			gomega.Expect(paths.signatureOutput).To(gomega.BeAnExistingFile(), "Signature file should exist from pull test")

			cli.Command("verify").
				WithArgs(paths.record, paths.signatureOutput, "--key", paths.publicKey).
				ShouldContain("Record signature verified successfully!")
		})

		ginkgo.It("should clean up by deleting the record from store", func() {
			// Delete the record to ensure clean state for subsequent test runs
			gomega.Expect(tempAgentCID).NotTo(gomega.BeEmpty(), "Agent CID should be available for deletion")

			cli.Delete(tempAgentCID).ShouldContain("Deleted")
		})
	})
})
