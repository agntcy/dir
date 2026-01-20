// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package reverification

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
	"github.com/agntcy/dir/server/database/sqlite"
	"github.com/agntcy/dir/server/naming"
	revtypes "github.com/agntcy/dir/server/reverification/types"
	"github.com/agntcy/dir/server/types"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
)

// Worker processes re-verification work items.
type Worker struct {
	id        int
	db        types.DatabaseAPI
	store     types.StoreAPI
	provider  *naming.Provider
	workQueue <-chan revtypes.WorkItem
	timeout   time.Duration
}

// NewWorker creates a new worker instance.
func NewWorker(
	id int,
	db types.DatabaseAPI,
	store types.StoreAPI,
	provider *naming.Provider,
	workQueue <-chan revtypes.WorkItem,
	timeout time.Duration,
) *Worker {
	return &Worker{
		id:        id,
		db:        db,
		store:     store,
		provider:  provider,
		workQueue: workQueue,
		timeout:   timeout,
	}
}

// Run starts the worker loop.
func (w *Worker) Run(ctx context.Context, stopCh <-chan struct{}) {
	logger.Info("Starting re-verification worker", "worker_id", w.id)

	for {
		select {
		case <-ctx.Done():
			logger.Info("Worker stopping due to context cancellation", "worker_id", w.id)

			return
		case <-stopCh:
			logger.Info("Worker stopping due to stop signal", "worker_id", w.id)

			return
		case workItem := <-w.workQueue:
			w.processWorkItem(ctx, workItem)
		}
	}
}

// processWorkItem handles a single verification work item.
func (w *Worker) processWorkItem(ctx context.Context, item revtypes.WorkItem) {
	logger.Info("Processing verification work item", "worker_id", w.id, "cid", item.RecordCID, "name", item.Name)

	// Create timeout context for this work item
	workCtx, cancel := context.WithTimeout(ctx, w.timeout)
	defer cancel()

	// Perform verification (stores result internally)
	w.verify(workCtx, item.RecordCID, item.Name)
}

// verify performs the verification for a record.
// It retrieves all public keys attached to the record and tries verification with each one.
func (w *Worker) verify(ctx context.Context, cid, recordName string) {
	// Get all public keys for this record
	publicKeys, err := w.getRecordPublicKeys(ctx, cid)
	if err != nil {
		w.storeVerification(cid, "", "", fmt.Sprintf("failed to get public keys: %v", err))

		return
	}

	if len(publicKeys) == 0 {
		w.storeVerification(cid, "", "", "no public keys found for record")

		return
	}

	// Try verification with each public key until one succeeds
	var lastResult *naming.Result

	for _, publicKey := range publicKeys {
		result := w.provider.Verify(ctx, recordName, publicKey)
		lastResult = result

		if result.Verified {
			// Store successful verification
			w.storeVerification(cid, result.Method, result.MatchedKeyID, "")

			logger.Info("Verification successful",
				"worker_id", w.id,
				"cid", cid,
				"domain", result.Domain,
				"method", result.Method)

			return
		}
	}

	// All keys failed verification
	errMsg := "verification failed"
	if lastResult != nil && lastResult.Error != "" {
		errMsg = lastResult.Error
	}

	w.storeVerification(cid, lastResult.Method, "", errMsg)
}

// storeVerification stores a verification result in the database.
// If errMsg is empty, the verification is considered successful.
func (w *Worker) storeVerification(cid, method, keyID, errMsg string) {
	verificationStatus := sqlite.VerificationStatusVerified
	if errMsg != "" {
		verificationStatus = sqlite.VerificationStatusFailed
	}

	nv := &sqlite.NameVerification{
		RecordCID: cid,
		Method:    method,
		KeyID:     keyID,
		Status:    verificationStatus,
		Error:     errMsg,
	}

	// Check if verification already exists to avoid UNIQUE constraint error noise
	_, err := w.db.GetVerificationByCID(cid)

	switch {
	case errors.Is(err, sqlite.ErrVerificationNotFound):
		// No existing verification, create new one
		if err := w.db.CreateNameVerification(nv); err != nil {
			logger.Warn("Failed to create verification in database", "error", err, "cid", cid)
		}
	case err != nil:
		// Unexpected error checking for existing verification
		logger.Warn("Failed to check existing verification", "error", err, "cid", cid)
	default:
		// Verification exists, update it
		if err := w.db.UpdateNameVerification(nv); err != nil {
			logger.Warn("Failed to update verification in database", "error", err, "cid", cid)
		}
	}
}

// getRecordPublicKeys retrieves all public keys attached to a record.
func (w *Worker) getRecordPublicKeys(ctx context.Context, cid string) ([][]byte, error) {
	referrerStore, ok := w.store.(types.ReferrerStoreAPI)
	if !ok {
		return nil, errors.New("store does not support referrers")
	}

	var publicKeys [][]byte

	// Walk all public key referrers to get all keys
	err := referrerStore.WalkReferrers(ctx, cid, corev1.PublicKeyReferrerType, func(referrer *corev1.RecordReferrer) error {
		pk := &signv1.PublicKey{}
		if err := pk.UnmarshalReferrer(referrer); err != nil {
			logger.Debug("Failed to unmarshal public key referrer", "error", err)

			return nil // Continue walking
		}

		// The public key is stored as PEM-encoded string
		pemKey := pk.GetKey()
		if pemKey == "" {
			logger.Debug("Empty public key")

			return nil // Continue walking
		}

		// Parse the PEM-encoded public key to get the actual key
		parsedKey, err := cryptoutils.UnmarshalPEMToPublicKey([]byte(pemKey))
		if err != nil {
			// Try base64 decoding if not PEM
			keyBytes, decodeErr := base64.StdEncoding.DecodeString(pemKey)
			if decodeErr == nil {
				publicKeys = append(publicKeys, keyBytes)

				return nil // Continue walking
			}

			logger.Debug("Failed to parse public key", "error", err)

			return nil // Continue walking
		}

		// Marshal the key to DER format for comparison
		keyBytes, err := cryptoutils.MarshalPublicKeyToDER(parsedKey)
		if err != nil {
			logger.Debug("Failed to marshal public key to DER", "error", err)

			return nil // Continue walking
		}

		publicKeys = append(publicKeys, keyBytes)

		return nil // Continue walking
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk referrers: %w", err)
	}

	return publicKeys, nil
}
