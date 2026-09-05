// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sqlstore

import (
	"context"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"gorm.io/gorm/clause"
)

// PushReferrer stores a referrer against its subject record. The referrer's
// payload is the same canonical JSON the OCI store writes as a blob, so
// referrer CIDs are identical across backends.
func (s *store) PushReferrer(ctx context.Context, recordCID string, referrer *corev1.RecordReferrer) (*corev1.ReferrerRef, error) {
	if referrer == nil {
		return nil, status.Error(codes.InvalidArgument, "referrer is required") //nolint:wrapcheck
	}

	if recordCID == "" {
		return nil, status.Error(codes.InvalidArgument, "record CID is required") //nolint:wrapcheck
	}

	if referrer.GetType() == "" {
		return nil, status.Error(codes.InvalidArgument, "referrer type is required") //nolint:wrapcheck
	}

	if referrer.GetRecordRef() == nil {
		referrer.RecordRef = &corev1.RecordRef{Cid: recordCID}
	} else if referrer.GetRecordRef().GetCid() != recordCID {
		return nil, status.Error(codes.InvalidArgument, "referrer's record CID must match record CID") //nolint:wrapcheck
	}

	if _, err := s.Lookup(ctx, &corev1.RecordRef{Cid: recordCID}); err != nil {
		return nil, status.Errorf(codes.NotFound, "record not found for CID %s: %v", recordCID, err)
	}

	payload, err := referrer.Marshal()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal referrer: %v", err)
	}

	referrerCID, err := cidFromBytes(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to derive referrer CID: %v", err)
	}

	row := &Referrer{
		ReferrerCID: referrerCID,
		SubjectCID:  recordCID,
		Type:        referrer.GetType(),
		Payload:     payload,
	}

	if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(row).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to store referrer %s: %v", referrerCID, err)
	}

	logger.Debug("Referrer pushed to SQL store", "recordCID", recordCID, "referrerCID", referrerCID, "type", referrer.GetType())

	return &corev1.ReferrerRef{Cid: referrerCID}, nil
}

// WalkReferrers calls walkFn for every referrer of recordCID, optionally
// filtered to a single referrer type.
func (s *store) WalkReferrers(ctx context.Context, recordCID string, referrerType string, walkFn func(*corev1.RecordReferrer) error) error {
	if recordCID == "" {
		return status.Error(codes.InvalidArgument, "record CID is required") //nolint:wrapcheck
	}

	if walkFn == nil {
		return status.Error(codes.InvalidArgument, "walkFn is required") //nolint:wrapcheck
	}

	if _, err := s.Lookup(ctx, &corev1.RecordRef{Cid: recordCID}); err != nil {
		return status.Errorf(codes.NotFound, "failed to resolve record for CID %s: %v", recordCID, err)
	}

	rows, err := s.findReferrers(ctx, recordCID, "", referrerType)
	if err != nil {
		return err
	}

	for _, row := range rows {
		referrer := &corev1.RecordReferrer{}

		if err := protojson.Unmarshal(row.Payload, referrer); err != nil {
			logger.Error("Failed to unmarshal referrer", "referrerCID", row.ReferrerCID, "error", err)

			continue // Skip this referrer but continue with others
		}

		referrer.ReferrerRef = &corev1.ReferrerRef{Cid: row.ReferrerCID}

		if err := walkFn(referrer); err != nil {
			return err
		}
	}

	return nil
}

// DeleteReferrer removes referrers of recordCID, narrowed by referrerCID and
// referrerType when given, and returns the CIDs it removed.
func (s *store) DeleteReferrer(ctx context.Context, recordCID string, referrerCID string, referrerType string) ([]string, error) {
	if recordCID == "" {
		return nil, status.Error(codes.InvalidArgument, "record CID is required") //nolint:wrapcheck
	}

	rows, err := s.findReferrers(ctx, recordCID, referrerCID, referrerType)
	if err != nil {
		return nil, err
	}

	if len(rows) == 0 {
		return []string{}, nil
	}

	cids := make([]string, 0, len(rows))
	for _, row := range rows {
		cids = append(cids, row.ReferrerCID)
	}

	err = s.db.WithContext(ctx).
		Where("subject_cid = ? AND referrer_cid IN ?", recordCID, cids).
		Delete(&Referrer{}).Error
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete referrers for CID %s: %v", recordCID, err)
	}

	logger.Debug("Referrers deleted from SQL store", "recordCID", recordCID, "count", len(cids))

	return cids, nil
}

func (s *store) findReferrers(ctx context.Context, subjectCID, referrerCID, referrerType string) ([]Referrer, error) {
	query := s.db.WithContext(ctx).Model(&Referrer{}).Where("subject_cid = ?", subjectCID)

	if referrerType != "" {
		query = query.Where("type = ?", referrerType)
	}

	if referrerCID != "" {
		query = query.Where("referrer_cid = ?", referrerCID)
	}

	rows := []Referrer{}
	if err := query.Find(&rows).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list referrers for CID %s: %v", subjectCID, err)
	}

	return rows, nil
}
