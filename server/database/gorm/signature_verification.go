// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"errors"
	"fmt"
	"time"

	"github.com/agntcy/dir/server/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ErrSignatureVerificationNotFound is returned when no cached verification is found.
var ErrSignatureVerificationNotFound = errors.New("signature verification not found")

// SignatureVerification stores cached signature verification results.
// Each entry represents the verification result for a specific signature on a record.
// The composite key (record_cid, signature_digest) uniquely identifies a signature.
type SignatureVerification struct {
	ID              string `gorm:"primarykey"`
	CreatedAt       time.Time
	DeletedAt       gorm.DeletedAt `gorm:"index"` // soft delete support
	RecordCID       string         `gorm:"column:record_cid;not null;index:idx_sig_verification_composite,unique"`
	SignatureDigest string         `gorm:"column:signature_digest;not null;index:idx_sig_verification_composite,unique"`
	SignerType      string         `gorm:"column:signer_type;not null"` // "key" or "oidc"
	PublicKey       string         `gorm:"column:public_key"`           // for key-based signatures
	OIDCIssuer      string         `gorm:"column:oidc_issuer"`          // for OIDC-based signatures
	OIDCIdentity    string         `gorm:"column:oidc_identity"`        // for OIDC-based signatures
	Verified        bool           `gorm:"column:verified;not null"`
	Error           string         `gorm:"column:error"` // error message if verification failed
}

// Implement types.SignatureVerificationObject interface.

func (sv *SignatureVerification) GetID() string {
	return sv.ID
}

func (sv *SignatureVerification) GetRecordCID() string {
	return sv.RecordCID
}

func (sv *SignatureVerification) GetSignatureDigest() string {
	return sv.SignatureDigest
}

func (sv *SignatureVerification) GetSignerType() string {
	return sv.SignerType
}

func (sv *SignatureVerification) GetPublicKey() string {
	return sv.PublicKey
}

func (sv *SignatureVerification) GetOIDCIssuer() string {
	return sv.OIDCIssuer
}

func (sv *SignatureVerification) GetOIDCIdentity() string {
	return sv.OIDCIdentity
}

func (sv *SignatureVerification) GetVerified() bool {
	return sv.Verified
}

func (sv *SignatureVerification) GetError() string {
	return sv.Error
}

func (sv *SignatureVerification) GetCreatedAt() time.Time {
	return sv.CreatedAt
}

// CreateSignatureVerification creates a new signature verification cache entry.
func (d *DB) CreateSignatureVerification(input types.SignatureVerificationInput) error {
	sv := &SignatureVerification{
		ID:              uuid.New().String(),
		RecordCID:       input.RecordCID,
		SignatureDigest: input.SignatureDigest,
		SignerType:      input.SignerType,
		PublicKey:       input.PublicKey,
		OIDCIssuer:      input.OIDCIssuer,
		OIDCIdentity:    input.OIDCIdentity,
		Verified:        input.Verified,
		Error:           input.Error,
	}

	if err := d.gormDB.Create(sv).Error; err != nil {
		return fmt.Errorf("failed to create signature verification: %w", err)
	}

	logger.Debug("Created signature verification cache",
		"record_cid", sv.RecordCID,
		"signature_digest", sv.SignatureDigest,
		"signer_type", sv.SignerType,
		"verified", sv.Verified)

	return nil
}

// GetSignatureVerification retrieves a cached verification by record CID and signature digest.
func (d *DB) GetSignatureVerification(recordCID, signatureDigest string) (types.SignatureVerificationObject, error) {
	var sv SignatureVerification
	err := d.gormDB.Where("record_cid = ? AND signature_digest = ?", recordCID, signatureDigest).First(&sv).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSignatureVerificationNotFound
		}

		return nil, fmt.Errorf("failed to get signature verification: %w", err)
	}

	return &sv, nil
}

// GetSignatureVerificationsByRecord retrieves all cached verifications for a record.
func (d *DB) GetSignatureVerificationsByRecord(recordCID string) ([]types.SignatureVerificationObject, error) {
	var svs []SignatureVerification
	if err := d.gormDB.Where("record_cid = ?", recordCID).Find(&svs).Error; err != nil {
		return nil, fmt.Errorf("failed to get signature verifications: %w", err)
	}

	result := make([]types.SignatureVerificationObject, len(svs))
	for i := range svs {
		result[i] = &svs[i]
	}

	return result, nil
}

// DeleteSignatureVerification deletes a cached verification by record CID and signature digest.
func (d *DB) DeleteSignatureVerification(recordCID, signatureDigest string) error {
	result := d.gormDB.Where("record_cid = ? AND signature_digest = ?", recordCID, signatureDigest).
		Delete(&SignatureVerification{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete signature verification: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrSignatureVerificationNotFound
	}

	logger.Debug("Deleted signature verification cache",
		"record_cid", recordCID,
		"signature_digest", signatureDigest)

	return nil
}

// DeleteSignatureVerificationsByRecord deletes all cached verifications for a record.
func (d *DB) DeleteSignatureVerificationsByRecord(recordCID string) error {
	result := d.gormDB.Where("record_cid = ?", recordCID).Delete(&SignatureVerification{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete signature verifications: %w", result.Error)
	}

	logger.Debug("Deleted all signature verification caches for record",
		"record_cid", recordCID,
		"count", result.RowsAffected)

	return nil
}
