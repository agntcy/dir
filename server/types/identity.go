// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import "time"

// Claim roles: a record's own identity vs. its owning entity.
const (
	ClaimRoleIdentity = "identity"
	ClaimRoleOwner    = "owner"
)

// ClaimObject represents a record identity/ownership claim verification result.
type ClaimObject interface {
	GetRecordCID() string
	GetSubject() string
	GetStatus() string
	GetError() string
	GetVerifiedAt() *time.Time
}

// IdentityDatabaseAPI handles management of record identity/ownership claims.
type IdentityDatabaseAPI interface {
	// UpsertClaim creates or updates the claim verification result for a
	// record's role (ClaimRoleIdentity or ClaimRoleOwner).
	UpsertClaim(role string, claim ClaimObject) error

	// RemoveClaims deletes all claims (both roles) for a record.
	RemoveClaims(recordCID string) error

	// GetClaimByCID retrieves a record's claim for the given role.
	GetClaimByCID(cid, role string) (ClaimObject, error)
}
