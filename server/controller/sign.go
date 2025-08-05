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
	store types.StoreAPI
}

// NewSignController creates a new sign service controller.
func NewSignController(store types.StoreAPI) signv1.SignServiceServer {
	return &signCtrl{
		store: store,
	}
}

func (s *signCtrl) Sign(_ context.Context, _ *signv1.SignRequest) (*signv1.SignResponse, error) {
	signLogger.Debug("Sign request received")

	// Sign functionality is handled client-side
	return nil, status.Error(codes.Unimplemented, "server-side signing not implemented") //nolint:wrapcheck
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
