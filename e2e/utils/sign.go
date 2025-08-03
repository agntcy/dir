// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"strings"

	signv1 "github.com/agntcy/dir/api/sign/v1"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

const (
	// ExpectedJSONObjectCount is the expected number of JSON objects in combined agent+signature output.
	ExpectedJSONObjectCount = 2
)

// Test constants for signature operations.
const (
	TestPassword = "testpassword"
)

// GenerateCosignKeyPair generates a cosign key pair in the specified directory.
// Helper function for signature testing.
func GenerateCosignKeyPair(dir string) {
	cmd := exec.Command("cosign", "generate-key-pair")

	cmd.Env = append(os.Environ(), "COSIGN_PASSWORD="+TestPassword)
	cmd.Dir = dir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		ginkgo.Fail("cosign generate-key-pair failed: " + err.Error() + "\nStderr: " + stderr.String())
	}
}

// ExtractSignatureFromCombinedOutput extracts signature JSON from combined output.
// Input format: "{agent_json}{signature_json}".
func ExtractSignatureFromCombinedOutput(combinedOutput string) string {
	// Find the boundary between agent JSON and signature JSON
	// Look for "}{"" pattern which separates the two JSON objects
	parts := strings.Split(combinedOutput, "}{")
	gomega.Expect(parts).To(gomega.HaveLen(ExpectedJSONObjectCount), "Expected combined output to contain exactly 2 JSON objects")

	// The second part is the signature JSON, but we need to add back the opening "{"
	return "{" + parts[1]
}

// CompareSignatures compares two signature JSON strings for equality.
// Compares individual fields to avoid protobuf mutex copying issues.
func CompareSignatures(expected, actual string) {
	var expectedSignature, actualSignature signv1.Signature

	err := json.Unmarshal([]byte(expected), &expectedSignature)
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to unmarshal expected signature")

	err = json.Unmarshal([]byte(actual), &actualSignature)
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to unmarshal actual signature")

	// Compare individual fields to avoid protobuf lock copying
	// Note: SignedAt is skipped because it uses time.Now() and changes between test runs
	gomega.Expect(actualSignature.GetAlgorithm()).To(gomega.Equal(expectedSignature.GetAlgorithm()), "Algorithm should match")
	gomega.Expect(actualSignature.GetSignature()).To(gomega.Equal(expectedSignature.GetSignature()), "Signature should match")
	gomega.Expect(actualSignature.GetCertificate()).To(gomega.Equal(expectedSignature.GetCertificate()), "Certificate should match")
	gomega.Expect(actualSignature.GetContentType()).To(gomega.Equal(expectedSignature.GetContentType()), "ContentType should match")
	gomega.Expect(actualSignature.GetContentBundle()).To(gomega.Equal(expectedSignature.GetContentBundle()), "ContentBundle should match")

	// Verify SignedAt is present and valid, but don't require exact match
	gomega.Expect(actualSignature.GetSignedAt()).NotTo(gomega.BeEmpty(), "SignedAt should be present")
	gomega.Expect(expectedSignature.GetSignedAt()).NotTo(gomega.BeEmpty(), "Expected SignedAt should be present")
}
