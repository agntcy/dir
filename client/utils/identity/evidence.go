// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package identity

import (
	"context"
	"time"

	identityv1 "github.com/agntcy/dir/api/identity/v1"
)

// ClaimEvidenceVerifier checks record-specific evidence after the native claim
// signature has verified. Its deadline bounds how long the result may be used.
// This is separate from key resolution, whose cache is shared across records.
type ClaimEvidenceVerifier interface {
	VerifyEvidence(ctx context.Context, claim *identityv1.Claim) (time.Time, error)
}
