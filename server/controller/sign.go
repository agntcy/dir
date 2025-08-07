// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"

	signv1 "github.com/agntcy/dir/api/sign/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var signLogger = logging.Logger("controller/sign")

type signCtrl struct {
	signv1.UnimplementedSignServiceServer
	store types.SignatureStoreAPI
}

// NewSignController creates a new sign service controller.
func NewSignController(store types.StoreAPI) signv1.SignServiceServer {
	// Check if store supports signature operations
	var sigStore types.SignatureStoreAPI
	if s, ok := store.(types.SignatureStoreAPI); ok {
		sigStore = s
	}

	return &signCtrl{
		store: sigStore,
	}
}

func (s *signCtrl) Sign(ctx context.Context, req *signv1.SignRequest) (*signv1.SignResponse, error) {
	signLogger.Debug("Sign request received", "recordCID", req.GetRecordRef().GetCid())

	// Validate request
	if req.GetRecordRef() == nil || req.GetRecordRef().GetCid() == "" {
		return nil, status.Error(codes.InvalidArgument, "record ref must be set")
	}

	// For now, the Sign service doesn't handle server-side signing
	// Client-side signing is handled through SignWithKey/SignWithOIDC
	// This endpoint is reserved for future server-side signing implementation

	signLogger.Debug("Sign endpoint called - currently not implemented for server-side signing")

	return &signv1.SignResponse{
		Signature: nil,
	}, status.Error(codes.Unimplemented, "server-side signing not yet implemented")
}

func (s *signCtrl) PushSignature(ctx context.Context, req *signv1.PushSignatureRequest) (*signv1.PushSignatureResponse, error) {
	signLogger.Debug("PushSignature request received", "recordCID", req.GetRecordRef().GetCid())

	// Validate request
	if req.GetRecordRef() == nil || req.GetRecordRef().GetCid() == "" {
		return &signv1.PushSignatureResponse{
			Success:      false,
			ErrorMessage: stringPtr("record ref must be set"),
		}, nil
	}

	if req.GetSignature() == nil {
		return &signv1.PushSignatureResponse{
			Success:      false,
			ErrorMessage: stringPtr("signature must be set"),
		}, nil
	}

	// Check if store supports signature operations
	if s.store == nil {
		return &signv1.PushSignatureResponse{
			Success:      false,
			ErrorMessage: stringPtr("signature store not available"),
		}, nil
	}

	// Push signature to store
	err := s.store.PushSignature(ctx, req.GetRecordRef().GetCid(), req.GetSignature())
	if err != nil {
		signLogger.Error("Failed to push signature", "error", err, "recordCID", req.GetRecordRef().GetCid())
		return &signv1.PushSignatureResponse{
			Success:      false,
			ErrorMessage: stringPtr(err.Error()),
		}, nil
	}

	signLogger.Debug("Signature stored successfully", "recordCID", req.GetRecordRef().GetCid())

	return &signv1.PushSignatureResponse{
		Success: true,
	}, nil
}

func (s *signCtrl) PullSignature(ctx context.Context, req *signv1.PullSignatureRequest) (*signv1.PullSignatureResponse, error) {
	signLogger.Debug("PullSignature request received", "recordCID", req.GetRecordRef().GetCid())

	// Validate request
	if req.GetRecordRef() == nil || req.GetRecordRef().GetCid() == "" {
		return &signv1.PullSignatureResponse{
			Found: false,
		}, status.Error(codes.InvalidArgument, "record ref must be set")
	}

	// Check if store supports signature operations
	if s.store == nil {
		return &signv1.PullSignatureResponse{
			Found: false,
		}, status.Error(codes.Internal, "signature store not available")
	}

	// Pull signature from store
	signature, err := s.store.PullSignature(ctx, req.GetRecordRef().GetCid())
	if err != nil {
		signLogger.Debug("No signature found", "recordCID", req.GetRecordRef().GetCid(), "error", err)
		return &signv1.PullSignatureResponse{
			Found: false,
		}, nil
	}

	signLogger.Debug("Signature retrieved successfully", "recordCID", req.GetRecordRef().GetCid())

	return &signv1.PullSignatureResponse{
		Signature: signature,
		Found:     true,
	}, nil
}

func (s *signCtrl) Verify(ctx context.Context, req *signv1.VerifyRequest) (*signv1.VerifyResponse, error) {
	signLogger.Debug("Verify request received")

	// Validate request
	if req.GetRecordRef() == nil || req.GetRecordRef().GetCid() == "" {
		return nil, status.Error(codes.InvalidArgument, "record ref must be set") //nolint:wrapcheck
	}

	// Server-side verification is enabled by zot verification.
	return s.verifyWithZot(ctx, req.GetRecordRef().GetCid())
}

// verifyWithZot attempts zot verification if the store supports it.
func (s *signCtrl) verifyWithZot(ctx context.Context, recordCID string) (*signv1.VerifyResponse, error) {
	// Try zot verification if the store supports it
	if zotStore, ok := s.store.(interface {
		VerifyWithZot(ctx context.Context, recordCID string) (bool, error)
	}); ok {
		signLogger.Debug("Attempting zot verification", "recordCID", recordCID)

		verified, err := zotStore.VerifyWithZot(ctx, recordCID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "zot verification failed: %v", err)
		}

		signLogger.Debug("Zot verification completed", "recordCID", recordCID, "verified", verified)

		var errMsg string
		if !verified {
			errMsg = "Signature verification failed"
		}

		return &signv1.VerifyResponse{
			Success:      verified,
			ErrorMessage: &errMsg,
		}, nil
	}

	return nil, status.Error(codes.Unimplemented, "zot verification not available in this store configuration") //nolint:wrapcheck
}

// stringPtr returns a pointer to the given string
func stringPtr(s string) *string {
	return &s
}
