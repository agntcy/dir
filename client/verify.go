// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
	storev1 "github.com/agntcy/dir/api/store/v1"
	"github.com/agntcy/dir/client/utils/cosign"
)

// Verify verifies signatures for a record by fetching referrers and performing
// client-side verification. This does not require a server-side verification endpoint.
//
//nolint:cyclop
func (c *Client) Verify(ctx context.Context, req *signv1.VerifyRequest) (*signv1.VerifyResponse, error) {
	// Validate input
	if req.GetProvider() == nil || req.GetProvider().GetRequest() == nil {
		// Set default provider to "any" if not specified, which accepts any valid signature.
		req.Provider = &signv1.VerifyRequestProvider{
			Request: &signv1.VerifyRequestProvider_Any{
				Any: &signv1.VerifyWithAny{
					OidcOptions: signv1.DefaultVerifyOptionsOIDC,
				},
			},
		}
	}

	if req.GetRecordRef() == nil || req.GetRecordRef().GetCid() == "" {
		return nil, fmt.Errorf("record CID is required")
	}

	// Pull signature referrers
	signatures, err := c.pullSignatures(ctx, req.GetRecordRef())
	if err != nil {
		return nil, fmt.Errorf("failed to pull signatures: %w", err)
	}

	if len(signatures) == 0 {
		errMsg := "no signatures found"

		return &signv1.VerifyResponse{
			Success:      false,
			ErrorMessage: &errMsg,
		}, nil
	}

	// Pull public key referrers (for any-based verification)
	// If a specific key is provided in the request, use that instead of pulling from the store.
	var publicKeys []string

	switch {
	case req.GetProvider().GetKey() != nil:
		publicKeys = []string{req.GetProvider().GetKey().GetPublicKey()}

	case req.GetProvider().GetAny() != nil:
		publicKeys, err = c.pullPublicKeys(ctx, req.GetRecordRef())
		if err != nil {
			return nil, fmt.Errorf("failed to pull public keys: %w", err)
		}
	}

	// Payload that needs to be verified is the record CID in bytes
	payload := []byte(req.GetRecordRef().GetCid())

	// Verify each signature
	var (
		seenKeys   = make(map[string]bool) // Track unique signers by key/identity
		signers    []*signv1.SignerInfo
		signerInfo *signv1.SignerInfo
		verifyErr  error
	)

	for _, sig := range signatures {
		// Check if a specific provider is requested
		switch provider := req.GetProvider().GetRequest().(type) {
		case *signv1.VerifyRequestProvider_Oidc:
			signerInfo, verifyErr = cosign.VerifyWithOIDC(payload, provider.Oidc, sig)

		case *signv1.VerifyRequestProvider_Key:
			signerInfo, verifyErr = cosign.VerifyWithKeys(ctx, payload, publicKeys, sig)

		case *signv1.VerifyRequestProvider_Any:
			// VerifyWithAny accepts any valid signature.
			// If a signature has no bundle, it must be verified with a key.
			if len(sig.GetContentBundle()) == 0 {
				signerInfo, verifyErr = cosign.VerifyWithKeys(ctx, payload, publicKeys, sig)
			} else {
				signerInfo, verifyErr = cosign.VerifyWithOIDC(payload, &signv1.VerifyWithOIDC{
					Options: provider.Any.GetOidcOptions().GetDefaultOptions(),
				}, sig)
			}

		default:
			return nil, fmt.Errorf("unsupported verification provider type")
		}

		if verifyErr != nil {
			logger.Debug("Signature verification failed for this signature", "error", verifyErr)

			continue
		}

		// Deduplicate signers based on unique key (public key or issuer+subject)
		signerKey := getSignerKey(signerInfo)
		if seenKeys[signerKey] {
			continue // Skip duplicate signer
		}

		seenKeys[signerKey] = true

		signers = append(signers, signerInfo)
	}

	// If no valid signers found, return not trusted
	if len(signers) == 0 {
		errMsg := "no valid signatures found matching verification criteria"

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

// getSignerKey returns a unique key for a signer to use for deduplication.
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

// pullSignatures fetches all signature referrers for a record.
func (c *Client) pullSignatures(ctx context.Context, recordRef *corev1.RecordRef) ([]*signv1.Signature, error) {
	referrerType := corev1.SignatureReferrerType

	respCh, err := c.PullReferrer(ctx, &storev1.PullReferrerRequest{
		RecordRef:    recordRef,
		ReferrerType: &referrerType,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create pull referrer stream: %w", err)
	}

	var signatures []*signv1.Signature

	for resp := range respCh {
		if resp.GetReferrer() == nil {
			continue
		}

		sig := &signv1.Signature{}
		if err := sig.UnmarshalReferrer(resp.GetReferrer()); err != nil {
			logger.Debug("Failed to unmarshal signature referrer", "error", err)

			continue
		}

		signatures = append(signatures, sig)
	}

	return signatures, nil
}

// pullPublicKeys fetches all public key referrers for a record.
func (c *Client) pullPublicKeys(ctx context.Context, recordRef *corev1.RecordRef) ([]string, error) {
	referrerType := corev1.PublicKeyReferrerType

	respCh, err := c.PullReferrer(ctx, &storev1.PullReferrerRequest{
		RecordRef:    recordRef,
		ReferrerType: &referrerType,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create pull referrer stream: %w", err)
	}

	var publicKeys []string

	for resp := range respCh {
		if resp.GetReferrer() == nil {
			continue
		}

		pk := &signv1.PublicKey{}
		if err := pk.UnmarshalReferrer(resp.GetReferrer()); err != nil {
			logger.Debug("Failed to unmarshal public key referrer", "error", err)

			continue
		}

		if key := pk.GetKey(); key != "" {
			publicKeys = append(publicKeys, key)
		}
	}

	return publicKeys, nil
}
