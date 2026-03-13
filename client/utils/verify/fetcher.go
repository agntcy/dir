// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package verify provides signature verification types and logic (Fetcher, VerifyWithFetcher).
package verify

import (
	"context"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
	"github.com/agntcy/dir/client/utils/cosign"
)

// Fetcher supplies signatures and public keys for a record.
type Fetcher interface {
	PullSignatures(ctx context.Context, recordRef *corev1.CID) ([]*signv1.Signature, error)
	PullPublicKeys(ctx context.Context, recordRef *corev1.CID) ([]string, error)
}

// PerSignatureResult is the verification result for one signer (for DB cache).
type PerSignatureResult struct {
	SignerKey  string
	Status     string // "verified" or "failed"
	SignerInfo *signv1.SignerInfo
}

// VerifyWithFetcher runs signature verification using the given fetcher and returns the response plus per-signature results.
// Used by Client.Verify (with the client as Fetcher) and by the reconciler (with a store Fetcher).
//
//nolint:cyclop
func VerifyWithFetcher(ctx context.Context, req *signv1.VerifyRequest, fetcher Fetcher) (*signv1.VerifyResponse, []PerSignatureResult, error) {
	if req.GetRecordRef() == nil || req.GetRecordRef().GetCid() == "" {
		return nil, nil, fmt.Errorf("record ref is required")
	}

	provider := req.GetProvider()
	if provider == nil || provider.GetRequest() == nil {
		req.Provider = &signv1.VerifyRequestProvider{
			Request: &signv1.VerifyRequestProvider_Any{
				Any: &signv1.VerifyWithAny{
					OidcOptions: signv1.DefaultVerifyOptionsOIDC,
				},
			},
		}
		provider = req.GetProvider()
	}

	signatures, err := fetcher.PullSignatures(ctx, req.GetRecordRef())
	if err != nil {
		return nil, nil, fmt.Errorf("pull signatures: %w", err)
	}

	if len(signatures) == 0 {
		errMsg := "no signatures found"

		return &signv1.VerifyResponse{
			Success:      false,
			ErrorMessage: &errMsg,
		}, nil, nil
	}

	var publicKeys []string

	switch {
	case provider.GetKey() != nil:
		publicKeys = []string{provider.GetKey().GetPublicKey()}
	case provider.GetAny() != nil:
		publicKeys, err = fetcher.PullPublicKeys(ctx, req.GetRecordRef())
		if err != nil {
			return nil, nil, fmt.Errorf("pull public keys: %w", err)
		}
	}

	payload := []byte(req.GetRecordRef().GetCid())

	var (
		seenKeys   = make(map[string]bool)
		signers    []*signv1.SignerInfo
		perSig     []PerSignatureResult
		signerInfo *signv1.SignerInfo
		verifyErr  error
	)

	for _, sig := range signatures {
		switch p := provider.GetRequest().(type) {
		case *signv1.VerifyRequestProvider_Oidc:
			signerInfo, verifyErr = cosign.VerifyWithOIDC(payload, p.Oidc, sig)
		case *signv1.VerifyRequestProvider_Key:
			signerInfo, verifyErr = cosign.VerifyWithKeys(ctx, payload, publicKeys, sig)
		case *signv1.VerifyRequestProvider_Any:
			signerInfo, verifyErr = verifyWithAny(ctx, payload, publicKeys, sig)
		default:
			return nil, nil, fmt.Errorf("unsupported verification provider type")
		}

		if verifyErr != nil {
			continue // Failed verifications have no SignerInfo; we report only one per signer (verified)
		}

		signerKey := getSignerKey(signerInfo)
		if seenKeys[signerKey] {
			continue // Only report one result per signer
		}

		seenKeys[signerKey] = true

		perSig = append(perSig, PerSignatureResult{
			SignerKey:  signerKey,
			Status:     "verified",
			SignerInfo: signerInfo,
		})
		signers = append(signers, signerInfo)
	}

	// If no valid signers found, return not trusted
	if len(signers) == 0 {
		errMsg := "no valid signatures found matching verification criteria"

		return &signv1.VerifyResponse{
			Success:      false,
			ErrorMessage: &errMsg,
		}, perSig, nil
	}

	return &signv1.VerifyResponse{
		Success: true,
		Signers: signers,
	}, perSig, nil
}

func verifyWithAny(ctx context.Context, payload []byte, publicKeys []string, sig *signv1.Signature) (*signv1.SignerInfo, error) {
	if len(sig.GetContentBundle()) == 0 {
		info, err := cosign.VerifyWithKeys(ctx, payload, publicKeys, sig)
		if err != nil {
			return nil, fmt.Errorf("failed to verify signature with keys: %w", err)
		}

		return info, nil
	}

	info, err := cosign.VerifyWithOIDC(payload, &signv1.VerifyWithOIDC{
		Options: signv1.DefaultVerifyOptionsOIDC.GetDefaultOptions(),
	}, sig)
	if err != nil {
		return nil, fmt.Errorf("failed to verify signature with OIDC: %w", err)
	}

	return info, nil
}

func getSignerKey(signer *signv1.SignerInfo) string {
	switch s := signer.GetType().(type) {
	case *signv1.SignerInfo_Key:
		return "key:" + s.Key.String()
	case *signv1.SignerInfo_Oidc:
		return "oidc:" + s.Oidc.String()
	default:
		return ""
	}
}
