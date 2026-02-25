// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
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

const verificationStatusVerified = "verified"

type signCtrl struct {
	signv1.UnimplementedSignServiceServer
	db types.SignatureVerificationDatabaseAPI
}

// NewSignController creates a new sign service controller.
// Sign is client-side only; Verify reads cached results written by the reconciler.
func NewSignController(db types.SignatureVerificationDatabaseAPI) signv1.SignServiceServer {
	return &signCtrl{db: db}
}

func (s *signCtrl) Sign(_ context.Context, _ *signv1.SignRequest) (*signv1.SignResponse, error) {
	signLogger.Debug("Sign request received - redirecting to client-side")

	// Sign functionality is handled client-side
	return nil, status.Error(codes.Unimplemented, "server-side signing not implemented - use client SDK")
}

func (s *signCtrl) Verify(ctx context.Context, req *signv1.VerifyRequest) (*signv1.VerifyResponse, error) {
	if req.GetRecordRef() == nil || req.GetRecordRef().GetCid() == "" {
		return nil, status.Error(codes.InvalidArgument, "record CID is required")
	}

	recordCID := req.GetRecordRef().GetCid()

	rows, err := s.db.GetSignatureVerificationsByRecordCID(recordCID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get signature verifications: %v", err)
	}

	var signers []*signv1.SignerInfo

	for _, obj := range rows {
		if obj.GetStatus() != verificationStatusVerified {
			continue
		}

		signers = append(signers, signerInfoFromVerification(obj))
	}

	if len(signers) == 0 {
		errMsg := "no verified signatures found"

		return &signv1.VerifyResponse{
			Success:      false,
			ErrorMessage: &errMsg,
		}, nil
	}

	return &signv1.VerifyResponse{
		Success: true,
		Signers: signers,
	}, nil
}

func signerInfoFromVerification(obj types.SignatureVerificationObject) *signv1.SignerInfo {
	switch obj.GetSignerType() {
	case "oidc":
		return &signv1.SignerInfo{
			Type: &signv1.SignerInfo_Oidc{
				Oidc: &signv1.SignerInfoOIDC{
					Issuer:  obj.GetSignerIssuer(),
					Subject: obj.GetSignerSubject(),
				},
			},
		}
	case "key":
		return &signv1.SignerInfo{
			Type: &signv1.SignerInfo_Key{
				Key: &signv1.SignerInfoKey{
					PublicKey: obj.GetSignerPublicKey(),
					Algorithm: obj.GetSignerAlgorithm(),
				},
			},
		}
	default:
		return &signv1.SignerInfo{}
	}
}
