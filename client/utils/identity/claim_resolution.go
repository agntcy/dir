// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package identity

import (
	"context"
	"crypto"
	"time"

	identityv1 "github.com/agntcy/dir/api/identity/v1"
)

// ClaimResolution contains authenticated keys and the deadline of accepted
// record-specific evidence. The caller must still verify the native signature.
type ClaimResolution struct {
	PublicKeys []crypto.PublicKey
	ValidUntil time.Time
}

// ClaimResolver is an optional record-aware extension to key resolution. It
// obtains keys and required evidence together without changing Resolver. Its
// result applies only to the exact claim and must not be cached by subject.
type ClaimResolver interface {
	ResolveClaim(context.Context, *identityv1.Claim) (ClaimResolution, error)
}
