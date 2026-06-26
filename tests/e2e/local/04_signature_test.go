// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package local

import (
	_ "embed"
	"net/http"
	"os"
	"path/filepath"
	"time"

	signv1 "github.com/agntcy/dir/api/sign/v1"
	"github.com/agntcy/dir/tests/e2e/shared/testdata"
	"github.com/agntcy/dir/tests/e2e/shared/utils"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"google.golang.org/protobuf/encoding/protojson"
)

var _ = ginkgo.Describe("Running dirctl end-to-end tests to check signature support", func() {
	ginkgo.BeforeEach(func() {
		utils.ResetCLIState()
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
			cosignPassword := "testpassword"
			utils.GenerateCosignKeyPair(paths.tempDir, cosignPassword)

			// Verify key files were created
			gomega.Expect(paths.privateKey).To(gomega.BeAnExistingFile())
			gomega.Expect(paths.publicKey).To(gomega.BeAnExistingFile())

			// Set cosign password for all tests
			err = os.Setenv("COSIGN_PASSWORD", cosignPassword)
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
			cid = testEnv.CLI.Push(paths.record).WithArgs("--output", "raw").ShouldSucceed()

			// Validate that the returned CID correctly represents the pushed data
			utils.LoadAndValidateCID(cid, paths.record)
		})

		ginkgo.It("should sign a record with a key pair", func() {
			_ = testEnv.CLI.Sign(cid, paths.privateKey).ShouldSucceed()

			time.Sleep(10 * time.Second)
		})

		ginkgo.It("should verify a signature with a public key", func() {
			// Verify using the public key and write output to file
			_ = testEnv.CLI.Command("verify").
				WithArgs(cid).
				WithArgs("--key", paths.publicKey).
				WithArgs("--output-file", paths.verifyOutput).
				ShouldSucceed()

			// Read and parse the verify response from the output file
			verifyData, err := os.ReadFile(paths.verifyOutput)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			var verifyResponse signv1.VerifyResponse

			err = protojson.Unmarshal(verifyData, &verifyResponse)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Verify the response indicates success
			gomega.Expect(verifyResponse.GetSuccess()).To(gomega.BeTrue())

			// Verify that signers array is present and not empty
			gomega.Expect(verifyResponse.GetSigners()).NotTo(gomega.BeEmpty())

			// For key-based signing, verify key signer info
			signer := verifyResponse.GetSigners()[0]
			keySigner := signer.GetKey()
			gomega.Expect(keySigner).NotTo(gomega.BeNil(), "Expected key signer info for key-signed record")
			gomega.Expect(keySigner.GetPublicKey()).NotTo(gomega.BeEmpty(), "Expected public key in signer info")
			gomega.Expect(keySigner.GetAlgorithm()).NotTo(gomega.BeEmpty(), "Expected algorithm in signer info")
		})

		ginkgo.It("should verify any valid signature on the record", func() {
			// Verify without specifying a key (any valid signature)
			testEnv.CLI.Command("verify").
				WithArgs(cid).
				ShouldContain("Record signature is: trusted")
		})

		ginkgo.It("should pull a signature from the store", func() {
			testEnv.CLI.Command("pull").
				WithArgs(cid, "--signature").
				WithArgs("--output", "json").
				ShouldContain("\"signature\":")
		})

		ginkgo.It("should pull a public key from the store", func() {
			testEnv.CLI.Command("pull").
				WithArgs(cid, "--public-key").
				ShouldContain("-----BEGIN PUBLIC KEY-----")
		})
	})

	// Regression test for: key-based signatures stored with empty Signature field
	// cause catalogSignatures() to skip the entry, producing no trustManifest.
	ginkgo.Context("key signature appears in AI catalog after reconciliation", ginkgo.Ordered, func() {
		var (
			catalogPaths     *signTestPaths
			catalogRecordCID string
		)

		ginkgo.BeforeAll(func() {
			if testEnv.Config.GatewayAddress == "" {
				ginkgo.Skip("HTTP gateway address not configured for this environment")
			}

			catalogPaths = setupSignTestPaths()

			// v100 record carries integration/mcp, so it projects to an AI catalog entry.
			err := os.WriteFile(catalogPaths.record, testdata.ExpectedRecordV100JSON, 0o600)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			cosignPassword := "testpassword"
			utils.GenerateCosignKeyPair(catalogPaths.tempDir, cosignPassword)
			gomega.Expect(catalogPaths.privateKey).To(gomega.BeAnExistingFile())
			gomega.Expect(catalogPaths.publicKey).To(gomega.BeAnExistingFile())

			err = os.Setenv("COSIGN_PASSWORD", cosignPassword)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})

		ginkgo.AfterAll(func() {
			os.Unsetenv("COSIGN_PASSWORD")

			if catalogPaths != nil && catalogPaths.tempDir != "" {
				_ = os.RemoveAll(catalogPaths.tempDir)
			}
		})

		ginkgo.It("should push and sign a catalog record with a key", func() {
			catalogRecordCID = testEnv.CLI.Push(catalogPaths.record).
				WithArgs("--output", "raw").ShouldSucceed()
			gomega.Expect(catalogRecordCID).NotTo(gomega.BeEmpty())

			_ = testEnv.CLI.Sign(catalogRecordCID, catalogPaths.privateKey).ShouldSucceed()
		})

		ginkgo.It("should include trustManifest in the AI catalog entry after the reconciler processes the key signature", func(ctx ginkgo.SpecContext) {
			// The reconciler (interval: 5s) reads the signature from OCI and stores it in the DB.
			// catalogSignatures() skips any DB row where Signature=="", so the trustManifest field
			// is only populated when the Signature is correctly persisted by the reconciler.
			// This test fails when fetcher.go stores base64(ContentBundle) instead of GetSignature()
			// for key-based signatures, because ContentBundle is always empty for key-based sigs.
			gomega.Eventually(func(g gomega.Gomega) {
				status, body := getAgents(ctx, "")
				g.Expect(status).To(gomega.Equal(http.StatusOK))
				g.Expect(body).To(gomega.ContainSubstring(catalogRecordCID))
				g.Expect(body).To(gomega.ContainSubstring(`"trustManifest"`))
			}).WithContext(ctx).WithTimeout(30 * time.Second).WithPolling(2 * time.Second).Should(gomega.Succeed())
		})
	})
})

// Test file paths helper.
type signTestPaths struct {
	tempDir         string
	record          string
	privateKey      string
	publicKey       string
	signature       string
	signatureOutput string
	verifyOutput    string
}

func setupSignTestPaths() *signTestPaths {
	tempDir, err := os.MkdirTemp("", "sign-test-*")
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	return &signTestPaths{
		tempDir:         tempDir,
		record:          filepath.Join(tempDir, "record.json"),
		signature:       filepath.Join(tempDir, "signature.json"),
		signatureOutput: filepath.Join(tempDir, "signature-output.json"),
		verifyOutput:    filepath.Join(tempDir, "verify-output.json"),
		privateKey:      filepath.Join(tempDir, "cosign.key"),
		publicKey:       filepath.Join(tempDir, "cosign.pub"),
	}
}
