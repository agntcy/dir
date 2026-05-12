// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"encoding/json"
	"fmt"
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

// verifyOutput is the JSON shape printed by `dirctl naming verify --output json`.
type verifyOutput struct {
	Cid        string `json:"cid"`
	Verified   bool   `json:"verified"`
	Message    string `json:"message,omitempty"`
	Domain     string `json:"domain,omitempty"`
	Method     string `json:"method,omitempty"`
	KeyID      string `json:"key_id,omitempty"`
	VerifiedAt string `json:"verified_at,omitempty"`
}

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

// rewriteRecordNameForEnv takes the shared embedded OASF record and rewrites
// only its top-level `name` field so it points at the dns-validation
// infrastructure of the current test environment.
func rewriteRecordNameForEnv(recordJSON []byte, host string) ([]byte, string) {
	ginkgo.GinkgoHelper()

	var record map[string]json.RawMessage
	gomega.Expect(json.Unmarshal(recordJSON, &record)).To(gomega.Succeed(),
		"embedded naming record must be valid JSON")

	newName := fmt.Sprintf("http://%s/example/research-assistant-v4", host)
	encoded, err := json.Marshal(newName)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	record["name"] = encoded

	out, err := json.Marshal(record)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	return out, newName
}

// parseVerifyOutput parses the JSON returned by `dirctl naming verify --output json`.
func parseVerifyOutput(raw string) (*verifyOutput, error) {
	var out verifyOutput
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, fmt.Errorf("parse naming verify JSON: %w (raw=%q)", err, raw)
	}

	return &out, nil
}

// isPendingVerificationMessage reports whether the server response indicates
// that the name reconciler has not yet attempted (or has expired the cache for)
// verification of this record.
func isPendingVerificationMessage(msg string) bool {
	switch msg {
	case "",
		// Strings produced by server/controller/naming.go.
		"no verification found",
		"verification invalid or expired":
		return true
	default:
		return false
	}
}

// expectVerified polls `dirctl naming verify` for the given reference until the
// server reports a terminal state.
func expectVerified(ref, expectedHost string) *verifyOutput {
	ginkgo.GinkgoHelper()

	deadline := time.Now().Add(verificationWaitTimeout)

	var last *verifyOutput

	for {
		raw, err := testEnv.CLI.Naming().Verify(ref).
			WithArgs("--output", "json").
			Execute()
		gomega.Expect(err).NotTo(gomega.HaveOccurred(),
			"naming verify command must succeed for ref %q", ref)

		parsed, err := parseVerifyOutput(raw)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		last = parsed

		if parsed.Verified {
			break
		}

		gomega.Expect(isPendingVerificationMessage(parsed.Message)).To(
			gomega.BeTrue(),
			"verification for %q reached terminal non-verified state: %s",
			ref, parsed.Message,
		)

		if !time.Now().Before(deadline) {
			ginkgo.Fail(fmt.Sprintf(
				"verification for %q did not reach terminal `verified: true` state within %s (last message=%q)",
				ref, verificationWaitTimeout, parsed.Message,
			))
		}

		time.Sleep(verificationPollInterval)
	}

	gomega.Expect(last).NotTo(gomega.BeNil())
	gomega.Expect(last.Verified).To(gomega.BeTrue())
	gomega.Expect(last.Domain).To(gomega.Equal(expectedHost),
		"verified domain should match the dns-validation host for this env")
	gomega.Expect(last.Method).NotTo(gomega.BeEmpty(),
		"verified result must report a verification method")
	gomega.Expect(last.KeyID).NotTo(gomega.BeEmpty(),
		"verified result must report a matched key id")
	gomega.Expect(last.VerifiedAt).NotTo(gomega.BeEmpty(),
		"verified result must report a verified_at timestamp")

	return last
}

var _ = ginkgo.Describe("Running dirctl e2e tests for DNS name verification", func() {
	ginkgo.BeforeEach(func() {
		utils.ResetCLIState()
	})

	ginkgo.Context("naming verification workflow", ginkgo.Ordered, func() {
		var (
			paths      *namingTestPaths
			cid        string
			recordName string
			host       string
		)

		ginkgo.BeforeAll(func() {
			paths = setupNamingTestPaths()

			host = testEnv.Config.NameVerificationHost
			gomega.Expect(host).NotTo(gomega.BeEmpty(),
				"test-config must set local.name_verification_host to the dns-validation host for this environment")

			recordBytes, name := rewriteRecordNameForEnv(testdata.ExpectedRecordV080V5JSON, host)
			recordName = name

			err := os.WriteFile(paths.record, recordBytes, 0o600)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			gomega.Expect(paths.privateKey).To(gomega.BeAnExistingFile(),
				"Pre-generated cosign.key not found at %s", paths.privateKey)
			gomega.Expect(paths.publicKey).To(gomega.BeAnExistingFile(),
				"Pre-generated cosign.pub not found at %s", paths.publicKey)

			err = os.Setenv("COSIGN_PASSWORD", "")
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})

		ginkgo.AfterAll(func() {
			os.Unsetenv("COSIGN_PASSWORD")

			if cid != "" {
				_, _ = testEnv.CLI.Delete(cid).Execute()
			}

			if paths != nil && paths.tempDir != "" {
				_ = os.RemoveAll(paths.tempDir)
			}
		})

		ginkgo.It("should push a record with DNS-prefixed name", func() {
			cid = testEnv.CLI.Push(paths.record).WithArgs("--output", "raw").ShouldSucceed()
			gomega.Expect(cid).NotTo(gomega.BeEmpty())

			utils.LoadAndValidateCID(cid, paths.record)
		})

		ginkgo.It("should sign the record with cosign key", func() {
			_ = testEnv.CLI.Sign(cid, paths.privateKey).ShouldSucceed()

			// Allow time for signature processing
			time.Sleep(5 * time.Second)
		})

		ginkgo.It("should verify signature is trusted", func() {
			testEnv.CLI.Command("verify").
				WithArgs(cid).
				ShouldContain("Record signature is: trusted")
		})

		ginkgo.It("should check naming verification status by CID", func() {
			expectVerified(cid, host)
		})

		ginkgo.It("should check naming verification status by name", func() {
			expectVerified(recordName, host)
		})

		ginkgo.It("should check naming verification status by name with version", func() {
			expectVerified(recordName+":v5.0.0", host)
		})
	})
})
