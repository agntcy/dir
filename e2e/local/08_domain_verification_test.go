// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"

	"github.com/agntcy/dir/e2e/shared/config"
	"github.com/agntcy/dir/e2e/shared/utils"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
)

// Test constants for domain verification.
const (
	domainVerifyTempDirPrefix = "domain-verify-test"
)

// WellKnownFile represents the structure of /.well-known/oasf.json
type WellKnownFile struct {
	Version int            `json:"version"`
	Keys    []WellKnownKey `json:"keys"`
}

// WellKnownKey represents a key entry in the well-known file.
type WellKnownKey struct {
	ID        string `json:"id,omitempty"`
	Type      string `json:"type"`
	PublicKey string `json:"publicKey"`
}

// domainVerifyTestPaths holds paths for test files.
type domainVerifyTestPaths struct {
	tempDir    string
	record     string
	privateKey string
	publicKey  string
}

func setupDomainVerifyTestPaths() *domainVerifyTestPaths {
	tempDir, err := os.MkdirTemp("", domainVerifyTempDirPrefix)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	return &domainVerifyTestPaths{
		tempDir:    tempDir,
		record:     filepath.Join(tempDir, "record.json"),
		privateKey: filepath.Join(tempDir, "cosign.key"),
		publicKey:  filepath.Join(tempDir, "cosign.pub"),
	}
}

// createTestRecord creates a test record with the given domain in the name field.
func createTestRecord(domain string) []byte {
	record := map[string]interface{}{
		"name":           fmt.Sprintf("%s/test-agent", domain),
		"version":        "v1.0.0",
		"schema_version": "0.8.0",
		"description":    "Test agent for domain verification e2e tests",
		"authors":        []string{"Test Author"},
		"created_at":     "2025-01-01T00:00:00Z",
		"skills": []map[string]interface{}{
			{
				"name": "natural_language_processing/natural_language_generation/text_completion",
				"id":   10201,
			},
		},
		"locators": []map[string]interface{}{
			{
				"type": "docker_image",
				"url":  "https://ghcr.io/test/agent",
			},
		},
		"domains": []map[string]interface{}{
			{
				"id":   301,
				"name": "life_science/biotechnology",
			},
		},
	}

	data, err := json.MarshalIndent(record, "", "  ")
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	return data
}

// getPublicKeyDER reads a PEM public key file and returns DER-encoded bytes.
func getPublicKeyDER(pubKeyPath string) []byte {
	pemData, err := os.ReadFile(pubKeyPath)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	pubKey, err := cryptoutils.UnmarshalPEMToPublicKey(pemData)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	derBytes, err := cryptoutils.MarshalPublicKeyToDER(pubKey)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	return derBytes
}

