// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package verification provides domain ownership verification for OASF records.
// It implements DNS TXT and well-known file verification inspired by AT Protocol
// and ACME DNS-01 challenge patterns.
package verification

import "time"

// PublicKey represents a public key extracted from DNS TXT or well-known file.
type PublicKey struct {
	// ID is an optional identifier for the key (used in well-known format).
	ID string

	// Type is the key algorithm (e.g., "ed25519", "ecdsa-p256").
	Type string

	// Key is the raw public key bytes (decoded from base64).
	Key []byte

	// KeyBase64 is the original base64-encoded key string.
	KeyBase64 string
}

// Result represents the outcome of a domain verification attempt.
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

// WellKnownFile represents the structure of /.well-known/oasf.json
type WellKnownFile struct {
	// Version is the format version (currently 1).
	Version int `json:"version"`

	// Keys is the list of public keys for this domain.
	Keys []WellKnownKey `json:"keys"`
}

// WellKnownKey represents a key entry in the well-known file.
type WellKnownKey struct {
	// ID is an optional identifier for the key.
	ID string `json:"id,omitempty"`

	// Type is the key algorithm (e.g., "ed25519").
	Type string `json:"type"`

	// PublicKey is the base64-encoded public key.
	PublicKey string `json:"publicKey"`
}

// VerificationMethod represents the method used to verify domain ownership.
type VerificationMethod string

const (
	// MethodDNS indicates verification via DNS TXT record.
	MethodDNS VerificationMethod = "dns"

	// MethodWellKnown indicates verification via well-known file.
	MethodWellKnown VerificationMethod = "wellknown"

	// MethodNone indicates no verification was possible.
	MethodNone VerificationMethod = "none"
)

// DNSRecordPrefix is the subdomain prefix for OASF DNS TXT records.
const DNSRecordPrefix = "_oasf."

// WellKnownPath is the path for the OASF well-known file.
const WellKnownPath = "/.well-known/oasf.json"

// DefaultHTTPTimeout is the default timeout for HTTP requests.
const DefaultHTTPTimeout = 10 * time.Second
