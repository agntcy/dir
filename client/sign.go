// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:mnd,gosec
package client

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
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

	// Create payload temporary file (same as SignWithKey)
	payload := cosign.GeneratePayload("localhost:5000", "dir", digest.String())

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Write payload to temporary file
	payloadFile := "payload-oidc-temp.json"
	err = os.WriteFile(payloadFile, payloadBytes, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to write payload: %w", err)
	}
	defer os.Remove(payloadFile)

	// Prepare output files for signature and certificate
	signatureFile := "signature-oidc-temp.sig"
	certificateFile := "certificate-oidc-temp.crt"
	bundleFile := "bundle-oidc-temp.json"

	// Build cosign sign-blob command with OIDC options
	args := []string{
		"sign-blob",
		"-y",
		"--output-signature", signatureFile,
		"--output-certificate", certificateFile,
		"--bundle", bundleFile,
		"--new-bundle-format",    // Use new bundle format
		"--insecure-skip-verify", // Skip SCT verification for staging/testing
	}

	// Add identity token only if provided
	// If no token provided, cosign will attempt interactive OIDC flow
	if idToken := oidcSigner.GetIdToken(); idToken != "" {
		args = append(args, "--identity-token", idToken)
	} else {
		// For interactive OIDC flow, we need to ensure proper provider configuration
		// Add OIDC client ID for better auth flow
		args = append(args, "--oidc-client-id", DefaultOIDCClientID)
	}

	// Add optional OIDC URLs if provided
	if opts := oidcSigner.GetOptions(); opts != nil {
		if fulcioURL := opts.GetFulcioUrl(); fulcioURL != "" {
			args = append(args, "--fulcio-url", fulcioURL)
		} else {
			args = append(args, "--fulcio-url", DefaultFulcioURL)
		}

		if rekorURL := opts.GetRekorUrl(); rekorURL != "" {
			args = append(args, "--rekor-url", rekorURL)
		} else {
			args = append(args, "--rekor-url", DefaultRekorURL)
		}

		if timestampURL := opts.GetTimestampUrl(); timestampURL != "" {
			args = append(args, "--timestamp-server-url", timestampURL)
		} else {
			args = append(args, "--timestamp-server-url", DefaultTimestampURL)
		}

		if oidcProviderURL := opts.GetOidcProviderUrl(); oidcProviderURL != "" {
			args = append(args, "--oidc-issuer", oidcProviderURL)
		} else {
			args = append(args, "--oidc-issuer", DefaultOIDCProviderURL)
		}
	} else {
		// Use default URLs if no options provided
		args = append(args,
			"--fulcio-url", DefaultFulcioURL,
			"--rekor-url", DefaultRekorURL,
			"--timestamp-server-url", DefaultTimestampURL,
			"--oidc-issuer", DefaultOIDCProviderURL,
		)
	}

	// Add the payload file as the last argument
	args = append(args, payloadFile)

	cmd := exec.Command("cosign", args...)

	// Execute the command
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("cosign sign-blob OIDC failed: %w\nOutput: %s", err, string(output))
	}

	// Read the signature
	signature, err := os.ReadFile(signatureFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read signature: %w", err)
	}
	defer os.Remove(signatureFile)

	// Extract public key from certificate
	publicKeyPEM, err := extractPublicKeyFromCertificateFile(certificateFile)
	if err != nil {
		return nil, fmt.Errorf("failed to extract public key from certificate: %w", err)
	}
	defer os.Remove(certificateFile)

	// Remove the bundle file
	// TODO Investigate why sign fails if the bundle file is not added as a parameter
	defer os.Remove(bundleFile)

	// Create the signature object with proper structure
	signatureObj := &signv1.Signature{
		Signature:     string(signature),
		PublicKey:     &publicKeyPEM,
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

	// Create payload temporary file
	payload := cosign.GeneratePayload("localhost:5000", "dir", digest.String())

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Write payload to temporary file
	payloadFile := "payload-temp.json"

	err = os.WriteFile(payloadFile, payloadBytes, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to write payload: %w", err)
	}

	defer os.Remove(payloadFile)

	// Write private key to temporary file
	keyFile := "cosign-temp.key"

	err = os.WriteFile(keyFile, keySigner.GetPrivateKey(), 0o600)
	if err != nil {
		return nil, fmt.Errorf("failed to write private key: %w", err)
	}

	defer os.Remove(keyFile)

	signatureFile := "signature-temp.sig"
	cmd := exec.Command("cosign", "sign-blob",
		"-y",
		"--key", keyFile,
		"--output-signature", signatureFile,
		payloadFile)

	// Set environment variables
	cmd.Env = append(os.Environ(), "COSIGN_PASSWORD="+string(password))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("cosign sign failed: %w\nOutput: %s", err, string(output))
	}

	signature, err := os.ReadFile(signatureFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read signature: %w", err)
	}

	defer os.Remove(signatureFile)

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

// extractPublicKeyFromCertificateFile extracts the public key from a base64-encoded certificate file
func extractPublicKeyFromCertificateFile(certificateFile string) (string, error) {
	// Read the base64-encoded certificate file
	certData, err := os.ReadFile(certificateFile)
	if err != nil {
		return "", fmt.Errorf("failed to read certificate file: %w", err)
	}

	// Clean up the base64 data - remove URL encoding and whitespace
	certDataStr := strings.TrimSpace(string(certData))
	certDataStr = strings.TrimSuffix(certDataStr, "%")      // Remove URL encoding artifacts at the end
	certDataStr = strings.ReplaceAll(certDataStr, "\n", "") // Remove any newlines
	certDataStr = strings.ReplaceAll(certDataStr, "\r", "") // Remove any carriage returns

	// Decode the base64 certificate (this gives us PEM data)
	pemBytes, err := base64.StdEncoding.DecodeString(certDataStr)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 certificate: %w", err)
	}

	// Parse the PEM-encoded certificate
	block, _ := pem.Decode(pemBytes)
	if block == nil || block.Type != "CERTIFICATE" {
		return "", fmt.Errorf("failed to decode PEM certificate")
	}

	// Parse the X.509 certificate from the PEM block
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("failed to parse X.509 certificate: %w", err)
	}

	// Extract the public key
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(cert.PublicKey)
	if err != nil {
		return "", fmt.Errorf("failed to marshal public key: %w", err)
	}

	// Encode the public key as PEM
	pubKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	})

	return string(pubKeyPEM), nil
}
