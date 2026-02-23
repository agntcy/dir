// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package name implements the name reconciler task: name/DNS verification.
// It periodically re-verifies DNS ownership of named records and stores results in the database.
// Distinct from the signature task (see reconciler/tasks/signature for that).
package name

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
	gormdb "github.com/agntcy/dir/server/database/gorm"
	namingprovider "github.com/agntcy/dir/server/naming"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
)

var logger = logging.Logger("reconciler/name")

// Task implements the name reconciler task (name/DNS ownership verification).
type Task struct {
	config   Config
	db       types.DatabaseAPI
	store    types.StoreAPI
	provider *namingprovider.Provider
}

// NewTask creates a new name reconciliation task.
func NewTask(config Config, db types.DatabaseAPI, store types.StoreAPI, provider *namingprovider.Provider) (*Task, error) {
	return &Task{
		config:   config,
		db:       db,
		store:    store,
		provider: provider,
	}, nil
}

// Name returns the task name: name/DNS verification (distinct from signature task).
func (t *Task) Name() string {
	return "name"
}

// Interval returns how often this task should run.
func (t *Task) Interval() time.Duration {
	return t.config.GetInterval()
}

// IsEnabled returns whether this task is enabled.
func (t *Task) IsEnabled() bool {
	return t.config.Enabled
}

// Run executes name verification: fetch records needing verification, then verify each.
func (t *Task) Run(ctx context.Context) error {
	logger.Debug("Running name verification")

	records, err := t.db.GetRecordsNeedingVerification(t.config.GetTTL())
	if err != nil {
		return fmt.Errorf("get records needing name verification: %w", err)
	}

	if len(records) == 0 {
		logger.Info("No records need name verification")

		return nil
	}

	logger.Info("Processing records for name verification", "count", len(records))

	var succeeded, failed int

	for _, r := range records {
		recordData, err := r.GetRecordData()
		if err != nil {
			logger.Warn("Failed to get record data", "cid", r.GetCid(), "error", err)

			failed++

			continue
		}

		recordCtx, cancel := context.WithTimeout(ctx, t.config.GetRecordTimeout())
		verified := t.verifyNameOwnership(recordCtx, r.GetCid(), recordData.GetName())

		cancel()

		if verified {
			succeeded++
		} else {
			failed++
		}
	}

	logger.Info("Name verification complete", "succeeded", succeeded, "failed", failed)

	return nil
}

// verifyNameOwnership performs DNS/name ownership verification for one record and stores the result.
// Returns true if verification succeeded. Distinct from the signature task.
func (t *Task) verifyNameOwnership(ctx context.Context, cid, recordName string) bool {
	publicKeys, err := t.getRecordPublicKeysForNameVerification(ctx, cid)
	if err != nil {
		t.storeNameVerificationResult(cid, "", "", fmt.Sprintf("failed to get public keys: %v", err))

		return false
	}

	if len(publicKeys) == 0 {
		t.storeNameVerificationResult(cid, "", "", "no public keys found for record")

		return false
	}

	var lastResult *namingprovider.Result

	for _, publicKey := range publicKeys {
		result := t.provider.Verify(ctx, recordName, publicKey)
		lastResult = result

		if result.Verified {
			t.storeNameVerificationResult(cid, result.Method, result.MatchedKeyID, "")

			logger.Info("Name verification successful", "cid", cid, "domain", result.Domain, "method", result.Method)

			return true
		}
	}

	errMsg := "verification failed"
	if lastResult != nil && lastResult.Error != "" {
		errMsg = lastResult.Error
	}

	t.storeNameVerificationResult(cid, lastResult.Method, "", errMsg)

	return false
}

// storeNameVerificationResult persists a name verification result in the database.
func (t *Task) storeNameVerificationResult(cid, method, keyID, errMsg string) {
	status := gormdb.VerificationStatusVerified
	if errMsg != "" {
		status = gormdb.VerificationStatusFailed
	}

	nv := &gormdb.NameVerification{
		RecordCID: cid,
		Method:    method,
		KeyID:     keyID,
		Status:    status,
		Error:     errMsg,
	}

	_, err := t.db.GetVerificationByCID(cid)

	switch {
	case errors.Is(err, gormdb.ErrVerificationNotFound):
		if err := t.db.CreateNameVerification(nv); err != nil {
			logger.Warn("Failed to create name verification", "error", err, "cid", cid)
		}
	case err != nil:
		logger.Warn("Failed to check existing name verification", "error", err, "cid", cid)
	default:
		if err := t.db.UpdateNameVerification(nv); err != nil {
			logger.Warn("Failed to update name verification", "error", err, "cid", cid)
		}
	}
}

// getRecordPublicKeysForNameVerification returns public keys attached to the record for name verification.
func (t *Task) getRecordPublicKeysForNameVerification(ctx context.Context, cid string) ([][]byte, error) {
	referrerStore, ok := t.store.(types.ReferrerStoreAPI)
	if !ok {
		return nil, errors.New("store does not support referrers")
	}

	var publicKeys [][]byte

	err := referrerStore.WalkReferrers(ctx, cid, corev1.PublicKeyReferrerType, func(referrer *corev1.RecordReferrer) error {
		pk := &signv1.PublicKey{}
		if err := pk.UnmarshalReferrer(referrer); err != nil {
			logger.Debug("Failed to unmarshal public key referrer", "error", err)

			return nil
		}

		pemKey := pk.GetKey()
		if pemKey == "" {
			logger.Debug("Empty public key")

			return nil
		}

		parsedKey, err := cryptoutils.UnmarshalPEMToPublicKey([]byte(pemKey))
		if err != nil {
			keyBytes, decodeErr := base64.StdEncoding.DecodeString(pemKey)
			if decodeErr == nil {
				publicKeys = append(publicKeys, keyBytes)

				return nil
			}

			logger.Debug("Failed to parse public key", "error", err)

			return nil
		}

		keyBytes, err := cryptoutils.MarshalPublicKeyToDER(parsedKey)
		if err != nil {
			logger.Debug("Failed to marshal public key to DER", "error", err)

			return nil
		}

		publicKeys = append(publicKeys, keyBytes)

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk referrers: %w", err)
	}

	return publicKeys, nil
}
