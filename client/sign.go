// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
	"github.com/agntcy/dir/utils/cosign"
)

const (
	DefaultFulcioURL       = "https://fulcio.sigstage.dev"
	DefaultRekorURL        = "https://rekor.sigstage.dev"
	DefaultTimestampURL    = "https://timestamp.sigstage.dev/api/v1/timestamp"
	DefaultOIDCProviderURL = "https://oauth2.sigstage.dev/auth"
	DefaultOIDCClientID    = "sigstore"

	DefaultFulcioTimeout             = 30 * time.Second
	DefaultTimestampAuthorityTimeout = 30 * time.Second
	DefaultRekorTimeout              = 90 * time.Second
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
		return nil, fmt.Errorf("signature provider must be specified")
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

// SignOIDC signs the record using keyless OIDC service-based signing.
// The OIDC ID Token must be provided by the caller.
// An ephemeral keypair is generated for signing.
func (c *Client) SignWithOIDC(ctx context.Context, req *signv1.SignRequest) (*signv1.SignResponse, error) {
	// Run cosign sign command and retrieve signature payload and certificate
	// cosign sign --key cosign.key --payload payload.json --certificate certificate.pem --output-signature signature.sig

	// // Validate request.
	// if req.GetRecord() == nil {
	// 	return nil, errors.New("record must be set")
	// }

	// oidcSigner := req.GetProvider().GetOidc()

	// // Load signing options.
	// var signOpts sign.BundleOptions
	// {
	// 	// Define config to use for signing.
	// 	signingConfig, err := root.NewSigningConfig(
	// 		root.SigningConfigMediaType02,
	// 		// Fulcio URLs
	// 		[]root.Service{
	// 			{
	// 				URL:                 setOrDefault(oidcSigner.GetOptions().GetFulcioUrl(), DefaultFulcioURL),
	// 				MajorAPIVersion:     1,
	// 				ValidityPeriodStart: time.Now().Add(-time.Hour),
	// 				ValidityPeriodEnd:   time.Now().Add(time.Hour),
	// 			},
	// 		},
	// 		// OIDC Provider URLs
	// 		// Usage and requirements: https://docs.sigstore.dev/certificate_authority/oidc-in-fulcio/
	// 		[]root.Service{
	// 			{
	// 				URL:                 setOrDefault(oidcSigner.GetOptions().GetOidcProviderUrl(), DefaultOIDCProviderURL),
	// 				MajorAPIVersion:     1,
	// 				ValidityPeriodStart: time.Now().Add(-time.Hour),
	// 				ValidityPeriodEnd:   time.Now().Add(time.Hour),
	// 			},
	// 		},
	// 		// Rekor URLs
	// 		[]root.Service{
	// 			{
	// 				URL:                 setOrDefault(oidcSigner.GetOptions().GetRekorUrl(), DefaultRekorURL),
	// 				MajorAPIVersion:     1,
	// 				ValidityPeriodStart: time.Now().Add(-time.Hour),
	// 				ValidityPeriodEnd:   time.Now().Add(time.Hour),
	// 			},
	// 		},
	// 		root.ServiceConfiguration{
	// 			Selector: v1.ServiceSelector_ANY,
	// 		},
	// 		[]root.Service{
	// 			{
	// 				URL:                 setOrDefault(oidcSigner.GetOptions().GetTimestampUrl(), DefaultTimestampURL),
	// 				MajorAPIVersion:     1,
	// 				ValidityPeriodStart: time.Now().Add(-time.Hour),
	// 				ValidityPeriodEnd:   time.Now().Add(time.Hour),
	// 			},
	// 		},
	// 		root.ServiceConfiguration{
	// 			Selector: v1.ServiceSelector_ANY,
	// 		},
	// 	)
	// 	if err != nil {
	// 		return nil, fmt.Errorf("failed to create signing config: %w", err)
	// 	}

	// 	// Use fulcio to sign the agent.
	// 	fulcioURL, err := root.SelectService(signingConfig.FulcioCertificateAuthorityURLs(), []uint32{1}, time.Now())
	// 	if err != nil {
	// 		return nil, fmt.Errorf("failed to select fulcio URL: %w", err)
	// 	}

	// 	fulcioOpts := &sign.FulcioOptions{
	// 		BaseURL: fulcioURL,
	// 		Timeout: DefaultFulcioTimeout,
	// 		Retries: 1,
	// 	}
	// 	signOpts.CertificateProvider = sign.NewFulcio(fulcioOpts)
	// 	signOpts.CertificateProviderOptions = &sign.CertificateProviderOptions{
	// 		IDToken: oidcSigner.GetIdToken(),
	// 	}

	// 	// Use timestamp authortiy to sign the agent.
	// 	tsaURLs, err := root.SelectServices(signingConfig.TimestampAuthorityURLs(),
	// 		signingConfig.TimestampAuthorityURLsConfig(), []uint32{1}, time.Now())
	// 	if err != nil {
	// 		return nil, fmt.Errorf("failed to select timestamp authority URL: %w", err)
	// 	}

	// 	for _, tsaURL := range tsaURLs {
	// 		tsaOpts := &sign.TimestampAuthorityOptions{
	// 			URL:     tsaURL,
	// 			Timeout: DefaultTimestampAuthorityTimeout,
	// 			Retries: 1,
	// 		}
	// 		signOpts.TimestampAuthorities = append(signOpts.TimestampAuthorities, sign.NewTimestampAuthority(tsaOpts))
	// 	}

	// 	// Use rekor to sign the agent.
	// 	rekorURLs, err := root.SelectServices(signingConfig.RekorLogURLs(),
	// 		signingConfig.RekorLogURLsConfig(), []uint32{1}, time.Now())
	// 	if err != nil {
	// 		return nil, fmt.Errorf("failed to select rekor URL: %w", err)
	// 	}

	// 	for _, rekorURL := range rekorURLs {
	// 		rekorOpts := &sign.RekorOptions{
	// 			BaseURL: rekorURL,
	// 			Timeout: DefaultRekorTimeout,
	// 			Retries: 1,
	// 		}
	// 		signOpts.TransparencyLogs = append(signOpts.TransparencyLogs, sign.NewRekor(rekorOpts))
	// 	}
	// }

	// // Generate an ephemeral keypair for signing.
	// signKeypair, err := sign.NewEphemeralKeypair(nil)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to create ephemeral keypair: %w", err)
	// }

	// signature, err := c.sign(ctx, req.GetRecord(), signKeypair, signOpts)
	// if err != nil {
	// 	return nil, err
	// }

	return &signv1.SignResponse{
		Signature: nil,
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

	// Create payload temporary file
	payload := cosign.GeneratePayload("localhost:5000", "dir", digest.String())
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}
	err = os.WriteFile("payload-temp.json", payloadBytes, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to write payload: %w", err)
	}
	defer os.Remove("payload-temp.json")

	// Write private key to temporary file
	keyFile := "cosign.key"
	err = os.WriteFile(keyFile, keySigner.GetPrivateKey(), 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to write private key: %w", err)
	}
	defer os.Remove(keyFile)

	cmd := exec.Command("cosign", "sign-blob",
		"-y",
		"--key", keyFile,
		"--output-signature", "signature.sig",
		"payload-temp.json")

	// Set environment variables
	cmd.Env = append(os.Environ(), "COSIGN_PASSWORD="+string(password))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("cosign sign failed: %v\nOutput: %s", err, string(output))
	}

	signature, err := os.ReadFile("signature.sig")
	if err != nil {
		return nil, fmt.Errorf("failed to read signature: %w", err)
	}

	cosignKeypair, err := cosign.LoadKeypair(keySigner.GetPrivateKey(), password)
	if err != nil {
		return nil, fmt.Errorf("failed to load cosign keypair: %w", err)
	}

	publicKey, err := cosignKeypair.GetPublicKeyPem()
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %w", err)
	}

	// Create the signature object
	signatureObj := &signv1.Signature{
		Signature: string(signature),
		PublicKey: &publicKey,
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

func setOrDefault(value string, defaultValue string) string {
	if value == "" {
		value = defaultValue
	}

	return value
}

// pushSignatureToStore stores a signature using the new PushSignature RPC.
func (c *Client) pushSignatureToStore(ctx context.Context, recordCID string, signature *signv1.Signature) error {
	req := &signv1.PushSignatureRequest{
		RecordRef: &corev1.RecordRef{Cid: recordCID},
		Signature: signature,
	}

	_, err := c.SignServiceClient.PushSignature(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to push signature to store: %w", err)
	}

	return nil
}
