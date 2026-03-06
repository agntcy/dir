// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package verify provides signature verification logic that can be driven by any
// fetcher (e.g. gRPC client or store), so the client and reconciler share one implementation.
package verify

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
	"github.com/agntcy/dir/utils/cosign"
	"google.golang.org/protobuf/proto"
)

const (
	StatusVerified = "verified"
	StatusFailed   = "failed"
)

// SigWithDigest pairs a signature with its referrer digest (cache key).
type SigWithDigest struct {
	Digest string
	Sig    *signv1.Signature
}

// PerSignatureResult is the verification result for one signature (for DB cache).
type PerSignatureResult struct {
	Digest       string
	Status       string // StatusVerified or StatusFailed
	SignerInfo   *signv1.SignerInfo
	ErrorMessage string
}

// ReferrerDigest returns a stable digest for a record referrer (e.g. for use as a cache key).
func ReferrerDigest(ref *corev1.RecordReferrer) string {
	if ref == nil {
		return ""
	}

	data, _ := proto.Marshal(ref)
	h := sha256.Sum256(data)

	return hex.EncodeToString(h[:])
}

// Fetcher supplies signatures and public keys for a record (implemented by client or store).
type Fetcher interface {
	PullSignatures(ctx context.Context, recordRef *corev1.RecordRef) ([]SigWithDigest, error)
	PullPublicKeys(ctx context.Context, recordRef *corev1.RecordRef) ([]string, error)
}

// Verify runs signature verification using the given fetcher and returns the response plus per-signature results.
// Does not perform record lookup or from_server; the caller handles those.
//
//nolint:cyclop
func Verify(ctx context.Context, req *signv1.VerifyRequest, fetcher Fetcher) (*signv1.VerifyResponse, []PerSignatureResult, error) {
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
		msg := "no signatures found"

		return &signv1.VerifyResponse{
			Success:      false,
			ErrorMessage: &msg,
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

	for _, item := range signatures {
		switch p := provider.GetRequest().(type) {
		case *signv1.VerifyRequestProvider_Oidc:
			signerInfo, verifyErr = cosign.VerifyWithOIDC(payload, p.Oidc, item.Sig)
		case *signv1.VerifyRequestProvider_Key:
			signerInfo, verifyErr = cosign.VerifyWithKeys(ctx, payload, publicKeys, item.Sig)
		case *signv1.VerifyRequestProvider_Any:
			signerInfo, verifyErr = verifyWithAny(ctx, payload, publicKeys, item.Sig)
		default:
			return nil, nil, fmt.Errorf("unsupported verification provider type")
		}

		if verifyErr != nil {
			perSig = append(perSig, PerSignatureResult{
				Digest:       item.Digest,
				Status:       StatusFailed,
				ErrorMessage: verifyErr.Error(),
			})

			continue
		}

		perSig = append(perSig, PerSignatureResult{
			Digest:     item.Digest,
			Status:     StatusVerified,
			SignerInfo: signerInfo,
		})

		key := signerKey(signerInfo)
		if !seenKeys[key] {
			seenKeys[key] = true

			signers = append(signers, signerInfo)
		}
	}

	if len(signers) == 0 {
		msg := "no valid signatures found matching verification criteria"

		return &signv1.VerifyResponse{
			Success:      false,
			ErrorMessage: &msg,
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

func signerKey(signer *signv1.SignerInfo) string {
	if signer == nil {
		return ""
	}

	switch s := signer.GetType().(type) {
	case *signv1.SignerInfo_Key:
		return "key:" + s.Key.String()
	case *signv1.SignerInfo_Oidc:
		return "oidc:" + s.Oidc.String()
	default:
		return ""
	}
}
