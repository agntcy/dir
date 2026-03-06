// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package signature implements the signature verification reconciler task.
// It verifies signed records via utils/verify and the store as Fetcher, then caches results in the database.
package signature

import (
	"context"
	"fmt"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
	gormdb "github.com/agntcy/dir/server/database/gorm"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	"github.com/agntcy/dir/utils/verify"
)

var logger = logging.Logger("reconciler/signature")

// Task implements the signature verification reconciler task.
type Task struct {
	config  Config
	db      types.DatabaseAPI
	fetcher verify.Fetcher
}

// NewTask creates a new signature verification task.
// fetcher supplies signatures and public keys (e.g. storeFetcher from the OCI store).
func NewTask(config Config, db types.DatabaseAPI, fetcher verify.Fetcher) (*Task, error) {
	return &Task{
		config:  config,
		db:      db,
		fetcher: fetcher,
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

// Run fetches records needing signature verification and verifies each via verify.Verify, then updates the DB.
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

		err := t.verifyRecord(recordCtx, cid)
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

// verifyRecord runs verify.Verify for one record using the task's fetcher and updates signature_verifications and record.trusted.
func (t *Task) verifyRecord(ctx context.Context, recordCID string) error {
	if t.fetcher == nil {
		return fmt.Errorf("verify fetcher not configured")
	}

	req := &signv1.VerifyRequest{
		RecordRef: &corev1.RecordRef{Cid: recordCID},
		Provider: &signv1.VerifyRequestProvider{
			Request: &signv1.VerifyRequestProvider_Any{
				Any: &signv1.VerifyWithAny{
					OidcOptions: signv1.DefaultVerifyOptionsOIDC.GetDefaultOptions(),
				},
			},
		},
	}

	resp, perSig, err := verify.Verify(ctx, req, t.fetcher)
	if err != nil {
		return fmt.Errorf("verify: %w", err)
	}

	now := time.Now()

	for _, p := range perSig {
		var signerType, issuer, subject, pubKey, algorithm string

		if p.SignerInfo != nil {
			switch s := p.SignerInfo.GetType().(type) {
			case *signv1.SignerInfo_Oidc:
				if s.Oidc != nil {
					signerType = "oidc"
					issuer = s.Oidc.GetIssuer()
					subject = s.Oidc.GetSubject()
				}
			case *signv1.SignerInfo_Key:
				if s.Key != nil {
					signerType = "key"
					pubKey = s.Key.GetPublicKey()
					algorithm = s.Key.GetAlgorithm()
				}
			}
		}

		sv := &gormdb.SignatureVerification{
			RecordCID:       recordCID,
			SignatureDigest: p.Digest,
			Status:          p.Status,
			ErrorMessage:    p.ErrorMessage,
			SignerType:      signerType,
			SignerIssuer:    issuer,
			SignerSubject:   subject,
			SignerPublicKey: pubKey,
			SignerAlgorithm: algorithm,
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		if err := t.db.UpsertSignatureVerification(sv); err != nil {
			logger.Warn("Failed to upsert signature verification", "record_cid", recordCID, "digest", p.Digest, "error", err)
		}
	}

	logger.Debug("Signature verification complete", "record_cid", recordCID, "success", resp.GetSuccess())

	return nil
}

// storeFetcher implements verify.Fetcher using a ReferrerStoreAPI (e.g. OCI store).
type storeFetcher struct {
	store types.ReferrerStoreAPI
}

// NewStoreFetcher returns a Fetcher that reads signatures and public keys from the store.
func NewStoreFetcher(store types.ReferrerStoreAPI) verify.Fetcher {
	return &storeFetcher{store: store}
}

// PullSignatures implements verify.Fetcher.
func (s *storeFetcher) PullSignatures(ctx context.Context, recordRef *corev1.RecordRef) ([]verify.SigWithDigest, error) {
	recordCID := recordRef.GetCid()

	var out []verify.SigWithDigest

	err := s.store.WalkReferrers(ctx, recordCID, corev1.SignatureReferrerType, func(ref *corev1.RecordReferrer) error {
		sig := &signv1.Signature{}
		if err := sig.UnmarshalReferrer(ref); err != nil {
			return fmt.Errorf("unmarshal signature referrer: %w", err)
		}

		out = append(out, verify.SigWithDigest{
			Digest: verify.ReferrerDigest(ref),
			Sig:    sig,
		})

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk signature referrers: %w", err)
	}

	return out, nil
}

// PullPublicKeys implements verify.Fetcher.
func (s *storeFetcher) PullPublicKeys(ctx context.Context, recordRef *corev1.RecordRef) ([]string, error) {
	recordCID := recordRef.GetCid()

	var out []string

	err := s.store.WalkReferrers(ctx, recordCID, corev1.PublicKeyReferrerType, func(ref *corev1.RecordReferrer) error {
		pk := &signv1.PublicKey{}
		if err := pk.UnmarshalReferrer(ref); err != nil {
			// Skip invalid referrer; continue walk.
			//nolint:nilerr
			return nil
		}

		if k := pk.GetKey(); k != "" {
			out = append(out, k)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk public key referrers: %w", err)
	}

	return out, nil
}
