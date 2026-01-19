// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package authn

import (
	"context"
	"crypto/ed25519"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/agntcy/dir/server/healthcheck"
	"github.com/mr-tron/base58"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// DIDContextKey is used to store the DID in context
type didContextKeyType string

const (
	DIDContextKey        didContextKeyType = "did"
	VerificationMethodID didContextKeyType = "verification_method_id"
)

// DIDAuthMessage represents the message structure for DID authentication
type DIDAuthMessage struct {
	Method    string `json:"method"`    // gRPC method being called
	Timestamp int64  `json:"timestamp"` // Unix timestamp
	Nonce     string `json:"nonce"`     // Random nonce for replay protection
}

// DIDAuthPayload contains the parsed authentication data
type DIDAuthPayload struct {
	DID                  string
	VerificationMethodID string
	Message              []byte
	Signature            []byte
}

// DIDResolver interface for resolving DIDs
type DIDResolver interface {
	Resolve(did string) (*DIDDocument, error)
}

// DIDResolutionResult represents the Universal Resolver response
// Based on W3C DID Resolution spec: https://www.w3.org/TR/did-core/#did-resolution
type DIDResolutionResult struct {
	Context               interface{}            `json:"@context,omitempty"`
	DIDDocument           *DIDDocument           `json:"didDocument,omitempty"`
	DIDDocumentMetadata   map[string]interface{} `json:"didDocumentMetadata,omitempty"`
	DIDResolutionMetadata ResolutionMetadata     `json:"didResolutionMetadata,omitempty"`
}

// ResolutionMetadata contains metadata about the resolution process
type ResolutionMetadata struct {
	ContentType string `json:"contentType,omitempty"`
	Error       string `json:"error,omitempty"`
}

// DIDDocument represents a W3C DID Document
type DIDDocument struct {
	Context            interface{}          `json:"@context,omitempty"`
	ID                 string               `json:"id"`
	Controller         interface{}          `json:"controller,omitempty"`
	VerificationMethod []VerificationMethod `json:"verificationMethod,omitempty"`
	Authentication     interface{}          `json:"authentication,omitempty"`
	AssertionMethod    interface{}          `json:"assertionMethod,omitempty"`
	Service            interface{}          `json:"service,omitempty"`
}

// VerificationMethod represents a verification method in a DID Document
type VerificationMethod struct {
	ID                 string                 `json:"id"`
	Type               string                 `json:"type"`
	Controller         string                 `json:"controller"`
	PublicKeyBase58    string                 `json:"publicKeyBase58,omitempty"`
	PublicKeyMultibase string                 `json:"publicKeyMultibase,omitempty"`
	PublicKeyJwk       map[string]interface{} `json:"publicKeyJwk,omitempty"`
}

// UniversalResolver implements DIDResolver using Universal Resolver HTTP API
type UniversalResolver struct {
	endpoint   string
	httpClient *http.Client
}

// NewUniversalResolver creates a new Universal Resolver client
// endpoint examples:
//   - "https://dev.uniresolver.io" (DIF public instance, testing only)
//   - "https://resolver.cheqd.net" (Cheqd's resolver, did:cheqd only)
//   - "http://localhost:8080" (self-hosted instance)
func NewUniversalResolver(endpoint string) (*UniversalResolver, error) {
	return &UniversalResolver{
		endpoint:   strings.TrimSuffix(endpoint, "/"),
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// Resolve resolves a DID using the Universal Resolver HTTP API
// API spec: https://w3c-ccg.github.io/did-resolution/
func (r *UniversalResolver) Resolve(did string) (*DIDDocument, error) {
	// Build URL: /1.0/identifiers/{did}
	url := fmt.Sprintf("%s/1.0/identifiers/%s", r.endpoint, did)

	// Create HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set Accept header for DID Resolution result (includes metadata)
	req.Header.Set("Accept", "application/did+ld+json")

	// Make request
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve DID %s: %w", did, err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("DID resolution failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result DIDResolutionResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse DID resolution result: %w", err)
	}

	// Check for resolution errors
	if result.DIDResolutionMetadata.Error != "" {
		return nil, fmt.Errorf("DID resolution error: %s", result.DIDResolutionMetadata.Error)
	}

	if result.DIDDocument == nil {
		return nil, fmt.Errorf("DID document not found in resolution result")
	}

	return result.DIDDocument, nil
}

// NewDIDInterceptor creates an interceptor that verifies DID signatures
func NewDIDInterceptor(resolver DIDResolver) DIDInterceptorFn {
	return func(ctx context.Context, method string) (context.Context, error) {
		// Extract metadata from gRPC context
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return ctx, status.Error(codes.Unauthenticated, "no metadata provided")
		}

		// Get DID-Auth header
		authHeaders := md.Get("authorization")
		if len(authHeaders) == 0 {
			return ctx, status.Error(codes.Unauthenticated, "no authorization header")
		}

		authHeader := authHeaders[0]
		if !strings.HasPrefix(authHeader, "DID-Auth ") {
			return ctx, status.Error(codes.Unauthenticated, "invalid auth scheme, expected DID-Auth")
		}

		// Parse DID-Auth payload
		payload, err := parseDIDAuthHeader(authHeader)
		if err != nil {
			return ctx, status.Error(codes.Unauthenticated, fmt.Sprintf("invalid payload: %v", err))
		}

		// Verify the message timestamp (prevent replay attacks)
		var authMsg DIDAuthMessage
		if err := json.Unmarshal(payload.Message, &authMsg); err != nil {
			return ctx, status.Error(codes.Unauthenticated, "invalid message format")
		}

		// Check timestamp is within acceptable window (5 minutes)
		now := time.Now().Unix()
		if abs(now-authMsg.Timestamp) > 300 {
			return ctx, status.Error(codes.Unauthenticated, "timestamp expired")
		}

		// Verify method matches
		if authMsg.Method != method {
			return ctx, status.Error(codes.Unauthenticated, "method mismatch")
		}

		// Resolve DID Document to get public key
		didDoc, err := resolver.Resolve(payload.DID)
		if err != nil {
			return ctx, status.Error(codes.Unauthenticated, "failed to resolve DID")
		}

		// Find verification method
		vm, err := findVerificationMethod(didDoc, payload.VerificationMethodID)
		if err != nil {
			return ctx, status.Error(codes.Unauthenticated, "verification method not found")
		}

		// Verify signature using the verification method
		if err := verifySignatureFromVM(vm, payload.Message, payload.Signature); err != nil {
			return ctx, status.Error(codes.Unauthenticated, "signature verification failed")
		}

		// Store DID and verification method in context for authorization
		ctx = context.WithValue(ctx, DIDContextKey, payload.DID)
		ctx = context.WithValue(ctx, VerificationMethodID, payload.VerificationMethodID)

		return ctx, nil
	}
}

// parseDIDAuthHeader parses the DID-Auth header
// Format: "DID-Auth <did>:<verification_method_id>:<base64_message>:<base64_signature>"
func parseDIDAuthHeader(authHeader string) (*DIDAuthPayload, error) {
	authData := strings.TrimPrefix(authHeader, "DID-Auth ")
	parts := strings.Split(authData, ":")

	if len(parts) != 4 {
		return nil, fmt.Errorf("invalid DID-Auth format, expected 4 parts, got %d", len(parts))
	}

	did := parts[0]
	vmID := parts[1]
	messageB64 := parts[2]
	signatureB64 := parts[3]

	// Decode base64 message
	message, err := base64.StdEncoding.DecodeString(messageB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode message: %w", err)
	}

	// Decode base64 signature
	signature, err := base64.StdEncoding.DecodeString(signatureB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode signature: %w", err)
	}

	return &DIDAuthPayload{
		DID:                  did,
		VerificationMethodID: vmID,
		Message:              message,
		Signature:            signature,
	}, nil
}

// findVerificationMethod finds a verification method in the DID Document
func findVerificationMethod(didDoc *DIDDocument, vmID string) (*VerificationMethod, error) {
	// Handle both full DID URLs and fragments
	searchID := vmID
	if !strings.HasPrefix(vmID, "did:") {
		// If it's a fragment, prepend the DID
		searchID = didDoc.ID + vmID
	}

	for _, vm := range didDoc.VerificationMethod {
		if vm.ID == searchID {
			return &vm, nil
		}
	}

	return nil, fmt.Errorf("verification method %s not found in DID document", vmID)
}

// verifySignatureFromVM verifies a signature using a verification method
func verifySignatureFromVM(vm *VerificationMethod, message []byte, signature []byte) error {
	switch vm.Type {
	case "Ed25519VerificationKey2018", "Ed25519VerificationKey2020":
		return verifyEd25519Signature(vm, message, signature)
	case "JsonWebKey2020":
		return verifyJWKSignature(vm, message, signature)
	default:
		return fmt.Errorf("unsupported verification method type: %s", vm.Type)
	}
}

// verifyEd25519Signature verifies an Ed25519 signature
func verifyEd25519Signature(vm *VerificationMethod, message []byte, signature []byte) error {
	var pubKeyBytes []byte
	var err error

	// Extract public key from different formats
	if vm.PublicKeyBase58 != "" {
		// Decode base58 (most common for Ed25519)
		pubKeyBytes, err = base58.Decode(vm.PublicKeyBase58)
		if err != nil {
			return fmt.Errorf("failed to decode publicKeyBase58: %w", err)
		}
	} else if vm.PublicKeyMultibase != "" {
		// Decode multibase (starts with 'z' for base58btc)
		pubKeyBytes, err = decodeMultibase(vm.PublicKeyMultibase)
		if err != nil {
			return fmt.Errorf("failed to decode publicKeyMultibase: %w", err)
		}
	} else if vm.PublicKeyJwk != nil {
		// Extract from JWK
		pubKeyBytes, err = extractEd25519FromJWK(vm.PublicKeyJwk)
		if err != nil {
			return fmt.Errorf("failed to extract key from JWK: %w", err)
		}
	} else {
		return fmt.Errorf("no public key found in verification method")
	}

	// Verify using local Ed25519 verification function
	pubKey := ed25519.PublicKey(pubKeyBytes)
	return VerifyED25519Signature(pubKey, message, signature)
}

// verifyJWKSignature verifies a signature from a JWK verification method
func verifyJWKSignature(vm *VerificationMethod, message []byte, signature []byte) error {
	if vm.PublicKeyJwk == nil {
		return fmt.Errorf("publicKeyJwk not found")
	}

	// Determine key type from JWK
	kty, ok := vm.PublicKeyJwk["kty"].(string)
	if !ok {
		return fmt.Errorf("kty field not found in JWK")
	}

	switch kty {
	case "OKP":
		// Ed25519 in JWK format
		pubKeyBytes, err := extractEd25519FromJWK(vm.PublicKeyJwk)
		if err != nil {
			return err
		}
		pubKey := ed25519.PublicKey(pubKeyBytes)
		return VerifyED25519Signature(pubKey, message, signature)

	case "RSA":
		// RSA in JWK format
		pubKey, err := extractRSAFromJWK(vm.PublicKeyJwk)
		if err != nil {
			return err
		}
		return VerifyRSASignature(*pubKey, message, signature)

	default:
		return fmt.Errorf("unsupported JWK key type: %s", kty)
	}
}

// Helper functions for key decoding

func decodeMultibase(s string) ([]byte, error) {
	// Multibase format: first character indicates encoding
	// 'z' = base58btc
	if len(s) == 0 {
		return nil, fmt.Errorf("empty multibase string")
	}

	prefix := s[0]
	encoded := s[1:]

	switch prefix {
	case 'z': // base58btc
		return base58.Decode(encoded)
	default:
		return nil, fmt.Errorf("unsupported multibase encoding: %c", prefix)
	}
}

func extractEd25519FromJWK(jwk map[string]interface{}) ([]byte, error) {
	// Ed25519 JWK format:
	// {"kty": "OKP", "crv": "Ed25519", "x": "<base64url>"}

	crv, ok := jwk["crv"].(string)
	if !ok || crv != "Ed25519" {
		return nil, fmt.Errorf("invalid or missing crv field for Ed25519")
	}

	xStr, ok := jwk["x"].(string)
	if !ok {
		return nil, fmt.Errorf("x coordinate not found in JWK")
	}

	// Decode base64url
	return base64.RawURLEncoding.DecodeString(xStr)
}

func extractRSAFromJWK(jwk map[string]interface{}) (*rsa.PublicKey, error) {
	// RSA JWK format:
	// {"kty": "RSA", "n": "<base64url>", "e": "<base64url>"}

	nStr, ok := jwk["n"].(string)
	if !ok {
		return nil, fmt.Errorf("n (modulus) not found in RSA JWK")
	}

	eStr, ok := jwk["e"].(string)
	if !ok {
		return nil, fmt.Errorf("e (exponent) not found in RSA JWK")
	}

	// Decode base64url
	nBytes, err := base64.RawURLEncoding.DecodeString(nStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode modulus: %w", err)
	}

	eBytes, err := base64.RawURLEncoding.DecodeString(eStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode exponent: %w", err)
	}

	// Convert to big.Int
	n := new(big.Int).SetBytes(nBytes)

	// Exponent is typically small (commonly 65537)
	var e int
	for i := 0; i < len(eBytes); i++ {
		e = e*256 + int(eBytes[i])
	}

	return &rsa.PublicKey{
		N: n,
		E: e,
	}, nil
}

// Helper function for absolute value
func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

type DIDInterceptorFn func(ctx context.Context, method string) (context.Context, error)

// didUnaryInterceptorFor wraps the DID interceptor function for unary RPCs
func didUnaryInterceptorFor(fn DIDInterceptorFn) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		// Skip authentication for health check endpoints
		if healthcheck.IsHealthCheckEndpoint(info.FullMethod) {
			return handler(ctx, req)
		}

		newCtx, err := fn(ctx, info.FullMethod)
		if err != nil {
			return nil, err
		}

		return handler(newCtx, req)
	}
}

