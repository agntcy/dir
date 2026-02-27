// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	signv1 "github.com/agntcy/dir/api/sign/v1"
	"github.com/agntcy/dir/client/utils/cosign"
)

// Verify returns the cached signature verification result for a record by querying the server.
// All matching against the user's provider (--key or --oidc-issuer) is done client-side using
// the signers returned by the server; verification succeeds only if at least one signer matches.
func (c *Client) Verify(ctx context.Context, req *signv1.VerifyRequest) (*signv1.VerifyResponse, error) {
	if req.GetRecordRef() == nil || req.GetRecordRef().GetCid() == "" {
		return nil, fmt.Errorf("record CID is required")
	}

	_, err := c.Lookup(ctx, req.GetRecordRef())
	if err != nil {
		if strings.Contains(err.Error(), "record not found") {
			errMsg := "record not found"

			return &signv1.VerifyResponse{
				Success:      false,
				ErrorMessage: &errMsg,
			}, nil
		}

		return nil, fmt.Errorf("failed to lookup record: %w", err)
	}

	resp, err := c.SignServiceClient.Verify(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("verify: %w", err)
	}

	// Match against user's provider using only the data received from the server.
	if prov := req.GetProvider(); prov != nil {
		resp = filterVerifyResponseByProvider(ctx, prov, resp)
	}

	return resp, nil
}

// filterVerifyResponseByProvider keeps only signers that match the requested provider (key or OIDC).
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
