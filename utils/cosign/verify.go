// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package cosign

import (
	"bytes"
	"crypto"
	"encoding/base64"
	"fmt"

	sigs "github.com/sigstore/cosign/v3/pkg/signature"
)

// VerifySignaturesOptions contains the inputs needed for signature verification.
type VerifySignaturesOptions struct {
	// ExpectedPayload is the payload that signatures should be verified against.
	// This is typically generated from the record's digest using GeneratePayload().
	ExpectedPayload []byte

	// Signatures is a list of base64-encoded signatures to verify.
	Signatures []string

	// PublicKeys is a list of PEM-encoded public keys to try for verification.
	PublicKeys []string
}

// VerifySignatures verifies signatures against public keys using cosign.
// It iterates through all combinations of public keys and signatures to find
// a valid match, similar to Zot's VerifyCosignSignature pattern.
//
// Returns true if any signature verifies with any public key.
// Returns false with nil error if no valid combination is found or if
// no signatures/public keys are provided (record is not signed).
func VerifySignatures(opts *VerifySignaturesOptions) (bool, error) {
	if opts == nil {
		return false, nil
	}

	if len(opts.ExpectedPayload) == 0 || len(opts.Signatures) == 0 || len(opts.PublicKeys) == 0 {
		return false, nil
	}

	// Try each public key against each signature
	for _, publicKey := range opts.PublicKeys {
		for _, signature := range opts.Signatures {
			verified, err := verifySignatureWithKey(publicKey, signature, opts.ExpectedPayload)
			if err != nil {
				// Log and continue to try next combination
				continue
			}

			if verified {
				return true, nil
			}
		}
	}

	return false, nil
}

// verifySignatureWithKey verifies a single signature using a specific public key.
func verifySignatureWithKey(publicKey string, signature string, expectedPayload []byte) (bool, error) {
	// Load public key verifier using cosign's sigstore library
	verifier, err := sigs.LoadPublicKeyRaw([]byte(publicKey), crypto.SHA256)
	if err != nil {
		return false, fmt.Errorf("failed to load public key: %w", err)
	}

	// Decode base64 signature
	signatureBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		// If decoding fails, assume it's already raw bytes
		signatureBytes = []byte(signature)
	}

	// Verify signature against the expected payload
	err = verifier.VerifySignature(bytes.NewReader(signatureBytes), bytes.NewReader(expectedPayload))
	if err != nil {
		return false, fmt.Errorf("signature verification failed: %w", err)
	}

	return true, nil
}

// GenerateExpectedPayload creates the expected payload for a record CID's digest.
// This is a convenience function that combines ConvertCIDToDigest and GeneratePayload.
func GenerateExpectedPayload(digest string) ([]byte, error) {
	return GeneratePayload(digest)
}
