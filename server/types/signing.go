// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"context"

	signv1 "github.com/agntcy/dir/api/sign/v1"
)

// SigningAPI handles signature operations for records.
type SigningAPI interface {
	// Verify verifies record signatures and returns information about all valid signers.
	// The options parameter specifies optional verification criteria (key or OIDC identity).
	// If options is nil, all valid signatures are returned.
	// Returns a list of signers for all valid signatures, along with legacy metadata.
	Verify(ctx context.Context, recordCID string, options *signv1.VerifyOptions) (*VerifyResult, error)
}

// VerifyResult contains the result of signature verification.
type VerifyResult struct {
	// Success indicates whether at least one valid signature was found.
	Success bool

	// Signers contains information about all valid signers.
	// Each entry represents one valid signature on the record.
	Signers []*signv1.SignerInfo

	// ErrorMessage contains the error message if verification failed.
	ErrorMessage string

	// LegacyMetadata contains legacy signer metadata for backward compatibility.
	// Deprecated: Use Signers field instead.
	LegacyMetadata map[string]string
}
