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
	"github.com/agntcy/dir/server/types/adapters"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
)

// Sentinel error for stopping referrer walk.
var errStopWalk = errors.New("stop walking")

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

// processWorkItem handles a single re-verification work item.
func (w *Worker) processWorkItem(ctx context.Context, item revtypes.WorkItem) {
	logger.Info("Processing re-verification work item", "worker_id", w.id, "cid", item.RecordCID)

	// Create timeout context for this work item
	workCtx, cancel := context.WithTimeout(ctx, w.timeout)
	defer cancel()

	// Perform re-verification (stores result internally)
	w.reVerify(workCtx, item.RecordCID)
}

// reVerify performs the re-verification for a record.
func (w *Worker) reVerify(ctx context.Context, cid string) {
	// Get the record to extract the name
	record, err := w.store.Pull(ctx, &corev1.RecordRef{Cid: cid})
	if err != nil {
		w.storeFailedVerification(cid, "", fmt.Sprintf("failed to get record: %v", err))

		return
	}

	// Extract the name from the record
	adapter := adapters.NewRecordAdapter(record)

	recordData, err := adapter.GetRecordData()
	if err != nil {
		w.storeFailedVerification(cid, "", fmt.Sprintf("failed to get record data: %v", err))

		return
	}

	recordName := recordData.GetName()
	if recordName == "" {
		w.storeFailedVerification(cid, "", "record has no name field")

		return
	}

	// Get the public key for this record
	publicKey, err := w.getRecordPublicKey(ctx, cid)
	if err != nil {
		w.storeFailedVerification(cid, "", fmt.Sprintf("failed to get public key: %v", err))

		return
	}

	// Perform verification
	result := w.provider.Verify(ctx, recordName, publicKey)

	if !result.Verified {
		errMsg := result.Error
		if errMsg == "" {
			errMsg = "verification failed"
		}

		w.storeFailedVerification(cid, result.Method, errMsg)

		return
	}

	// Store successful verification
	w.storeSuccessfulVerification(cid, result)

	logger.Info("Re-verification successful",
		"worker_id", w.id,
		"cid", cid,
		"domain", result.Domain,
		"method", result.Method)
}

// storeSuccessfulVerification stores a successful verification in the database.
func (w *Worker) storeSuccessfulVerification(cid string, result *naming.Result) {
	nv := &sqlite.NameVerification{
		RecordCID: cid,
		Method:    result.Method,
		KeyID:     result.MatchedKeyID,
		Status:    sqlite.VerificationStatusVerified,
	}

	if err := w.db.UpdateNameVerification(nv); err != nil {
		logger.Warn("Failed to store verification in database", "error", err, "cid", cid)
	}
}

// storeFailedVerification stores a failed verification in the database for audit.
func (w *Worker) storeFailedVerification(cid, method, errMsg string) {
	nv := &sqlite.NameVerification{
		RecordCID: cid,
		Method:    method,
		Status:    sqlite.VerificationStatusFailed,
		Error:     errMsg,
	}

	if err := w.db.UpdateNameVerification(nv); err != nil {
		logger.Warn("Failed to store failed verification in database", "error", err, "cid", cid)
	}
}

// getRecordPublicKey retrieves the public key associated with a record.
func (w *Worker) getRecordPublicKey(ctx context.Context, cid string) ([]byte, error) {
	referrerStore, ok := w.store.(types.ReferrerStoreAPI)
	if !ok {
		return nil, errors.New("store does not support referrers")
	}

	var publicKey []byte

	// Walk public key referrers to find the key
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
				publicKey = keyBytes

				return errStopWalk
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

		publicKey = keyBytes

		return errStopWalk
	})

	if err != nil && !errors.Is(err, errStopWalk) {
		return nil, fmt.Errorf("failed to walk referrers: %w", err)
	}

	if publicKey == nil {
		return nil, errors.New("no public key found for record")
	}

	return publicKey, nil
}
