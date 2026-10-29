// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"errors"
	"fmt"
	"strings"
	"time"

	coretypes "github.com/agntcy/dir/api/core/types"
	"github.com/agntcy/dir/server/types"
	gormlib "gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SignatureVerification stores one signer verification result (one row per signer).
// Rows are keyed by (record_cid, signer_key).
type SignatureVerification struct {
	RecordCID               string    `gorm:"column:record_cid;primaryKey;not null"`
	SignerKey               string    `gorm:"column:signer_key;primaryKey;not null"`
	Status                  string    `gorm:"column:status;not null;index"` // "verified" or "failed"
	ErrorMessage            string    `gorm:"column:error_message"`
	Signature               string    `gorm:"column:signature"`                 // encoded signature data
	ContentType             string    `gorm:"column:content_type;not null"`     // media type of the signed content (e.g. "application/vnd.oci.image.manifest.v1+json")
	SignerType              string    `gorm:"column:signer_type"`               // "oidc" or "key"
	SignerIssuer            string    `gorm:"column:signer_issuer"`             // OIDC issuer
	SignerSubject           string    `gorm:"column:signer_subject"`            // OIDC subject
	SignerCertificateIssuer string    `gorm:"column:signer_certificate_issuer"` // X.509 certificate issuer (e.g. CN=sigstore-intermediate,O=sigstore.dev)
	SignerPublicKey         string    `gorm:"column:signer_public_key"`         // PEM or ref for key type
	SignerAlgorithm         string    `gorm:"column:signer_algorithm"`          // e.g. RSA-SHA256, Ed25519
	CreatedAt               time.Time `gorm:"column:created_at;not null"`
	UpdatedAt               time.Time `gorm:"column:updated_at;not null"`
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

func (s *SignatureVerification) GetSignerCertificateIssuer() string {
	return s.SignerCertificateIssuer
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

func (s *SignatureVerification) GetSignature() string {
	return s.Signature
}

func (s *SignatureVerification) GetContentType() string {
	return s.ContentType
}

// CreateSignatureVerification creates a new signature verification row.
func (d *DB) CreateSignatureVerification(verification types.SignatureVerificationObject) error {
	now := time.Now()

	sv := &SignatureVerification{
		RecordCID:               verification.GetRecordCID(),
		SignerKey:               verification.GetSignerKey(),
		Status:                  verification.GetStatus(),
		ErrorMessage:            verification.GetErrorMessage(),
		SignerType:              verification.GetSignerType(),
		SignerIssuer:            verification.GetSignerIssuer(),
		SignerSubject:           verification.GetSignerSubject(),
		SignerCertificateIssuer: verification.GetSignerCertificateIssuer(),
		SignerPublicKey:         verification.GetSignerPublicKey(),
		SignerAlgorithm:         verification.GetSignerAlgorithm(),
		Signature:               verification.GetSignature(),
		ContentType:             verification.GetContentType(),
		CreatedAt:               now,
		UpdatedAt:               now,
	}
	if err := d.gormDB.Create(sv).Error; err != nil {
		return fmt.Errorf("failed to create signature verification: %w", err)
	}

	logger.Debug("Created signature verification", "record_cid", sv.RecordCID, "status", sv.Status)

	return nil
}

// UpdateSignatureVerification updates an existing row by (record_cid, signature).
//
// The signature task uses this to mark a signer failed once it stops appearing
// in a verification run, which is a conclusion changing rather than a row being
// refreshed, so it supersedes the record's verdicts.
func (d *DB) UpdateSignatureVerification(verification types.SignatureVerificationObject) error {
	now := time.Now()

	err := d.gormDB.Transaction(func(tx *gormlib.DB) error {
		var existing SignatureVerification

		err := tx.Where("record_cid = ? AND signer_key = ?",
			verification.GetRecordCID(), verification.GetSignerKey()).Take(&existing).Error
		if errors.Is(err, gormlib.ErrRecordNotFound) {
			return fmt.Errorf("signature verification not found: record_cid=%s", verification.GetRecordCID())
		}

		if err != nil {
			return fmt.Errorf("load existing signature verification: %w", err)
		}

		result := tx.Model(&SignatureVerification{}).
			Where("record_cid = ? AND signer_key = ?", verification.GetRecordCID(), verification.GetSignerKey()).
			Updates(map[string]any{
				"status":        verification.GetStatus(),
				"error_message": verification.GetErrorMessage(),
				"updated_at":    now,
			})
		if result.Error != nil {
			return fmt.Errorf("update row: %w", result.Error)
		}

		// Only the status is compared, rather than the whole projected
		// signature fact: this writes the status and the error message and
		// leaves the signer fields as they are, and the error message is not
		// something a policy reads.
		if existing.Status == verification.GetStatus() {
			return nil
		}

		_, err = supersedePolicyEvaluations(tx, verification.GetRecordCID(), types.FactGroupSignature)

		return err
	})
	if err != nil {
		return fmt.Errorf("failed to update signature verification: %w", err)
	}

	logger.Debug("Updated signature verification", "record_cid", verification.GetRecordCID(), "status", verification.GetStatus())

	return nil
}

// UpsertSignatureVerification inserts or updates a row keyed by (record_cid, signature).
func (d *DB) UpsertSignatureVerification(verification types.SignatureVerificationObject) error {
	now := time.Now()
	sv := &SignatureVerification{
		RecordCID:               verification.GetRecordCID(),
		SignerKey:               verification.GetSignerKey(),
		Status:                  verification.GetStatus(),
		ErrorMessage:            verification.GetErrorMessage(),
		SignerType:              verification.GetSignerType(),
		SignerIssuer:            verification.GetSignerIssuer(),
		SignerSubject:           verification.GetSignerSubject(),
		SignerCertificateIssuer: verification.GetSignerCertificateIssuer(),
		SignerPublicKey:         verification.GetSignerPublicKey(),
		SignerAlgorithm:         verification.GetSignerAlgorithm(),
		Signature:               verification.GetSignature(),
		ContentType:             verification.GetContentType(),
		CreatedAt:               now,
		UpdatedAt:               now,
	}

	// The write and the verdicts it supersedes share a transaction. A
	// verification that commits while its invalidation does not leaves a
	// verdict that is wrong about the record with nothing left to retry it.
	err := d.gormDB.Transaction(func(tx *gormlib.DB) error {
		var existing SignatureVerification

		err := tx.Where("record_cid = ? AND signer_key = ?", sv.RecordCID, sv.SignerKey).Take(&existing).Error
		if err != nil && !errors.Is(err, gormlib.ErrRecordNotFound) {
			return fmt.Errorf("load existing signature verification: %w", err)
		}

		existed := err == nil

		err = tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "record_cid"}, {Name: "signer_key"}},
			DoUpdates: clause.AssignmentColumns([]string{"status", "error_message", "signer_type", "signer_issuer", "signer_subject", "signer_certificate_issuer", "signer_public_key", "signer_algorithm", "signature", "content_type", "updated_at"}),
		}).Create(sv).Error
		if err != nil {
			return err
		}

		if existed && !signatureVerificationFactsChanged(&existing, sv) {
			return nil
		}

		_, err = supersedePolicyEvaluations(tx, sv.RecordCID, types.FactGroupSignature)

		return err
	})
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

