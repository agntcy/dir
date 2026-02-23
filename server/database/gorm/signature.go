// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"fmt"
	"time"

	"github.com/agntcy/dir/server/types"
	"gorm.io/gorm/clause"
)

// SignatureVerification stores one signer verification result (one row per signer).
// Rows are keyed by (record_cid, signer_key).
type SignatureVerification struct {
	RecordCID       string    `gorm:"column:record_cid;primaryKey;not null"`
	SignerKey       string    `gorm:"column:signer_key;primaryKey;not null"`
	Status          string    `gorm:"column:status;not null;index"` // "verified" or "failed"
	ErrorMessage    string    `gorm:"column:error_message"`
	SignerType      string    `gorm:"column:signer_type"`       // "oidc" or "key"
	SignerIssuer    string    `gorm:"column:signer_issuer"`     // OIDC issuer
	SignerSubject   string    `gorm:"column:signer_subject"`    // OIDC subject
	SignerPublicKey string    `gorm:"column:signer_public_key"` // PEM or ref for key type
	SignerAlgorithm string    `gorm:"column:signer_algorithm"`  // e.g. RSA-SHA256, Ed25519
	CreatedAt       time.Time `gorm:"column:created_at;not null"`
	UpdatedAt       time.Time `gorm:"column:updated_at;not null"`
}

// Ensure SignatureVerification implements types.SignatureVerificationObject.
var _ types.SignatureVerificationObject = (*SignatureVerification)(nil)

func (s *SignatureVerification) GetRecordCID() string {
	return s.RecordCID
}

func (s *SignatureVerification) GetSignerKey() string {
	return s.SignerKey
}

func (s *SignatureVerification) GetStatus() string {
	return s.Status
}

func (s *SignatureVerification) GetErrorMessage() string {
	return s.ErrorMessage
}

func (s *SignatureVerification) GetSignerType() string {
	return s.SignerType
}

func (s *SignatureVerification) GetSignerIssuer() string {
	return s.SignerIssuer
}

func (s *SignatureVerification) GetSignerSubject() string {
	return s.SignerSubject
}

func (s *SignatureVerification) GetSignerPublicKey() string {
	return s.SignerPublicKey
}

func (s *SignatureVerification) GetSignerAlgorithm() string {
	return s.SignerAlgorithm
}

func (s *SignatureVerification) GetCreatedAt() time.Time {
	return s.CreatedAt
}

func (s *SignatureVerification) GetUpdatedAt() time.Time {
	return s.UpdatedAt
}

// CreateSignatureVerification creates a new signature verification row.
func (d *DB) CreateSignatureVerification(verification types.SignatureVerificationObject) error {
	now := time.Now()

	sv := &SignatureVerification{
		RecordCID:       verification.GetRecordCID(),
		SignerKey:       verification.GetSignerKey(),
		Status:          verification.GetStatus(),
		ErrorMessage:    verification.GetErrorMessage(),
		SignerType:      verification.GetSignerType(),
		SignerIssuer:    verification.GetSignerIssuer(),
		SignerSubject:   verification.GetSignerSubject(),
		SignerPublicKey: verification.GetSignerPublicKey(),
		SignerAlgorithm: verification.GetSignerAlgorithm(),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := d.gormDB.Create(sv).Error; err != nil {
		return fmt.Errorf("failed to create signature verification: %w", err)
	}

	logger.Debug("Created signature verification", "record_cid", sv.RecordCID, "status", sv.Status)

	return nil
}

// UpdateSignatureVerification updates an existing row by (record_cid, signature).
func (d *DB) UpdateSignatureVerification(verification types.SignatureVerificationObject) error {
	now := time.Now()

	result := d.gormDB.Model(&SignatureVerification{}).
		Where("record_cid = ? AND signer_key = ?", verification.GetRecordCID(), verification.GetSignerKey()).
		Updates(map[string]any{
			"status":        verification.GetStatus(),
			"error_message": verification.GetErrorMessage(),
			"updated_at":    now,
		})
	if result.Error != nil {
		return fmt.Errorf("failed to update signature verification: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("signature verification not found: record_cid=%s", verification.GetRecordCID())
	}

	logger.Debug("Updated signature verification", "record_cid", verification.GetRecordCID(), "status", verification.GetStatus())

	return nil
}

// UpsertSignatureVerification inserts or updates a row keyed by (record_cid, signature).
func (d *DB) UpsertSignatureVerification(verification types.SignatureVerificationObject) error {
	now := time.Now()
	sv := &SignatureVerification{
		RecordCID:       verification.GetRecordCID(),
		SignerKey:       verification.GetSignerKey(),
		Status:          verification.GetStatus(),
		ErrorMessage:    verification.GetErrorMessage(),
		SignerType:      verification.GetSignerType(),
		SignerIssuer:    verification.GetSignerIssuer(),
		SignerSubject:   verification.GetSignerSubject(),
		SignerPublicKey: verification.GetSignerPublicKey(),
		SignerAlgorithm: verification.GetSignerAlgorithm(),
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	err := d.gormDB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "record_cid"}, {Name: "signer_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"status", "error_message", "signer_type", "signer_issuer", "signer_subject", "signer_public_key", "signer_algorithm", "updated_at"}),
	}).Create(sv).Error
	if err != nil {
		return fmt.Errorf("upsert signature verification: %w", err)
	}

	logger.Debug("Upserted signature verification", "record_cid", sv.RecordCID, "status", sv.Status)

	return nil
}

// GetSignatureVerificationsByRecordCID returns all signature verification rows for a record.
func (d *DB) GetSignatureVerificationsByRecordCID(recordCID string) ([]types.SignatureVerificationObject, error) {
	var rows []SignatureVerification
	if err := d.gormDB.Where("record_cid = ?", recordCID).Order("updated_at DESC").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to get signature verifications: %w", err)
	}

	result := make([]types.SignatureVerificationObject, len(rows))
	for i := range rows {
		result[i] = &rows[i]
	}

	return result, nil
}

// GetRecordsNeedingSignatureVerification returns signed records that have no verification or expired verification.
func (d *DB) GetRecordsNeedingSignatureVerification(ttl time.Duration) ([]types.Record, error) {
	expiredBefore := time.Now().Add(-ttl)

	// Subquery: record_cids that have at least one verification row updated after expiredBefore.
	// We need records that are signed AND (have no rows OR max(updated_at) < expiredBefore).
	var records []Record

	err := d.gormDB.Table("records").
		Where("records.signed = ?", true).
		Where(`NOT EXISTS (
			SELECT 1 FROM signature_verifications sv
			WHERE sv.record_cid = records.record_cid
			AND sv.updated_at >= ?
		)`, expiredBefore).
		Find(&records).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get records needing signature verification: %w", err)
	}

	result := make([]types.Record, len(records))
	for i := range records {
		result[i] = &records[i]
	}

	return result, nil
}