var _ = ginkgo.Describe("Running dirctl end-to-end tests for domain verification", func() {
	var cli *utils.CLI

	ginkgo.BeforeEach(func() {
		if cfg.DeploymentMode != config.DeploymentModeLocal {
			ginkgo.Skip("Skipping test, not in local mode")
		}

		utils.ResetCLIState()
		cli = utils.NewCLI()
	})

	var (
		paths      *domainVerifyTestPaths
		cid        string
		testServer *httptest.Server
	)

	ginkgo.Context("domain verification workflow with well-known file", ginkgo.Ordered, func() {
		ginkgo.BeforeAll(func() {
			// Setup test paths
			paths = setupDomainVerifyTestPaths()

			// Generate cosign key pair
			utils.GenerateCosignKeyPair(paths.tempDir)
			gomega.Expect(paths.privateKey).To(gomega.BeAnExistingFile())
			gomega.Expect(paths.publicKey).To(gomega.BeAnExistingFile())

			// Get public key in DER format (base64 encoded)
			derBytes := getPublicKeyDER(paths.publicKey)
			pubKeyBase64 := base64.StdEncoding.EncodeToString(derBytes)

			// Create well-known file content
			wellKnown := WellKnownFile{
				Version: 1,
				Keys: []WellKnownKey{
					{
						ID:        "test-key-1",
						Type:      "ecdsa-p256",
						PublicKey: pubKeyBase64,
					},
				},
			}

			wellKnownJSON, err := json.Marshal(wellKnown)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Start test HTTP server that serves /.well-known/oasf.json
			testServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/.well-known/oasf.json" {
					w.Header().Set("Content-Type", "application/json")
					w.Write(wellKnownJSON)
				} else {
					http.NotFound(w, r)
				}
			}))

			// Extract host from test server URL (e.g., "127.0.0.1:12345")
			// The test server URL is like "http://127.0.0.1:12345"
			serverHost := testServer.URL[7:] // Remove "http://"

			// Create test record with server's domain
			recordData := createTestRecord(serverHost)
			err = os.WriteFile(paths.record, recordData, 0o600)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Set cosign password
			err = os.Setenv("COSIGN_PASSWORD", utils.TestPassword)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})

		ginkgo.AfterAll(func() {
			// Stop test server
			if testServer != nil {
				testServer.Close()
			}

			// Cleanup
			os.Unsetenv("COSIGN_PASSWORD")

			if paths != nil && paths.tempDir != "" {
				os.RemoveAll(paths.tempDir)
			}
		})

		ginkgo.It("should push a record with domain in name", func() {
			cid = cli.Push(paths.record).WithArgs("--output", "raw").ShouldSucceed()
			gomega.Expect(cid).NotTo(gomega.BeEmpty())
		})

		ginkgo.It("should sign the record and verify domain ownership", func() {
			// Sign should succeed and trigger domain verification
			output := cli.Sign(cid, paths.privateKey).ShouldSucceed()
			gomega.Expect(output).To(gomega.ContainSubstring("signed"))
		})

		ginkgo.It("should check domain verification status via naming check", func() {
			output := cli.Naming().Check(cid).
				WithArgs("--output", "json").
				ShouldSucceed()

			// Parse JSON output
			var result map[string]interface{}
			err := json.Unmarshal([]byte(output), &result)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Check verification status
			gomega.Expect(result["verified"]).To(gomega.BeTrue())
			gomega.Expect(result["verification"]).NotTo(gomega.BeNil())

			verification := result["verification"].(map[string]interface{})
			gomega.Expect(verification["method"]).To(gomega.Equal("wellknown"))
		})

		ginkgo.It("should not create duplicate verification when verify is called again", func() {
			// Call verify again
			output := cli.Naming().Verify(cid).
				WithArgs("--output", "json").
				ShouldSucceed()

			// Should return existing verification (not create new one)
			var result map[string]interface{}
			err := json.Unmarshal([]byte(output), &result)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			gomega.Expect(result["verified"]).To(gomega.BeTrue())
		})
	})

	ginkgo.Context("domain verification failure cases", ginkgo.Ordered, func() {
		var failPaths *domainVerifyTestPaths
		var failCid string

		ginkgo.BeforeAll(func() {
			// Setup test paths
			failPaths = setupDomainVerifyTestPaths()

			// Generate cosign key pair
			utils.GenerateCosignKeyPair(failPaths.tempDir)

			// Create test record with non-existent domain (no well-known file)
			recordData := createTestRecord("nonexistent.invalid.domain.test:9999")
			err := os.WriteFile(failPaths.record, recordData, 0o600)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Set cosign password
			err = os.Setenv("COSIGN_PASSWORD", utils.TestPassword)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})

		ginkgo.AfterAll(func() {
			os.Unsetenv("COSIGN_PASSWORD")

			if failPaths != nil && failPaths.tempDir != "" {
				os.RemoveAll(failPaths.tempDir)
			}
		})

		ginkgo.It("should push record with unreachable domain", func() {
			failCid = cli.Push(failPaths.record).WithArgs("--output", "raw").ShouldSucceed()
			gomega.Expect(failCid).NotTo(gomega.BeEmpty())
		})

		ginkgo.It("should sign record but domain verification should fail gracefully", func() {
			// Signing should still succeed even if domain verification fails
			output := cli.Sign(failCid, failPaths.privateKey).ShouldSucceed()
			gomega.Expect(output).To(gomega.ContainSubstring("signed"))
		})

		ginkgo.It("should show unverified status via naming check", func() {
			output := cli.Naming().Check(failCid).
				WithArgs("--output", "json").
				ShouldSucceed()

			var result map[string]interface{}
			err := json.Unmarshal([]byte(output), &result)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Should not be verified
			gomega.Expect(result["verified"]).To(gomega.BeFalse())
		})
	})

	ginkgo.Context("domain verification with mismatched key", ginkgo.Ordered, func() {
		var mismatchPaths *domainVerifyTestPaths
		var mismatchCid string
		var mismatchServer *httptest.Server

		ginkgo.BeforeAll(func() {
			// Setup test paths
			mismatchPaths = setupDomainVerifyTestPaths()

			// Generate cosign key pair (this will be used for signing)
			utils.GenerateCosignKeyPair(mismatchPaths.tempDir)

			// Create well-known file with a DIFFERENT key (not matching signing key)
			// Use a hardcoded different key for testing
			wellKnown := WellKnownFile{
				Version: 1,
				Keys: []WellKnownKey{
					{
						ID:        "different-key",
						Type:      "ecdsa-p256",
						PublicKey: "MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEDIFFRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRQ==",
					},
				},
			}

			wellKnownJSON, err := json.Marshal(wellKnown)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Start test HTTP server
			mismatchServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/.well-known/oasf.json" {
					w.Header().Set("Content-Type", "application/json")
					w.Write(wellKnownJSON)
				} else {
					http.NotFound(w, r)
				}
			}))

			serverHost := mismatchServer.URL[7:]

			// Create test record
			recordData := createTestRecord(serverHost)
			err = os.WriteFile(mismatchPaths.record, recordData, 0o600)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Set cosign password
			err = os.Setenv("COSIGN_PASSWORD", utils.TestPassword)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})

		ginkgo.AfterAll(func() {
			if mismatchServer != nil {
				mismatchServer.Close()
			}

			os.Unsetenv("COSIGN_PASSWORD")

			if mismatchPaths != nil && mismatchPaths.tempDir != "" {
				os.RemoveAll(mismatchPaths.tempDir)
			}
		})

		ginkgo.It("should push record", func() {
			mismatchCid = cli.Push(mismatchPaths.record).WithArgs("--output", "raw").ShouldSucceed()
			gomega.Expect(mismatchCid).NotTo(gomega.BeEmpty())
		})

		ginkgo.It("should sign record but domain verification should fail due to key mismatch", func() {
			// Signing should succeed
			output := cli.Sign(mismatchCid, mismatchPaths.privateKey).ShouldSucceed()
			gomega.Expect(output).To(gomega.ContainSubstring("signed"))
		})

		ginkgo.It("should show unverified status due to key mismatch", func() {
			output := cli.Naming().Check(mismatchCid).
				WithArgs("--output", "json").
				ShouldSucceed()

			var result map[string]interface{}
			err := json.Unmarshal([]byte(output), &result)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Should not be verified due to key mismatch
			gomega.Expect(result["verified"]).To(gomega.BeFalse())
		})
	})
})
