// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package cosign

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
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

// SetOrDefault returns the value if it's not empty, otherwise returns the default value.
func SetOrDefault(value string, defaultValue string) string {
	if value == "" {
		value = defaultValue
	}

	return value
}

// ExtractPublicKeyFromCertificateFile extracts the public key from a base64-encoded certificate file.
func ExtractPublicKeyFromCertificateFile(certificateFile string) (string, error) {
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
		return "", errors.New("failed to decode PEM certificate")
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

// SignBlobOIDCOptions contains options for OIDC-based blob signing.
type SignBlobOIDCOptions struct {
	Payload         []byte
	IDToken         string
	FulcioURL       string
	RekorURL        string
	TimestampURL    string
	OIDCProviderURL string
	OIDCClientID    string
}

// SignBlobOIDCResult contains the result of OIDC blob signing.
type SignBlobOIDCResult struct {
	Signature string
	PublicKey string
}

// SignBlobWithOIDC signs a blob using OIDC authentication.
func SignBlobWithOIDC(ctx context.Context, opts *SignBlobOIDCOptions) (*SignBlobOIDCResult, error) {
	// Create temporary files
	payloadFile, err := os.CreateTemp("", "payload-oidc-*.json")
	if err != nil {
		return nil, fmt.Errorf("failed to create payload temp file: %w", err)
	}
	defer os.Remove(payloadFile.Name())

	signatureFile, err := os.CreateTemp("", "signature-oidc-*.sig")
	if err != nil {
		return nil, fmt.Errorf("failed to create signature temp file: %w", err)
	}
	defer os.Remove(signatureFile.Name())

	certificateFile, err := os.CreateTemp("", "certificate-oidc-*.crt")
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate temp file: %w", err)
	}
	defer os.Remove(certificateFile.Name())

	bundleFile, err := os.CreateTemp("", "bundle-oidc-*.json")
	if err != nil {
		return nil, fmt.Errorf("failed to create bundle temp file: %w", err)
	}
	defer os.Remove(bundleFile.Name())

	// Write payload to file
	if _, err := payloadFile.Write(opts.Payload); err != nil {
		return nil, fmt.Errorf("failed to write payload: %w", err)
	}

	payloadFile.Close()

	// Build cosign command arguments
	args := []string{
		"sign-blob",
		"-y",
		"--output-signature", signatureFile.Name(),
		"--output-certificate", certificateFile.Name(),
		"--bundle", bundleFile.Name(),
		"--new-bundle-format",
		"--insecure-skip-verify",
	}

	// Add identity token if provided
	if opts.IDToken != "" {
		args = append(args, "--identity-token", opts.IDToken)
	} else {
		// For interactive OIDC flow
		clientID := SetOrDefault(opts.OIDCClientID, DefaultOIDCClientID)
		args = append(args, "--oidc-client-id", clientID)
	}

	// Add URLs with defaults
	fulcioURL := SetOrDefault(opts.FulcioURL, DefaultFulcioURL)
	rekorURL := SetOrDefault(opts.RekorURL, DefaultRekorURL)
	timestampURL := SetOrDefault(opts.TimestampURL, DefaultTimestampURL)
	oidcProviderURL := SetOrDefault(opts.OIDCProviderURL, DefaultOIDCProviderURL)

	args = append(args,
		"--fulcio-url", fulcioURL,
		"--rekor-url", rekorURL,
		"--timestamp-server-url", timestampURL,
		"--oidc-issuer", oidcProviderURL,
	)

	// Add payload file as last argument
	args = append(args, payloadFile.Name())

	// Execute command
	cmd := exec.CommandContext(ctx, "cosign", args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("cosign sign-blob OIDC failed: %w\nOutput: %s", err, string(output))
	}

	// Read signature
	signature, err := os.ReadFile(signatureFile.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to read signature: %w", err)
	}

	// Extract public key from certificate
	publicKeyPEM, err := ExtractPublicKeyFromCertificateFile(certificateFile.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to extract public key from certificate: %w", err)
	}

	return &SignBlobOIDCResult{
		Signature: string(signature),
		PublicKey: publicKeyPEM,
	}, nil
}

// SignBlobKeyOptions contains options for key-based blob signing.
type SignBlobKeyOptions struct {
	Payload    []byte
	PrivateKey []byte
	Password   []byte
}

// SignBlobKeyResult contains the result of key-based blob signing.
type SignBlobKeyResult struct {
	Signature string
	PublicKey string
}

