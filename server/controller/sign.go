// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package controller

import (
	"context"
	"fmt"
	"regexp"

	signv1 "github.com/agntcy/dir/api/sign/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/cosign"
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

	resp := &signv1.VerifyResponse{
		Success: true,
		Signers: signers,
	}

	// Apply provider filtering (key or OIDC) so the response matches the request.
	if prov := req.GetProvider(); prov != nil {
		resp = filterVerifyResponseByProvider(ctx, prov, resp)
	}

	return resp, nil
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

// filterVerifyResponseByProvider keeps only signers that match the requested provider (key or OIDC).
// Used when returning server-cached verification results so the server applies provider filtering.
//
//nolint:cyclop
func filterVerifyResponseByProvider(ctx context.Context, provider *signv1.VerifyRequestProvider, resp *signv1.VerifyResponse) *signv1.VerifyResponse {
	if !resp.GetSuccess() || len(resp.GetSigners()) == 0 {
		return resp
	}

	if provider.GetRequest() == nil {
		return resp
	}

	var matched []*signv1.SignerInfo

	switch p := provider.GetRequest().(type) {
	case *signv1.VerifyRequestProvider_Key:
		if p.Key == nil || p.Key.GetPublicKey() == "" {
			return resp
		}

		userPEM, err := cosign.ResolvePublicKeyToPEM(ctx, p.Key.GetPublicKey())
		if err != nil {
			msg := fmt.Sprintf("failed to load public key: %v", err)

			return &signv1.VerifyResponse{Success: false, ErrorMessage: &msg}
		}

		for _, s := range resp.GetSigners() {
			if k := s.GetKey(); k != nil && cosign.PublicKeyPEMsEqual(userPEM, k.GetPublicKey()) {
				matched = append(matched, s)
			}
		}

		if len(matched) == 0 {
			errMsg := "no verified signature matched the provided public key"

			return &signv1.VerifyResponse{Success: false, ErrorMessage: &errMsg, Signers: resp.GetSigners()}
		}
	case *signv1.VerifyRequestProvider_Oidc:
		if p.Oidc == nil {
			return resp
		}

		reqIssuer, reqSubject := p.Oidc.GetIssuer(), p.Oidc.GetSubject()
		for _, s := range resp.GetSigners() {
			if o := s.GetOidc(); o != nil && oidcSignerMatches(reqIssuer, reqSubject, o.GetIssuer(), o.GetSubject()) {
				matched = append(matched, s)
			}
		}

		if len(matched) == 0 {
			errMsg := "no verified signature matched the provided OIDC issuer/subject"

			return &signv1.VerifyResponse{Success: false, ErrorMessage: &errMsg, Signers: resp.GetSigners()}
		}
	default:
		return resp
	}

	return &signv1.VerifyResponse{Success: true, Signers: matched}
}

// oidcSignerMatches returns true if signer issuer/subject match the requested values.
// Requested values support exact match or regex (empty means match any).
func oidcSignerMatches(reqIssuer, reqSubject, signerIssuer, signerSubject string) bool {
	return oidcValueMatches(reqIssuer, signerIssuer) && oidcValueMatches(reqSubject, signerSubject)
}

func oidcValueMatches(reqValue, signerValue string) bool {
	if reqValue == "" {
		return true
	}

	if re, err := regexp.Compile(reqValue); err == nil {
		return re.MatchString(signerValue)
	}

	return reqValue == signerValue
}
