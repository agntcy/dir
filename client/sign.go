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
	"github.com/agntcy/dir/utils/cosign"
)

type SignOpts struct {
	FulcioURL       string
	RekorURL        string
	TimestampURL    string
	OIDCProviderURL string
	OIDCClientID    string
	OIDCToken       string
	Key             string
}

// Sign routes to the appropriate signing method based on provider type.
// This is the main entry point for signing operations.
func (c *Client) Sign(ctx context.Context, req *signv1.SignRequest) (*signv1.SignResponse, error) {
	if req.GetProvider() == nil {
		return nil, errors.New("signature provider must be specified")
	}

	switch provider := req.GetProvider().GetRequest().(type) {
	case *signv1.SignRequestProvider_Key:
		return c.SignWithKey(ctx, req)
	case *signv1.SignRequestProvider_Oidc:
		return c.SignWithOIDC(ctx, req)
	default:
		return nil, fmt.Errorf("unsupported signature provider type: %T", provider)
	}
}

// SignWithOIDC signs the record using keyless OIDC service-based signing.
// The OIDC ID Token can be provided by the caller, or cosign will handle interactive OIDC flow.
// This implementation uses cosign sign-blob command for OIDC signing.
func (c *Client) SignWithOIDC(ctx context.Context, req *signv1.SignRequest) (*signv1.SignResponse, error) {
	// Validate request.
	if req.GetRecordRef() == nil {
		return nil, errors.New("record ref must be set")
	}

	oidcSigner := req.GetProvider().GetOidc()

	digest, err := corev1.ConvertCIDToDigest(req.GetRecordRef().GetCid())
	if err != nil {
		return nil, fmt.Errorf("failed to convert CID to digest: %w", err)
	}

	payloadBytes, err := cosign.GeneratePayload(digest.String())
	if err != nil {
		return nil, fmt.Errorf("failed to generate payload: %w", err)
	}

	// Prepare options for signing
	signOpts := &cosign.SignBlobOIDCOptions{
		Payload: payloadBytes,
		IDToken: oidcSigner.GetIdToken(),
	}

	// Set URLs from options if provided
	if opts := oidcSigner.GetOptions(); opts != nil {
		signOpts.FulcioURL = opts.GetFulcioUrl()
		signOpts.RekorURL = opts.GetRekorUrl()
		signOpts.TimestampURL = opts.GetTimestampUrl()
		signOpts.OIDCProviderURL = opts.GetOidcProviderUrl()
	}

	// Sign using utility function
	result, err := cosign.SignBlobWithOIDC(ctx, signOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to sign with OIDC: %w", err)
	}

	// Create the signature object
	signatureObj := &signv1.Signature{
		Signature: result.Signature,
		PublicKey: &result.PublicKey,
		Annotations: map[string]string{
			"payload": string(payloadBytes),
		},
	}

	// Push signature to store
	err = c.pushSignatureToStore(ctx, req.GetRecordRef().GetCid(), signatureObj)
	if err != nil {
		return nil, fmt.Errorf("failed to store signature: %w", err)
	}

	return &signv1.SignResponse{
		Signature: signatureObj,
	}, nil
}

func (c *Client) SignWithKey(ctx context.Context, req *signv1.SignRequest) (*signv1.SignResponse, error) {
	keySigner := req.GetProvider().GetKey()

	password := keySigner.GetPassword()
	if password == nil {
		password = []byte("") // Empty password is valid for cosign.
	}

	digest, err := corev1.ConvertCIDToDigest(req.GetRecordRef().GetCid())
	if err != nil {
		return nil, fmt.Errorf("failed to convert CID to digest: %w", err)
	}

	payloadBytes, err := cosign.GeneratePayload(digest.String())
	if err != nil {
		return nil, fmt.Errorf("failed to generate payload: %w", err)
	}

	// Prepare options for signing
	signOpts := &cosign.SignBlobKeyOptions{
		Payload:    payloadBytes,
		PrivateKey: keySigner.GetPrivateKey(),
		Password:   password,
	}

	// Sign using utility function
	result, err := cosign.SignBlobWithKey(ctx, signOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to sign with key: %w", err)
	}

	// Create the signature object
	signatureObj := &signv1.Signature{
		Signature: result.Signature,
		PublicKey: &result.PublicKey,
		Annotations: map[string]string{
			"payload": string(payloadBytes),
		},
	}

	// Push signature to store
	err = c.pushSignatureToStore(ctx, req.GetRecordRef().GetCid(), signatureObj)
	if err != nil {
		return nil, fmt.Errorf("failed to store signature: %w", err)
	}

	return &signv1.SignResponse{
		Signature: signatureObj,
	}, nil
}

// pushSignatureToStore stores a signature using the PushReferrer RPC.
func (c *Client) pushSignatureToStore(ctx context.Context, recordCID string, signature *signv1.Signature) error {
	// Create streaming client
	stream, err := c.StoreServiceClient.PushReferrer(ctx)
	if err != nil {
		return fmt.Errorf("failed to create push referrer stream: %w", err)
	}

	// Create the push referrer request
	req := &storev1.PushReferrerRequest{
		RecordRef: &corev1.RecordRef{
			Cid: recordCID,
		},
		Options: &storev1.PushReferrerRequest_Signature{
			Signature: signature,
		},
	}

	// Send the request
	if err := stream.Send(req); err != nil {
		return fmt.Errorf("failed to send push referrer request: %w", err)
	}

	// Close send stream
	if err := stream.CloseSend(); err != nil {
		return fmt.Errorf("failed to close send stream: %w", err)
	}

	// Receive response
	_, err = stream.Recv()
	if err != nil {
		return fmt.Errorf("failed to receive push referrer response: %w", err)
	}

	return nil
}
