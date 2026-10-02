// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"errors"
	"fmt"
	"time"

	"github.com/agntcy/dir/server/database/utils"
	"github.com/agntcy/dir/server/types"
	gormlib "gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ErrIdentityClaimNotFound is returned when a record has no claim result for a role.
var ErrIdentityClaimNotFound = errors.New("identity claim not found")

// IdentityClaim stores the last verification result of one claim of a record.
// There is one row per (record_cid, role).
type IdentityClaim struct {
	RecordCID  string    `gorm:"column:record_cid;primaryKey;not null"`
	Role       string    `gorm:"column:role;primaryKey;not null"` // "identity" or "owner"
	Subject    string    `gorm:"column:subject;not null"`
	Status     string    `gorm:"column:status;not null;index"` // "verified" or "failed"
	Error      string    `gorm:"column:error"`
	VerifiedAt time.Time `gorm:"column:verified_at;not null"`
}

// Ensure IdentityClaim implements types.IdentityClaimObject.
var _ types.IdentityClaimObject = (*IdentityClaim)(nil)

func (c *IdentityClaim) GetRecordCID() string     { return c.RecordCID }
func (c *IdentityClaim) GetRole() string          { return c.Role }
func (c *IdentityClaim) GetSubject() string       { return c.Subject }
func (c *IdentityClaim) GetStatus() string        { return c.Status }
func (c *IdentityClaim) GetError() string         { return c.Error }
func (c *IdentityClaim) GetVerifiedAt() time.Time { return c.VerifiedAt }

// UpsertIdentityClaim inserts or updates the result keyed by (record_cid, role).
func (d *DB) UpsertIdentityClaim(claim types.IdentityClaimObject) error {
	if claim.GetRole() != types.ClaimRoleIdentity && claim.GetRole() != types.ClaimRoleOwner {
		return fmt.Errorf("invalid claim role %q", claim.GetRole())
	}

	if claim.GetStatus() != types.ClaimStatusVerified && claim.GetStatus() != types.ClaimStatusFailed {
		return fmt.Errorf("invalid claim status %q", claim.GetStatus())
	}

	verifiedAt := claim.GetVerifiedAt()
	if verifiedAt.IsZero() {
		verifiedAt = time.Now()
	}

	row := &IdentityClaim{
		RecordCID:  claim.GetRecordCID(),
		Role:       claim.GetRole(),
		Subject:    claim.GetSubject(),
		Status:     claim.GetStatus(),
		Error:      claim.GetError(),
		VerifiedAt: verifiedAt,
	}

	err := d.gormDB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "record_cid"}, {Name: "role"}},
		DoUpdates: clause.AssignmentColumns([]string{"subject", "status", "error", "verified_at"}),
	}).Create(row).Error
	if err != nil {
		return fmt.Errorf("upsert identity claim: %w", err)
	}

	logger.Debug("Upserted identity claim", "record_cid", row.RecordCID, "role", row.Role, "status", row.Status)

	return nil
}

// GetIdentityClaimByCID returns the result for a record's claim of the given role.
// Returns ErrIdentityClaimNotFound if none exists.
func (d *DB) GetIdentityClaimByCID(cid, role string) (types.IdentityClaimObject, error) {
	var row IdentityClaim
	if err := d.gormDB.Where("record_cid = ? AND role = ?", cid, role).First(&row).Error; err != nil {
		if errors.Is(err, gormlib.ErrRecordNotFound) {
			return nil, ErrIdentityClaimNotFound
		}

		return nil, fmt.Errorf("failed to get identity claim: %w", err)
	}

	return &row, nil
}

// applyIdentityFilters applies the claim subject and verified filters. Each is a
// correlated subquery on identity_claims rather than a JOIN, so a record is never
// duplicated.
func applyIdentityFilters(query *gormlib.DB, cfg *types.RecordFilters) *gormlib.DB {
	roles := []struct {
		role     string
		subjects []string
		verified bool
	}{
		{types.ClaimRoleIdentity, cfg.Identities, cfg.IdentityVerified},
		{types.ClaimRoleOwner, cfg.Owners, cfg.OwnerVerified},
	}

	for _, r := range roles {
		if len(r.subjects) > 0 {
			query = applyClaimSubjects(query, r.role, r.subjects)
		}

		if r.verified {
			query = applyClaimVerified(query, r.role)
		}
	}

	return query
}

// applyClaimSubjects keeps records with a claim of role whose subject matches any pattern.
func applyClaimSubjects(query *gormlib.DB, role string, patterns []string) *gormlib.DB {
	condition, args := utils.BuildWildcardCondition("ic.subject", patterns)
	if condition == "" {
		return query
	}

	inner := "ic.record_cid = records.record_cid AND ic.role = ? AND (" + condition + ")"

	return query.Where("EXISTS (SELECT 1 FROM identity_claims ic WHERE "+inner+")", append([]any{role}, args...)...)
}

// applyClaimVerified keeps records whose claim of role has a verified result.
func applyClaimVerified(query *gormlib.DB, role string) *gormlib.DB {
	return query.Where(
		"EXISTS (SELECT 1 FROM identity_claims ic WHERE ic.record_cid = records.record_cid AND ic.role = ? AND ic.status = ?)",
		role, types.ClaimStatusVerified,
	)
}
