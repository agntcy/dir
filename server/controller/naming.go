// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package controller

import (
	"context"
	"errors"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	namingv1 "github.com/agntcy/dir/api/naming/v1"
	"github.com/agntcy/dir/server/database/sqlite"
	"github.com/agntcy/dir/server/naming"
	reverificationconfig "github.com/agntcy/dir/server/reverification/config"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/server/types/adapters"
	"github.com/agntcy/dir/utils/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var namingLogger = logging.Logger("controller/naming")

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

// isVerificationValid checks if a verification is valid (verified status and not expired).
func (n *namingCtrl) isVerificationValid(v types.NameVerificationObject) bool {
	if v.GetStatus() != sqlite.VerificationStatusVerified {
		return false
	}

	// Check if verification has expired: updated_at + ttl < now
	return time.Now().Before(v.GetUpdatedAt().Add(n.ttl))
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
