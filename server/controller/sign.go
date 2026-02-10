// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"

	signv1 "github.com/agntcy/dir/api/sign/v1"
	"github.com/agntcy/dir/utils/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var signLogger = logging.Logger("controller/sign")

type signCtrl struct {
	signv1.UnimplementedSignServiceServer
}

// NewSignController creates a new sign service controller.
// Note: Both Sign and Verify are now handled client-side.
func NewSignController() signv1.SignServiceServer {
	return &signCtrl{}
}

//nolint:wrapcheck
func (s *signCtrl) Sign(_ context.Context, _ *signv1.SignRequest) (*signv1.SignResponse, error) {
	signLogger.Debug("Sign request received - redirecting to client-side")

	// Sign functionality is handled client-side
	return nil, status.Error(codes.Unimplemented, "server-side signing not implemented - use client SDK")
}

//nolint:wrapcheck
func (s *signCtrl) Verify(_ context.Context, _ *signv1.VerifyRequest) (*signv1.VerifyResponse, error) {
	signLogger.Debug("Verify request received - redirecting to client-side")

	// Verify functionality is now handled client-side
	return nil, status.Error(codes.Unimplemented, "server-side verification not implemented - use client SDK")
}
