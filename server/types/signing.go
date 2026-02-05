// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"context"
)

// SigningAPI handles signature operations for records.
type SigningAPI interface {
	// Verify verifies a record signature.
	// Returns true if the signature is valid and trusted, along with metadata about the signer.
	Verify(ctx context.Context, recordCID string) (bool, map[string]string, error)
}
