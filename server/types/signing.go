// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"context"

	corev1 "github.com/agntcy/dir/api/core/v1"
)

// SigningAPI handles signature operations for records.
type SigningAPI interface {
	// Verify verifies a record signature.
	// Returns true if the signature is valid and trusted.
	Verify(ctx context.Context, recordCID string) (bool, error)

	// PushSignature attaches a signature to a record using cosign.
	PushSignature(ctx context.Context, recordCID string, referrer *corev1.RecordReferrer) error

	// UploadPublicKey uploads a public key to the registry for signature verification.
	// For Zot registries, this uploads to the _cosign directory.
	UploadPublicKey(ctx context.Context, referrer *corev1.RecordReferrer) error

	// IsZotRegistry returns true if configured for a Zot registry.
	IsZotRegistry() bool
}
