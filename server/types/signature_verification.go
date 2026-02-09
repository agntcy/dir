// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import "time"

// SignatureVerificationObject represents a cached signature verification result.
// Each entry caches the verification result for a specific signature on a record.
type SignatureVerificationObject interface {
	// GetID returns the unique identifier for this verification cache entry.
	GetID() string

	// GetRecordCID returns the CID of the record that was signed.
	GetRecordCID() string

	// GetSignatureDigest returns the digest/hash of the signature blob.
	// This is used to uniquely identify the signature within a record's referrers.
	GetSignatureDigest() string

	// GetSignerType returns the type of signer: "key" or "oidc".
	GetSignerType() string

	// GetPublicKey returns the public key used for verification (for key-based signatures).
	GetPublicKey() string

	// GetOIDCIssuer returns the OIDC issuer URL (for OIDC-based signatures).
	GetOIDCIssuer() string

	// GetOIDCIdentity returns the OIDC identity/subject (for OIDC-based signatures).
	GetOIDCIdentity() string

	// GetVerified returns whether the signature was successfully verified.
	GetVerified() bool

	// GetError returns the error message if verification failed.
	GetError() string

	// GetCreatedAt returns when this cache entry was created.
	GetCreatedAt() time.Time
}

// SignatureVerificationInput is used to create or query signature verifications.
type SignatureVerificationInput struct {
	RecordCID       string
	SignatureDigest string
	SignerType      string
	PublicKey       string
	OIDCIssuer      string
	OIDCIdentity    string
	Verified        bool
	Error           string
}
