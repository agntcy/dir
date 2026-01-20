// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package controller

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	namingv1 "github.com/agntcy/dir/api/naming/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
	"github.com/agntcy/dir/server/database/sqlite"
	"github.com/agntcy/dir/server/naming"
	reverificationconfig "github.com/agntcy/dir/server/reverification/config"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/server/types/adapters"
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
	db       types.DatabaseAPI
	provider *naming.Provider
	ttl      time.Duration
}

// NamingControllerOption configures a naming controller.
type NamingControllerOption func(*namingCtrl)

// WithVerificationTTL sets the verification TTL.
func WithVerificationTTL(ttl time.Duration) NamingControllerOption {
	return func(n *namingCtrl) {
		n.ttl = ttl
	}
}

// NewNamingController creates a new naming service controller.
func NewNamingController(store types.StoreAPI, db types.DatabaseAPI, provider *naming.Provider, opts ...NamingControllerOption) namingv1.NamingServiceServer {
	ctrl := &namingCtrl{
		store:    store,
		db:       db,
		provider: provider,
		ttl:      reverificationconfig.DefaultTTL,
	}

	for _, opt := range opts {
		opt(ctrl)
	}

	return ctrl
}

// Verify performs name verification for a signed record.
func (n *namingCtrl) Verify(ctx context.Context, req *namingv1.VerifyRequest) (*namingv1.VerifyResponse, error) {
	namingLogger.Debug("Verify request received", "cid", req.GetCid())

	if req.GetCid() == "" {
		return nil, status.Error(codes.InvalidArgument, "cid is required")
	}

	// Check database cache first - only use if verified and not expired
	cached, err := n.db.GetVerificationByCID(req.GetCid())
	if err != nil && !errors.Is(err, sqlite.ErrVerificationNotFound) {
		namingLogger.Debug("Error checking verification cache", "error", err)
	} else if err == nil && n.isVerificationValid(cached) {
		namingLogger.Debug("Using cached verification", "cid", req.GetCid())

		return &namingv1.VerifyResponse{
			Verified: true,
			Verification: namingv1.NewDomainVerification(&namingv1.DomainVerification{
				Domain:     n.getDomainFromRecord(ctx, req.GetCid()),
				Method:     cached.GetMethod(),
				KeyId:      cached.GetKeyID(),
				VerifiedAt: timestamppb.New(cached.GetUpdatedAt()),
			}),
		}, nil
	}

	// Get the record to extract the name
	record, err := n.store.Pull(ctx, &corev1.RecordRef{Cid: req.GetCid()})
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "failed to get record: %v", err)
	}

	// Extract the name from the record using adapter
	adapter := adapters.NewRecordAdapter(record)

	recordData, err := adapter.GetRecordData()
	if err != nil {
		errMsg := fmt.Sprintf("failed to get record data: %v", err)
		n.storeFailedVerification(req.GetCid(), "", errMsg)

		return &namingv1.VerifyResponse{
			Verified:     false,
			ErrorMessage: &errMsg,
		}, nil
	}

	recordName := recordData.GetName()
	if recordName == "" {
		errMsg := "record has no name field"
		n.storeFailedVerification(req.GetCid(), "", errMsg)

		return &namingv1.VerifyResponse{
			Verified:     false,
			ErrorMessage: &errMsg,
		}, nil
	}

	// Get the public key for this record
	publicKey, err := n.getRecordPublicKey(ctx, req.GetCid())
	if err != nil {
		errMsg := fmt.Sprintf("failed to get public key: %v", err)
		n.storeFailedVerification(req.GetCid(), "", errMsg)

		return &namingv1.VerifyResponse{
			Verified:     false,
			ErrorMessage: &errMsg,
		}, nil
	}

	// Perform verification
	result := n.provider.Verify(ctx, recordName, publicKey)

	if !result.Verified {
		errMsg := result.Error
		if errMsg == "" {
			errMsg = "verification failed"
		}

		n.storeFailedVerification(req.GetCid(), result.Method, errMsg)

		return &namingv1.VerifyResponse{
			Verified:     false,
			ErrorMessage: &errMsg,
		}, nil
	}

	// Store the verification in database
	verifiedAt := time.Now()

	n.storeSuccessfulVerification(req.GetCid(), result)

	// Create verification object for response
	verification := namingv1.NewDomainVerification(&namingv1.DomainVerification{
		Domain:     result.Domain,
		Method:     result.Method,
		KeyId:      result.MatchedKeyID,
		VerifiedAt: timestamppb.New(verifiedAt),
	})

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

	// Query database for the latest verification attempt
	latest, err := n.db.GetVerificationByCID(req.GetCid())
	if err != nil {
		if errors.Is(err, sqlite.ErrVerificationNotFound) {
			errMsg := "no verification found"

			return &namingv1.GetVerificationInfoResponse{
				Verified:     false,
				ErrorMessage: &errMsg,
			}, nil
		}

		return nil, status.Errorf(codes.Internal, "failed to get verification: %v", err)
	}

	// Check if the latest verification is valid (verified and not expired)
	if !n.isVerificationValid(latest) {
		errMsg := latest.GetError()
		if errMsg == "" {
			errMsg = "verification invalid or expired"
		}

		return &namingv1.GetVerificationInfoResponse{
			Verified:     false,
			ErrorMessage: &errMsg,
		}, nil
	}

	// Return valid verification from database
	namingLogger.Debug("Returning verification from database", "cid", req.GetCid())

	return &namingv1.GetVerificationInfoResponse{
		Verified: true,
		Verification: namingv1.NewDomainVerification(&namingv1.DomainVerification{
			Domain:     n.getDomainFromRecord(ctx, req.GetCid()),
			Method:     latest.GetMethod(),
			KeyId:      latest.GetKeyID(),
			VerifiedAt: timestamppb.New(latest.GetUpdatedAt()),
		}),
	}, nil
}

