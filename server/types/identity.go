// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import "time"

// Claim roles, as stored in the claims table.
const (
	// ClaimRoleIdentity is a claim about the record's own identity.
	ClaimRoleIdentity = "identity"

	// ClaimRoleOwner is a claim about the record's owning entity.
	ClaimRoleOwner = "owner"
)

// Claim verification statuses, as stored in the claims table.
const (
	ClaimStatusVerified = "verified"
	ClaimStatusFailed   = "failed"
)

// IdentityClaimObject is the last verification result of one claim of a record,
// keyed by (record CID, role).
type IdentityClaimObject interface {
	GetRecordCID() string
	GetRole() string // ClaimRoleIdentity or ClaimRoleOwner
	GetSubject() string
	GetStatus() string // ClaimStatusVerified or ClaimStatusFailed
	GetError() string
	GetVerifiedAt() time.Time
}