// didStreamInterceptorFor wraps the DID interceptor function for stream RPCs
func didStreamInterceptorFor(fn DIDInterceptorFn) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Skip authentication for health check endpoints
		if healthcheck.IsHealthCheckEndpoint(info.FullMethod) {
			return handler(srv, ss)
		}

		newCtx, err := fn(ss.Context(), info.FullMethod)
		if err != nil {
			return err
		}

		// Create a wrapped stream with the new context
		wrappedStream := &wrappedServerStream{
			ServerStream: ss,
			ctx:          newCtx,
		}

		return handler(srv, wrappedStream)
	}
}

// Public interceptor functions
func DIDUnaryInterceptor(resolver DIDResolver) grpc.UnaryServerInterceptor {
	return didUnaryInterceptorFor(NewDIDInterceptor(resolver))
}

func DIDStreamInterceptor(resolver DIDResolver) grpc.StreamServerInterceptor {
	return didStreamInterceptorFor(NewDIDInterceptor(resolver))
}

// Helper to extract DID from context (for authorization)
func DIDFromContext(ctx context.Context) (string, bool) {
	did, ok := ctx.Value(DIDContextKey).(string)
	return did, ok
}

// Helper to extract verification method ID from context
func VerificationMethodFromContext(ctx context.Context) (string, bool) {
	vmID, ok := ctx.Value(VerificationMethodID).(string)
	return vmID, ok
}
