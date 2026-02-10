// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"errors"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
	storev1 "github.com/agntcy/dir/api/store/v1"
	"github.com/agntcy/dir/client/utils/cosign"
)

// Sign routes to the appropriate signing method based on provider type.
// This is the main entry point for signing operations.
func (c *Client) Sign(ctx context.Context, req *signv1.SignRequest) (*signv1.SignResponse, error) {
	if req.GetProvider() == nil || req.GetProvider().GetRequest() == nil {
		return nil, errors.New("signature provider must be specified")
	}

	if req.GetRecordRef() == nil || req.GetRecordRef().GetCid() == "" {
		return nil, errors.New("record ref must be specified")
	}

	// Handle results from signing
	var (
		signature *signv1.Signature
		pubKey    *signv1.PublicKey
		signErr   error
	)

	// Route to the appropriate signing method based on provider type
	switch provider := req.GetProvider().GetRequest().(type) {
	case *signv1.SignRequestProvider_Key:
		signature, pubKey, signErr = cosign.SignBlobWithKey(ctx, []byte(req.GetRecordRef().GetCid()), provider.Key)

	case *signv1.SignRequestProvider_Oidc:
		signature, pubKey, signErr = cosign.SignBlobWithOIDC(ctx, []byte(req.GetRecordRef().GetCid()), provider.Oidc)

	default:
		return nil, fmt.Errorf("unsupported signature provider type: %T", provider)
	}

	// Check if signing was successful
	if signErr != nil {
		return nil, fmt.Errorf("failed to sign record: %w", signErr)
	}

	// Push data to store as referrers
	err := c.pushReferrersToStore(ctx, req.GetRecordRef(), signature, pubKey)
	if err != nil {
		return nil, fmt.Errorf("failed to push referrers to store: %w", err)
	}

	return &signv1.SignResponse{
		Signature: signature,
	}, nil
}

func (c *Client) pushReferrersToStore(ctx context.Context, recordRef *corev1.RecordRef, signature *signv1.Signature, publicKey *signv1.PublicKey) error {
	// Create public key referrer
	publicKeyReferrer, err := publicKey.MarshalReferrer()
	if err != nil {
		return fmt.Errorf("failed to encode public key to referrer: %w", err)
	}

	// Push public key to store as a referrer
	err = c.PushReferrer(ctx, &storev1.PushReferrerRequest{
		RecordRef: recordRef,
		Referrer:  publicKeyReferrer,
	})
	if err != nil {
		return fmt.Errorf("failed to store public key: %w", err)
	}

	// Create signature referrer
	signatureReferrer, err := signature.MarshalReferrer()
	if err != nil {
		return fmt.Errorf("failed to encode signature to referrer: %w", err)
	}

	// Push signature to store as a referrer
	err = c.PushReferrer(ctx, &storev1.PushReferrerRequest{
		RecordRef: recordRef,
		Referrer:  signatureReferrer,
	})
	if err != nil {
		return fmt.Errorf("failed to store signature: %w", err)
	}

	return nil
}