// InvalidateSignatureVerificationsForRecord deletes all signature verification rows for a record so the reconciler will pick it up for re-verification.
func (d *DB) InvalidateSignatureVerificationsForRecord(recordCID string) error {
	if err := d.gormDB.Where("record_cid = ?", recordCID).Delete(&SignatureVerification{}).Error; err != nil {
		return fmt.Errorf("failed to invalidate signature verifications for record %s: %w", recordCID, err)
	}

	return nil
}

// GetRecordsNeedingSignatureVerification returns signed records that have no verification or expired verification.
func (d *DB) GetRecordsNeedingSignatureVerification(ttl time.Duration) ([]coretypes.Record, error) {
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

	result := make([]coretypes.Record, len(records))
	for i := range records {
		result[i] = &records[i]
	}

	return result, nil
}

func convertSignatures(sigs []SignatureVerification) []coretypes.ObjectSignature {
	result := make([]coretypes.ObjectSignature, len(sigs))
	for i := range sigs {
		result[i] = &sigs[i]
	}

	return result
}

// signatureStatusConditions maps each signature status to the SQL that selects
// records in it. A record with at least one verified signer is verified, even
// when other signers failed. A signed record with no verification row matches
// none of them.
var signatureStatusConditions = map[string]string{
	types.SignatureStatusVerified: "EXISTS (SELECT 1 FROM signature_verifications sv WHERE sv.record_cid = records.record_cid AND sv.status = 'verified')",
	types.SignatureStatusFailed: "EXISTS (SELECT 1 FROM signature_verifications sv WHERE sv.record_cid = records.record_cid) AND " +
		"NOT EXISTS (SELECT 1 FROM signature_verifications sv WHERE sv.record_cid = records.record_cid AND sv.status = 'verified')",
	types.SignatureStatusUnsigned: "records.signed = false",
}

// applySignatureStatuses keeps records whose signature status is any of statuses.
// Unrecognized statuses match nothing, so a bad value narrows the result instead
// of silently dropping the filter.
func applySignatureStatuses(query *gormlib.DB, statuses []string) *gormlib.DB {
	conditions := make([]string, 0, len(statuses))

	for _, status := range statuses {
		if condition, ok := signatureStatusConditions[status]; ok {
			conditions = append(conditions, "("+condition+")")
		}
	}

	if len(conditions) == 0 {
		return query.Where("1 = 0")
	}

	return query.Where("(" + strings.Join(conditions, " OR ") + ")")
}
