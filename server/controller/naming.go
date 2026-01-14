// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package controller

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	namingv1 "github.com/agntcy/dir/api/naming/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/server/types/adapters"
	"github.com/agntcy/dir/server/verification"
	"github.com/agntcy/dir/utils/logging"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var namingLogger = logging.Logger("controller/naming")

// Sentinel error for stopping referrer walk.
var errStopWalk = errors.New("stop walking")

type namingCtrl struct {
	namingv1.UnimplementedNamingServiceServer
	store    types.StoreAPI
	verifier *verification.Verifier
}

// NewNamingController creates a new naming service controller.
func NewNamingController(store types.StoreAPI, verifier *verification.Verifier) namingv1.NamingServiceServer {
	return &namingCtrl{
		store:    store,
		verifier: verifier,
	}
}

// Verify performs name verification for a signed record.
func (n *namingCtrl) Verify(ctx context.Context, req *namingv1.VerifyRequest) (*namingv1.VerifyResponse, error) {
	namingLogger.Debug("Verify request received", "cid", req.GetCid())

	if req.GetCid() == "" {
		return nil, status.Error(codes.InvalidArgument, "cid is required")
	}

	// Check if already verified - prevent duplicates
	existingResp, err := n.GetVerificationInfo(ctx, &namingv1.GetVerificationInfoRequest{Cid: req.GetCid()})
	if err == nil && existingResp.GetVerified() {
		namingLogger.Debug("Record already has name verification", "cid", req.GetCid())

		return &namingv1.VerifyResponse{
			Verified:     true,
			Verification: existingResp.GetVerification(),
		}, nil
	}

	// Get the record to extract the name
	record, err := n.store.Pull(ctx, &corev1.RecordRef{Cid: req.GetCid()})
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "failed to get record: %v", err)
	}

	// Extract the name from the record using adapter
	recordName, err := extractRecordName(record)
	if err != nil {
		errMsg := fmt.Sprintf("failed to extract record name: %v", err)

		return &namingv1.VerifyResponse{
			Verified:     false,
			ErrorMessage: &errMsg,
		}, nil
	}

	if recordName == "" {
		errMsg := "record has no name field"

		return &namingv1.VerifyResponse{
			Verified:     false,
			ErrorMessage: &errMsg,
		}, nil
	}

	// Get the public key for this record
	publicKey, err := n.getRecordPublicKey(ctx, req.GetCid())
	if err != nil {
		errMsg := fmt.Sprintf("failed to get public key: %v", err)

		return &namingv1.VerifyResponse{
			Verified:     false,
			ErrorMessage: &errMsg,
		}, nil
	}

	// Perform verification
	result := n.verifier.Verify(ctx, recordName, publicKey)

	if !result.Verified {
		errMsg := result.Error
		if errMsg == "" {
			errMsg = "verification failed"
		}

		return &namingv1.VerifyResponse{
			Verified:     false,
			ErrorMessage: &errMsg,
		}, nil
	}

	// Create verification object with domain verification info
	verification := namingv1.NewDomainVerification(&namingv1.DomainVerification{
		Domain:       result.Domain,
		Method:       result.Method,
		MatchedKeyId: result.MatchedKeyID,
		VerifiedAt:   timestamppb.New(result.VerifiedAt),
	})

	// Store the verification as a referrer
	referrer, err := verification.MarshalReferrer()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal verification: %v", err)
	}

	referrerStore, ok := n.store.(types.ReferrerStoreAPI)
	if !ok {
		return nil, status.Error(codes.Internal, "store does not support referrers")
	}

	if err := referrerStore.PushReferrer(ctx, req.GetCid(), referrer); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to store verification: %v", err)
	}

	namingLogger.Info("Name verification stored",
		"cid", req.GetCid(),
		"domain", result.Domain,
		"method", result.Method,
		"keyID", result.MatchedKeyID)

	return &namingv1.VerifyResponse{
		Verified:     true,
		Verification: verification,
	}, nil
}

// GetVerificationInfo retrieves the verification info for a record.
func (n *namingCtrl) GetVerificationInfo(ctx context.Context, req *namingv1.GetVerificationInfoRequest) (*namingv1.GetVerificationInfoResponse, error) {
	namingLogger.Debug("GetVerificationInfo request received", "cid", req.GetCid())

	if req.GetCid() == "" {
		return nil, status.Error(codes.InvalidArgument, "cid is required")
	}

	referrerStore, ok := n.store.(types.ReferrerStoreAPI)
	if !ok {
		return nil, status.Error(codes.Internal, "store does not support referrers")
	}

	var verification *namingv1.Verification

	// Walk verification referrers to find the verification
	err := referrerStore.WalkReferrers(ctx, req.GetCid(), corev1.VerificationReferrerType, func(referrer *corev1.RecordReferrer) error {
		v := &namingv1.Verification{}
		if err := v.UnmarshalReferrer(referrer); err != nil {
			namingLogger.Debug("Failed to unmarshal verification referrer", "error", err)

			return nil // Continue walking
		}

		verification = v

		return errStopWalk
	})

	if err != nil && !errors.Is(err, errStopWalk) {
		return nil, status.Errorf(codes.Internal, "failed to walk referrers: %v", err)
	}

	if verification == nil {
		errMsg := "no verification found"

		return &namingv1.GetVerificationInfoResponse{
			Verified:     false,
			ErrorMessage: &errMsg,
		}, nil
	}

	return &namingv1.GetVerificationInfoResponse{
		Verified:     true,
		Verification: verification,
	}, nil
}

// getRecordPublicKey retrieves the public key associated with a record.
func (n *namingCtrl) getRecordPublicKey(ctx context.Context, cid string) ([]byte, error) {
	referrerStore, ok := n.store.(types.ReferrerStoreAPI)
	if !ok {
		return nil, errors.New("store does not support referrers")
	}

	var publicKey []byte

	// Walk public key referrers to find the key
	err := referrerStore.WalkReferrers(ctx, cid, corev1.PublicKeyReferrerType, func(referrer *corev1.RecordReferrer) error {
		pk := &signv1.PublicKey{}
		if err := pk.UnmarshalReferrer(referrer); err != nil {
			namingLogger.Debug("Failed to unmarshal public key referrer", "error", err)

			return nil // Continue walking
		}

		// The public key is stored as PEM-encoded string
		pemKey := pk.GetKey()
		if pemKey == "" {
			namingLogger.Debug("Empty public key")

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

			namingLogger.Debug("Failed to parse public key", "error", err)

			return nil // Continue walking
		}

		// Marshal the key to DER format for comparison
		keyBytes, err := cryptoutils.MarshalPublicKeyToDER(parsedKey)
		if err != nil {
			namingLogger.Debug("Failed to marshal public key to DER", "error", err)

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

// extractRecordName extracts the name from a record using the adapter pattern.
func extractRecordName(record *corev1.Record) (string, error) {
	adapter := adapters.NewRecordAdapter(record)

	recordData, err := adapter.GetRecordData()
	if err != nil {
		return "", fmt.Errorf("failed to get record data: %w", err)
	}

	if recordData == nil {
		return "", nil
	}

	return recordData.GetName(), nil
}
