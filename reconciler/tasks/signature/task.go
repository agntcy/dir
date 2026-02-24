// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package signature implements the signature verification reconciler task.
// It periodically verifies signed records and caches results in the database.
package signature

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
	"github.com/agntcy/dir/client/utils/cosign"
	gormdb "github.com/agntcy/dir/server/database/gorm"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	"google.golang.org/protobuf/proto"
)

var logger = logging.Logger("reconciler/signature")

// Task implements the signature verification reconciler task.
type Task struct {
	config Config
	db     types.DatabaseAPI
	store  types.ReferrerStoreAPI
}

// NewTask creates a new signature verification task.
// store must implement types.ReferrerStoreAPI (e.g. OCI store).
func NewTask(config Config, db types.DatabaseAPI, store types.ReferrerStoreAPI) (*Task, error) {
	return &Task{
		config: config,
		db:     db,
		store:  store,
	}, nil
}

// Name returns the task name.
func (t *Task) Name() string {
	return "signature"
}

// Interval returns how often this task should run.
func (t *Task) Interval() time.Duration {
	return t.config.GetInterval()
}

// IsEnabled returns whether this task is enabled.
func (t *Task) IsEnabled() bool {
	return t.config.Enabled
}

// Run fetches records needing signature verification and verifies each.
func (t *Task) Run(ctx context.Context) error {
	logger.Debug("Running signature verification")

	records, err := t.db.GetRecordsNeedingSignatureVerification(t.config.GetTTL())
	if err != nil {
		return fmt.Errorf("get records needing signature verification: %w", err)
	}

	if len(records) == 0 {
		logger.Info("No records need signature verification")

		return nil
	}

	logger.Info("Processing records for signature verification", "count", len(records))

	var succeeded, failed int

	for _, r := range records {
		recordCtx, cancel := context.WithTimeout(ctx, t.config.GetRecordTimeout())
		defer cancel()

		cid := r.GetCid()

		err := t.verifySignatures(recordCtx, cid)
		if err != nil {
			logger.Warn("Signature verification failed for record", "cid", cid, "error", err)

			failed++
		} else {
			succeeded++
		}
	}

	logger.Info("Signature verification complete", "succeeded", succeeded, "failed", failed)

	return nil
}

// verifyRecordSignatures verifies all signatures for one record and persists results plus record.trusted.
func (t *Task) verifySignatures(ctx context.Context, recordCID string) error {
	signatures, err := t.collectSignatures(ctx, recordCID)
	if err != nil {
		return fmt.Errorf("collect signatures: %w", err)
	}

	if len(signatures) == 0 {
		logger.Debug("No signatures to verify", "record_cid", recordCID)

		return nil
	}

	publicKeys, err := t.collectPublicKeys(ctx, recordCID)
	if err != nil {
		return fmt.Errorf("collect public keys: %w", err)
	}

	var anyVerified bool

	for _, item := range signatures {
		sv := t.verifySignature(ctx, recordCID, item, publicKeys)
		if err := t.db.UpsertSignatureVerification(sv); err != nil {
			logger.Warn("Failed to upsert signature verification", "record_cid", recordCID, "digest", item.digest, "error", err)

			continue
		}

		if sv.GetStatus() == gormdb.VerificationStatusVerified {
			anyVerified = true
		}
	}

	if err := t.db.SetRecordTrusted(recordCID, anyVerified); err != nil {
		return fmt.Errorf("set record trusted: %w", err)
	}

	logger.Debug("Signature verification complete", "record_cid", recordCID, "trusted", anyVerified)

	return nil
}

type sigWithDigest struct {
	digest string
	sig    *signv1.Signature
}

func (t *Task) collectSignatures(ctx context.Context, recordCID string) ([]sigWithDigest, error) {
	var out []sigWithDigest

	err := t.store.WalkReferrers(ctx, recordCID, corev1.SignatureReferrerType, func(ref *corev1.RecordReferrer) error {
		sig := &signv1.Signature{}
		if err := sig.UnmarshalReferrer(ref); err != nil {
			logger.Debug("Failed to unmarshal signature referrer", "error", err)

			return fmt.Errorf("failed to unmarshal signature referrer: %w", err)
		}

		digest := digestReferrer(ref)
		out = append(out, sigWithDigest{digest: digest, sig: sig})

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk referrers: %w", err)
	}

	return out, nil
}

func (t *Task) collectPublicKeys(ctx context.Context, recordCID string) ([]string, error) {
	var out []string

	err := t.store.WalkReferrers(ctx, recordCID, corev1.PublicKeyReferrerType, func(ref *corev1.RecordReferrer) error {
		pk := &signv1.PublicKey{}
		if err := pk.UnmarshalReferrer(ref); err != nil {
			logger.Debug("Failed to unmarshal public key referrer", "error", err)

			return nil
		}

		if k := pk.GetKey(); k != "" {
			out = append(out, k)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk referrers: %w", err)
	}

	return out, nil
}

func digestReferrer(ref *corev1.RecordReferrer) string {
	data, _ := proto.Marshal(ref)
	h := sha256.Sum256(data)

	return hex.EncodeToString(h[:])
}

// verifySignature runs the same verification logic as client.Verify for provider Any.
func (t *Task) verifySignature(ctx context.Context, recordCID string, item sigWithDigest, publicKeys []string) *gormdb.SignatureVerification {
	now := time.Now()
	sv := &gormdb.SignatureVerification{
		RecordCID:       recordCID,
		SignatureDigest: item.digest,
		Status:          gormdb.VerificationStatusFailed,
		ErrorMessage:    "",
		SignerType:      "",
		SignerIssuer:    "",
		SignerSubject:   "",
		SignerPublicKey: "",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	var (
		info *signv1.SignerInfo
		err  error
	)

	payload := []byte(recordCID)

	if len(item.sig.GetContentBundle()) == 0 {
		info, err = cosign.VerifyWithKeys(
			ctx,
			payload,
			publicKeys,
			item.sig,
		)
	} else {
		info, err = cosign.VerifyWithOIDC(
			payload,
			&signv1.VerifyWithOIDC{
				Options: signv1.DefaultVerifyOptionsOIDC.GetDefaultOptions(),
			},
			item.sig,
		)
	}

	if err != nil {
		sv.ErrorMessage = err.Error()

		return sv
	}

	sv.Status = gormdb.VerificationStatusVerified
	if info.GetOidc() != nil {
		sv.SignerType = "oidc"
		sv.SignerIssuer = info.GetOidc().GetIssuer()
		sv.SignerSubject = info.GetOidc().GetSubject()
	} else if info.GetKey() != nil {
		sv.SignerType = "key"
		sv.SignerPublicKey = info.GetKey().GetPublicKey()
	}

	return sv
}
