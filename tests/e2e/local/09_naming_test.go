// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"os"
	"path/filepath"
	"time"

	"github.com/agntcy/dir/tests/e2e/shared/testdata"
	"github.com/agntcy/dir/tests/e2e/shared/utils"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

// Test constants for naming verification.
const (
	namingTempDirPrefix = "naming-test"

	// Pre-generated cosign keys directory (under testdata/dns-validation).
	// These keys match the JWKS served by the dns-validation chart.
	dnsValidationKeysDir = "./testdata/dns-validation"

	// verificationWaitTimeout is the maximum time to wait for the server
	// to create the name verification row after signing.
	verificationWaitTimeout = 60 * time.Second

	// verificationPollInterval is how often to poll for verification status.
	verificationPollInterval = 2 * time.Second
)

// namingTestPaths holds paths for test files.
type namingTestPaths struct {
	tempDir    string
	record     string
	privateKey string
	publicKey  string
}

func setupNamingTestPaths() *namingTestPaths {
	tempDir, err := os.MkdirTemp("", namingTempDirPrefix)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	return &namingTestPaths{
		tempDir:    tempDir,
		record:     filepath.Join(tempDir, "record.json"),
		privateKey: filepath.Join(dnsValidationKeysDir, "cosign.key"),
		publicKey:  filepath.Join(dnsValidationKeysDir, "cosign.pub"),
	}
}

var _ = ginkgo.Describe("Running dirctl e2e tests for DNS name verification", func() {
	ginkgo.BeforeEach(func() {
		utils.ResetCLIState()
	})

	ginkgo.Context("naming verification workflow", ginkgo.Ordered, func() {
		var (
			paths *namingTestPaths
			cid   string
		)

		ginkgo.BeforeAll(func() {
			paths = setupNamingTestPaths()

			// Write test record with DNS name to temp location
			// The record name (http://dns-validation-http/example/research-assistant-v4) must match
			// the service name that serves the JWKS endpoint
			err := os.WriteFile(paths.record, testdata.ExpectedRecordV080V5JSON, 0o600)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Verify pre-generated cosign key files exist
			// These keys match the JWKS served by the dns-validation chart
			gomega.Expect(paths.privateKey).To(gomega.BeAnExistingFile(),
				"Pre-generated cosign.key not found at %s", paths.privateKey)
			gomega.Expect(paths.publicKey).To(gomega.BeAnExistingFile(),
				"Pre-generated cosign.pub not found at %s", paths.publicKey)

			// Set empty cosign password (keys were generated with empty password)
			err = os.Setenv("COSIGN_PASSWORD", "")
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})

		ginkgo.AfterAll(func() {
			// Unset COSIGN_PASSWORD
			os.Unsetenv("COSIGN_PASSWORD")

			// Clean up pushed record
			if cid != "" {
				_, _ = testEnv.CLI.Delete(cid).Execute()
			}

			// Clean up temp directory
			if paths != nil && paths.tempDir != "" {
				_ = os.RemoveAll(paths.tempDir)
			}
		})

		ginkgo.It("should push a record with DNS-prefixed name", func() {
			cid = testEnv.CLI.Push(paths.record).WithArgs("--output", "raw").ShouldSucceed()
			gomega.Expect(cid).NotTo(gomega.BeEmpty())

			// Verify the record was pushed successfully
			utils.LoadAndValidateCID(cid, paths.record)
		})

		ginkgo.It("should sign the record with cosign key", func() {
			_ = testEnv.CLI.Sign(cid, paths.privateKey).ShouldSucceed()

			// Allow time for signature processing
			time.Sleep(5 * time.Second)
		})

		ginkgo.It("should verify signature is trusted", func() {
			// First verify the basic signature verification works
			testEnv.CLI.Command("verify").
				WithArgs(cid).
				ShouldContain("Record signature is: trusted")
		})

		ginkgo.It("should check naming verification status by CID", func() {
			// Poll until verification is created or timeout
			gomega.Eventually(func() string {
				output, err := testEnv.CLI.Naming().Verify(cid).
					WithArgs("--output", "json").
					Execute()
				if err != nil {
					return ""
				}

				return output
			}, verificationWaitTimeout, verificationPollInterval).Should(
				gomega.ContainSubstring("verified"),
				"Verification should be created after server processes the signed record",
			)
		})

		ginkgo.It("should check naming verification status by name", func() {
			output := testEnv.CLI.Naming().Verify("http://dns-validation-http/example/research-assistant-v4").
				WithArgs("--output", "json").
				ShouldSucceed()

			gomega.Expect(output).To(gomega.Or(
				gomega.ContainSubstring("verified"),
				gomega.ContainSubstring("verification"),
			))
		})

		ginkgo.It("should check naming verification status by name with version", func() {
			output := testEnv.CLI.Naming().Verify("http://dns-validation-http/example/research-assistant-v4:v5.0.0").
				WithArgs("--output", "json").
				ShouldSucceed()

			gomega.Expect(output).To(gomega.Or(
				gomega.ContainSubstring("verified"),
				gomega.ContainSubstring("verification"),
			))
		})
	})
})
