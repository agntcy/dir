// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package naming provides name ownership verification for OASF records.
// It implements DNS TXT and JWKS (RFC 7517) verification inspired by AT Protocol
// and ACME DNS-01 challenge patterns.
package naming

import "time"

// PublicKey represents a public key extracted from DNS TXT or JWKS.
type PublicKey struct {
	// ID is an optional identifier for the key (kid in JWK, or generated for DNS).
	ID string

	// Type is the key algorithm (e.g., "ed25519", "ecdsa-p256", "rsa").
	Type string

	// Key is the raw public key bytes in DER format for comparison.
	Key []byte

	// KeyBase64 is the original base64-encoded key string (for DNS TXT records).
	KeyBase64 string
}

// Result represents the outcome of a name verification attempt.
type Result struct {
	// Verified is true if the signing key matches a domain-published key.
	Verified bool

	// Domain is the domain that was verified.
	Domain string

	// Method is how the keys were retrieved ("dns" or "wellknown").
	Method string

	// VerifiedAt is when the verification was performed.
	VerifiedAt time.Time

	// Error contains the error message if verification failed.
	Error string

	// MatchedKeyID is the ID of the key that matched (if available).
	MatchedKeyID string
}

// VerificationMethod represents the method used to verify name ownership.
type VerificationMethod string

const (
	// MethodDNS indicates verification via DNS TXT record.
	MethodDNS VerificationMethod = "dns"

	// MethodWellKnown indicates verification via JWKS well-known file (RFC 7517).
	MethodWellKnown VerificationMethod = "wellknown"

	// MethodNone indicates no verification was possible.
	MethodNone VerificationMethod = "none"
)
