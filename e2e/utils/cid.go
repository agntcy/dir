// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/onsi/gomega"
)

// ValidateCID validates that a string is a properly formatted CID.
// Uses the ConvertCIDToDigest function to ensure the CID can be decoded successfully.
func ValidateCID(cidString string) {
	gomega.Expect(cidString).NotTo(gomega.BeEmpty(), "CID should not be empty")

	// Attempt to convert CID to digest - this validates the CID format
	_, err := corev1.ConvertCIDToDigest(cidString)
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "CID should be valid and decodable")
}

// ValidateCIDFormat validates CID format and returns whether it's valid.
// This is a non-assertion version for conditional logic.
func ValidateCIDFormat(cidString string) bool {
	if cidString == "" {
		return false
	}

	_, err := corev1.ConvertCIDToDigest(cidString)

	return err == nil
}

// MarshalOASFCanonical marshals OASF JSON data using canonical JSON serialization.
// This ensures deterministic, cross-language compatible byte representation.
// Mirrors the logic from api/core/v1/oasf.go marshalOASFCanonical function.
func MarshalOASFCanonical(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("cannot marshal empty data")
	}

	// Step 1: Parse the JSON to ensure it's valid
	var normalized interface{}
	if err := json.Unmarshal(data, &normalized); err != nil {
		return nil, fmt.Errorf("failed to parse JSON for canonical marshaling: %w", err)
	}

	// Step 2: Marshal with sorted keys for deterministic output
	// encoding/json.Marshal sorts map keys alphabetically
	canonicalBytes, err := json.Marshal(normalized)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal canonical JSON with sorted keys: %w", err)
	}

	return canonicalBytes, nil
}

// ValidateCIDAgainstData validates that a CID string correctly represents the given data.
// This performs comprehensive validation: format check + data integrity verification.
func ValidateCIDAgainstData(cidString string, originalData []byte) {
	// First validate CID format
	ValidateCID(cidString)

	expectedCID := CalculateCIDFromData(originalData)
	gomega.Expect(expectedCID).NotTo(gomega.BeEmpty(), "Should be able to calculate CID from data")

	// Verify the CID matches what we expect
	gomega.Expect(cidString).To(gomega.Equal(expectedCID),
		"CID should match the calculated CID for the canonical data")
}

// LoadAndValidateCID loads a JSON file, canonicalizes it, and validates the CID represents that data.
// This is a convenience function for test files.
func LoadAndValidateCID(cidString string, filePath string) {
	// Load the file
	data, err := os.ReadFile(filePath)
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Should be able to read file "+filePath)

	// Validate CID against the file data
	ValidateCIDAgainstData(cidString, data)
}

// CalculateCIDFromFile calculates the CID for the record file.
func CalculateCIDFromFile(filePath string) string {
	// Load the file
	data, err := os.ReadFile(filePath)
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Should be able to read file "+filePath)

	// Calculate the CID
	return CalculateCIDFromData(data)
}

// CalculateCIDFromData calculates the CID for the record data.
func CalculateCIDFromData(data []byte) string {
	// Canonicalize the original data
	canonicalData, err := MarshalOASFCanonical(data)
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Should be able to canonicalize data")

	// Calculate what the CID should be for this canonical data
	digest, err := corev1.CalculateDigest(canonicalData)
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Should be able to calculate digest")

	cid, err := corev1.ConvertDigestToCID(digest)
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Should be able to convert digest to CID")

	return cid
}

// ValidateCIDPrefix validates that a CID starts with expected prefixes.
// CIDv1 with codec 1 typically starts with "baf" for base32 encoding.
func ValidateCIDPrefix(cidString string) {
	ValidateCID(cidString) // First ensure it's a valid CID

	// CIDv1 with base32 encoding typically starts with "baf"
	gomega.Expect(cidString).To(gomega.HavePrefix("baf"),
		"CID should start with 'baf' prefix for CIDv1 base32 encoding")
}
