// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"errors"
	"fmt"
	"time"

	"github.com/agntcy/dir/server/types"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Claim status constants.
const (
	ClaimStatusVerified = "verified"
	ClaimStatusFailed   = "failed"
)

// ErrClaimNotFound is returned when no claim is found for a record.
var ErrClaimNotFound = errors.New("claim not found")

// Claim stores the verification result of a record's identity or ownership
// claim (one row per record_cid + role).
type Claim struct {
	ID         uint `gorm:"primarykey"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	RecordCID  string `gorm:"column:record_cid;not null;uniqueIndex:idx_claims_record_role"`
	Role       string `gorm:"not null;uniqueIndex:idx_claims_record_role;index"` // "identity" or "owner"
	Subject    string `gorm:"not null;index"`
	Status     string `gorm:"not null;index"` // "verified" or "failed"
	Error      string
	VerifiedAt *time.Time
}

// TableName overrides gorm's default pluralization so the table stays named
// "claims" regardless of the Go type name.
func (Claim) TableName() string { return "claims" }

// Implement types.ClaimObject.

func (c *Claim) GetRecordCID() string      { return c.RecordCID }
func (c *Claim) GetSubject() string        { return c.Subject }
func (c *Claim) GetStatus() string         { return c.Status }
func (c *Claim) GetError() string          { return c.Error }
func (c *Claim) GetVerifiedAt() *time.Time { return c.VerifiedAt }

// UpsertClaim creates or updates the claim verification result for a
// record's role (one row per record_cid + role).
func (d *DB) UpsertClaim(role string, claim types.ClaimObject) error {
	row := &Claim{
		RecordCID:  claim.GetRecordCID(),
		Role:       role,
		Subject:    claim.GetSubject(),
		Status:     claim.GetStatus(),
		Error:      claim.GetError(),
		VerifiedAt: claim.GetVerifiedAt(),
	}

	result := d.gormDB.
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "record_cid"}, {Name: "role"}},
			DoUpdates: clause.AssignmentColumns([]string{"subject", "status", "error", "verified_at", "updated_at"}),
		}).
		Create(row)
	if result.Error != nil {
		return fmt.Errorf("failed to upsert %s claim: %w", role, result.Error)
	}

	logger.Debug("Upserted claim", "record_cid", row.RecordCID, "role", role, "status", row.Status)

	return nil
}

// RemoveClaims deletes all claims (both roles) for a record.
func (d *DB) RemoveClaims(recordCID string) error {
	if err := d.gormDB.Where("record_cid = ?", recordCID).Delete(&Claim{}).Error; err != nil {
		return fmt.Errorf("failed to remove claims for record %s: %w", recordCID, err)
	}

	return nil
}

// GetClaimByCID retrieves a record's claim for the given role.
func (d *DB) GetClaimByCID(cid, role string) (types.ClaimObject, error) {
	var claim Claim
	if err := d.gormDB.Where("record_cid = ? AND role = ?", cid, role).First(&claim).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrClaimNotFound
		}

		return nil, fmt.Errorf("failed to get %s claim: %w", role, err)
	}

	return &claim, nil
}