// storeSuccessfulVerification stores a successful verification in the database.
func (n *namingCtrl) storeSuccessfulVerification(cid string, result *naming.Result) {
	nv := &sqlite.NameVerification{
		RecordCID: cid,
		Method:    result.Method,
		KeyID:     result.MatchedKeyID,
		Status:    sqlite.VerificationStatusVerified,
	}

	// Try to create first, if exists then update
	if err := n.db.CreateNameVerification(nv); err != nil {
		if err := n.db.UpdateNameVerification(nv); err != nil {
			namingLogger.Warn("Failed to store verification in database", "error", err, "cid", cid)
		}
	}
}

// isVerificationValid checks if a verification is valid (verified status and not expired).
func (n *namingCtrl) isVerificationValid(v types.NameVerificationObject) bool {
	if v.GetStatus() != sqlite.VerificationStatusVerified {
		return false
	}

	// Check if verification has expired: updated_at + ttl < now
	return time.Now().Before(v.GetUpdatedAt().Add(n.ttl))
}

// storeFailedVerification stores a failed verification in the database.
func (n *namingCtrl) storeFailedVerification(cid, method, errMsg string) {
	nv := &sqlite.NameVerification{
		RecordCID: cid,
		Method:    method,
		Status:    sqlite.VerificationStatusFailed,
		Error:     errMsg,
	}

	// Try to create first, if exists then update
	if err := n.db.CreateNameVerification(nv); err != nil {
		if err := n.db.UpdateNameVerification(nv); err != nil {
			namingLogger.Warn("Failed to store verification in database", "error", err, "cid", cid)
		}
	}
}

// getDomainFromRecord extracts the domain from a record's name.
func (n *namingCtrl) getDomainFromRecord(ctx context.Context, cid string) string {
	record, err := n.store.Pull(ctx, &corev1.RecordRef{Cid: cid})
	if err != nil {
		return ""
	}

	adapter := adapters.NewRecordAdapter(record)

	recordData, err := adapter.GetRecordData()
	if err != nil {
		return ""
	}

	parsed := naming.ParseName(recordData.GetName())
	if parsed == nil {
		return ""
	}

	return parsed.Domain
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
