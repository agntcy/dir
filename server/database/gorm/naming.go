// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"errors"
	"fmt"
	"time"

	"github.com/agntcy/dir/server/types"
	"gorm.io/gorm"
)

// NameVerification status constants.
const (
	VerificationStatusVerified = "verified"
	VerificationStatusFailed   = "failed"
)

// ErrVerificationNotFound is returned when no verification is found for a record.
var ErrVerificationNotFound = errors.New("verification not found")

// NameVerification stores name verification result for a record (one per CID).
type NameVerification struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`                                  // soft delete support
	RecordCID string         `gorm:"column:record_cid;not null;uniqueIndex"` // one verification per record
	Method    string         `gorm:"not null"`                               // "wellknown"
	KeyID     string         // matched key ID (if successful)
	Status    string         `gorm:"not null;index"` // "verified" or "failed"
	Error     string         // error message (if failed)
}

// Implement types.NameVerificationObject interface.

func (nv *NameVerification) GetRecordCID() string {
	return nv.RecordCID
}

func (nv *NameVerification) GetMethod() string {
	return nv.Method
}

func (nv *NameVerification) GetKeyID() string {
	return nv.KeyID
}

func (nv *NameVerification) GetStatus() string {
	return nv.Status
}

func (nv *NameVerification) GetError() string {
	return nv.Error
}

func (nv *NameVerification) GetCreatedAt() time.Time {
	return nv.CreatedAt
}

func (nv *NameVerification) GetUpdatedAt() time.Time {
	return nv.UpdatedAt
}

// CreateNameVerification creates a new name verification for a record.
func (d *DB) CreateNameVerification(verification types.NameVerificationObject) error {
	nv := &NameVerification{
		RecordCID: verification.GetRecordCID(),
		Method:    verification.GetMethod(),
		KeyID:     verification.GetKeyID(),
		Status:    verification.GetStatus(),
		Error:     verification.GetError(),
	}

	if err := d.gormDB.Create(nv).Error; err != nil {
		return fmt.Errorf("failed to create name verification: %w", err)
	}

	logger.Debug("Created name verification", "record_cid", nv.RecordCID, "status", nv.Status)

	return nil
}

// UpdateNameVerification updates an existing name verification for a record.
func (d *DB) UpdateNameVerification(verification types.NameVerificationObject) error {
	result := d.gormDB.Model(&NameVerification{}).
		Where("record_cid = ?", verification.GetRecordCID()).
		Updates(map[string]any{
			"method": verification.GetMethod(),
			"key_id": verification.GetKeyID(),
			"status": verification.GetStatus(),
			"error":  verification.GetError(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update name verification: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrVerificationNotFound
	}

	logger.Debug("Updated name verification", "record_cid", verification.GetRecordCID(), "status", verification.GetStatus())

	return nil
}

// GetVerificationByCID retrieves the verification for a record.
// Returns ErrVerificationNotFound if no verification exists.
func (d *DB) GetVerificationByCID(cid string) (types.NameVerificationObject, error) {
	var nv NameVerification
	if err := d.gormDB.Where("record_cid = ?", cid).First(&nv).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrVerificationNotFound
		}

		return nil, fmt.Errorf("failed to get name verification: %w", err)
	}

	return &nv, nil
}

// GetRecordsNeedingVerification retrieves signed records with verifiable names
// that either don't have a verification or have an expired verification.
func (d *DB) GetRecordsNeedingVerification(ttl time.Duration) ([]types.Record, error) {
	expiredBefore := time.Now().Add(-ttl)

	// Query records that:
	// 1. Are signed (have a signature attached)
	// 2. Have a verifiable name prefix (http://, https://)
	// 3. Either don't have a verification OR have an expired verification
	var records []Record

	err := d.gormDB.Table("records").
		Joins("LEFT JOIN name_verifications ON records.record_cid = name_verifications.record_cid").
		Where("records.signed = ?", true).
		Where("(records.name LIKE ? OR records.name LIKE ?)",
			"http://%", "https://%").
		Where("(name_verifications.record_cid IS NULL OR name_verifications.updated_at < ?)", expiredBefore).
		Find(&records).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get records needing verification: %w", err)
	}

	result := make([]types.Record, len(records))
	for i := range records {
		result[i] = &records[i]
	}

	return result, nil
}
