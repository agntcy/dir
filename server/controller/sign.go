// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package controller

import (
	"context"
	"strings"

	signv1 "github.com/agntcy/dir/api/sign/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var signLogger = logging.Logger("controller/sign")

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
		if obj.GetStatus() != "verified" {
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
		resp = filterVerifyResponseByProvider(prov, resp)
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

// filterVerifyResponseByProvider filters the response to keep only signers that match the requested provider (key or OIDC).
func filterVerifyResponseByProvider(provider *signv1.VerifyRequestProvider, resp *signv1.VerifyResponse) *signv1.VerifyResponse {
	if resp == nil || len(resp.GetSigners()) == 0 {
		return resp
	}

	var filtered []*signv1.SignerInfo

	switch p := provider.GetRequest().(type) {
	case *signv1.VerifyRequestProvider_Key:
		filtered = filterSignersByKey(resp.GetSigners(), p.Key)
	case *signv1.VerifyRequestProvider_Oidc:
		filtered = filterSignersByOIDC(resp.GetSigners(), p.Oidc)
	case *signv1.VerifyRequestProvider_Any:
		filtered = resp.GetSigners()
	default:
		return resp
	}

	out := &signv1.VerifyResponse{
		Success: resp.GetSuccess(),
		Signers: filtered,
	}
	if len(filtered) == 0 {
		out.Success = false
		msg := "no verified signers match the requested provider"
		out.ErrorMessage = &msg
	}

	return out
}

func filterSignersByKey(signers []*signv1.SignerInfo, key *signv1.VerifyWithKey) []*signv1.SignerInfo {
	if key == nil {
		return nil
	}

	wantPub := strings.TrimSpace(key.GetPublicKey())

	var out []*signv1.SignerInfo

	for _, s := range signers {
		k := s.GetKey()
		if k == nil {
			continue
		}

		if wantPub != "" {
			if strings.TrimSpace(k.GetPublicKey()) != wantPub {
				continue
			}
		}

		out = append(out, s)
	}

	return out
}

func filterSignersByOIDC(signers []*signv1.SignerInfo, oidc *signv1.VerifyWithOIDC) []*signv1.SignerInfo {
	if oidc == nil {
		return nil
	}

	var out []*signv1.SignerInfo

	for _, s := range signers {
		o := s.GetOidc()
		if o == nil {
			continue
		}

		if oidc.GetIssuer() != "" && o.GetIssuer() != oidc.GetIssuer() {
			continue
		}

		if oidc.GetSubject() != "" && o.GetSubject() != oidc.GetSubject() {
			continue
		}

		out = append(out, s)
	}

	return out
}
