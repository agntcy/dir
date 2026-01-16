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
	signing types.SigningAPI
}

// NewSignController creates a new sign service controller.
func NewSignController(signing types.SigningAPI) signv1.SignServiceServer {
	return &signCtrl{
		signing: signing,
	}
}

//nolint:wrapcheck
func (s *signCtrl) Sign(_ context.Context, _ *signv1.SignRequest) (*signv1.SignResponse, error) {
	signLogger.Debug("Sign request received")

	// Sign functionality is handled client-side
	return nil, status.Error(codes.Unimplemented, "server-side signing not implemented")
}

func (s *signCtrl) Verify(ctx context.Context, req *signv1.VerifyRequest) (*signv1.VerifyResponse, error) {
	signLogger.Debug("Verify request received")

	// Validate request
	if req.GetRecordRef() == nil || req.GetRecordRef().GetCid() == "" {
		return nil, status.Error(codes.InvalidArgument, "record ref must be set") //nolint:wrapcheck
	}

	recordCID := req.GetRecordRef().GetCid()

	signLogger.Debug("Attempting signature verification", "recordCID", recordCID)

	verified, err := s.signing.Verify(ctx, recordCID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "signature verification failed: %v", err)
	}

	signLogger.Debug("Signature verification completed", "recordCID", recordCID, "verified", verified)

	var errMsg string
	if !verified {
		errMsg = "Signature verification failed"
	}

	return &signv1.VerifyResponse{
		Success:      verified,
		ErrorMessage: &errMsg,
	}, nil
}