// SignBlobWithKey signs a blob using a private key.
//
//nolint:mnd,gosec
func SignBlobWithKey(ctx context.Context, opts *SignBlobKeyOptions) (*SignBlobKeyResult, error) {
	// Create temporary files
	payloadFile, err := os.CreateTemp("", "payload-key-*.json")
	if err != nil {
		return nil, fmt.Errorf("failed to create payload temp file: %w", err)
	}
	defer os.Remove(payloadFile.Name())

	keyFile, err := os.CreateTemp("", "cosign-key-*.key")
	if err != nil {
		return nil, fmt.Errorf("failed to create key temp file: %w", err)
	}
	defer os.Remove(keyFile.Name())

	signatureFile, err := os.CreateTemp("", "signature-key-*.sig")
	if err != nil {
		return nil, fmt.Errorf("failed to create signature temp file: %w", err)
	}
	defer os.Remove(signatureFile.Name())

	// Write payload and private key to files
	if _, err := payloadFile.Write(opts.Payload); err != nil {
		return nil, fmt.Errorf("failed to write payload: %w", err)
	}

	payloadFile.Close()

	if err := os.WriteFile(keyFile.Name(), opts.PrivateKey, 0o600); err != nil {
		return nil, fmt.Errorf("failed to write private key: %w", err)
	}

	// Build cosign command
	cmd := exec.CommandContext(ctx, "cosign", "sign-blob",
		"-y",
		"--key", keyFile.Name(),
		"--output-signature", signatureFile.Name(),
		payloadFile.Name())

	// Set password environment variable
	password := opts.Password
	if password == nil {
		password = []byte("")
	}

	cmd.Env = append(os.Environ(), "COSIGN_PASSWORD="+string(password))

	// Execute command
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("cosign sign-blob with key failed: %w\nOutput: %s", err, string(output))
	}

	// Read signature
	signature, err := os.ReadFile(signatureFile.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to read signature: %w", err)
	}

	// Load keypair to get public key
	cosignKeypair, err := LoadKeypair(opts.PrivateKey, password)
	if err != nil {
		return nil, fmt.Errorf("failed to load cosign keypair: %w", err)
	}

	publicKeyPEM, err := cosignKeypair.GetPublicKeyPem()
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %w", err)
	}

	return &SignBlobKeyResult{
		Signature: string(signature),
		PublicKey: publicKeyPEM,
	}, nil
}

// AttachSignatureOptions contains options for attaching signatures to OCI images.
type AttachSignatureOptions struct {
	ImageRef  string
	Signature string
	Payload   string
}

// AttachSignature attaches a signature to an OCI image using cosign.
//
//nolint:mnd,gosec
func AttachSignature(ctx context.Context, opts *AttachSignatureOptions) error {
	// Create temporary files
	signatureFile, err := os.CreateTemp("", "attach-signature-*.sig")
	if err != nil {
		return fmt.Errorf("failed to create signature temp file: %w", err)
	}
	defer os.Remove(signatureFile.Name())

	payloadFile, err := os.CreateTemp("", "attach-payload-*.json")
	if err != nil {
		return fmt.Errorf("failed to create payload temp file: %w", err)
	}
	defer os.Remove(payloadFile.Name())

	// Write signature and payload to files
	if err := os.WriteFile(signatureFile.Name(), []byte(opts.Signature), 0o644); err != nil {
		return fmt.Errorf("failed to write signature file: %w", err)
	}

	if err := os.WriteFile(payloadFile.Name(), []byte(opts.Payload), 0o644); err != nil {
		return fmt.Errorf("failed to write payload file: %w", err)
	}

	// Build cosign attach command
	args := []string{
		"attach", "signature",
		"--signature", signatureFile.Name(),
		"--payload", payloadFile.Name(),
		opts.ImageRef,
	}

	// Execute command
	cmd := exec.CommandContext(ctx, "cosign", args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("cosign attach signature failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// GenerateKeyPairOptions contains options for generating cosign key pairs.
type GenerateKeyPairOptions struct {
	Directory string
	Password  string
}

// GenerateKeyPair generates a cosign key pair in the specified directory.
func GenerateKeyPair(ctx context.Context, opts *GenerateKeyPairOptions) error {
	cmd := exec.CommandContext(ctx, "cosign", "generate-key-pair")

	if opts.Directory != "" {
		cmd.Dir = opts.Directory
	}

	if opts.Password != "" {
		cmd.Env = append(os.Environ(), "COSIGN_PASSWORD="+opts.Password)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("cosign generate-key-pair failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}
