// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import "C"

import (
	"context"
	"encoding/json"

	corev1 "github.com/agntcy/dir/api/core/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
	"github.com/agntcy/dir/client/utils/cosign"
)

// signatureJSON is a flat JSON view of signv1.Signature, matching the style
// already used by the pre-existing SignWithKey/VerifyWithKey pair.
type signatureJSON struct {
	SignedAt      string `json:"signed_at,omitempty"`
	Algorithm     string `json:"algorithm,omitempty"`
	Signature     string `json:"signature,omitempty"`
	Certificate   string `json:"certificate,omitempty"`
	ContentType   string `json:"content_type,omitempty"`
	ContentBundle string `json:"content_bundle,omitempty"`
}

func (s *signatureJSON) toProto() *signv1.Signature {
	if s == nil {
		return &signv1.Signature{}
	}

	return &signv1.Signature{
		SignedAt:      s.SignedAt,
		Algorithm:     s.Algorithm,
		Signature:     s.Signature,
		Certificate:   s.Certificate,
		ContentType:   s.ContentType,
		ContentBundle: s.ContentBundle,
	}
}

func signatureFromProto(sig *signv1.Signature) *signatureJSON {
	return &signatureJSON{
		SignedAt:      sig.GetSignedAt(),
		Algorithm:     sig.GetAlgorithm(),
		Signature:     sig.GetSignature(),
		Certificate:   sig.GetCertificate(),
		ContentType:   sig.GetContentType(),
		ContentBundle: sig.GetContentBundle(),
	}
}

// --- VerifyWithKey (local cosign, no handle) ---

type verifyWithKeyRequest struct {
	CID        string        `json:"cid"`
	PublicKeys []string      `json:"public_keys"`
	Signature  signatureJSON `json:"signature"`
}

type verifyWithKeyResponse struct {
	Verified  bool   `json:"verified"`
	PublicKey string `json:"public_key,omitempty"`
	Algorithm string `json:"algorithm,omitempty"`
	Error     string `json:"error,omitempty"`
}

// VerifyWithKey verifies a payload (identified by its CID) against one or
// more local public keys, without requiring a client handle.
//
//export VerifyWithKey
func VerifyWithKey(reqJSON *C.char) *C.char {
	var req verifyWithKeyRequest
	if err := json.Unmarshal([]byte(C.GoString(reqJSON)), &req); err != nil {
		return toC(verifyWithKeyResponse{Error: "invalid request: " + err.Error()})
	}

	info, err := cosign.VerifyWithKeys(context.Background(), []byte(req.CID), req.PublicKeys, req.Signature.toProto())
	if err != nil {
		return toC(verifyWithKeyResponse{Verified: false, Error: err.Error()})
	}

	key := info.GetKey()

	return toC(verifyWithKeyResponse{
		Verified:  true,
		PublicKey: key.GetPublicKey(),
		Algorithm: key.GetAlgorithm(),
	})
}

// --- VerifyWithOIDC (local cosign, no handle) ---

// verifyWithOIDCRequest carries the CID (payload) and the signature to check
// as plain/flat fields, alongside the fields of signv1.VerifyWithOIDC
// (issuer, subject, options), which are parsed separately via protojson.
type verifyWithOIDCRequest struct {
	CID       string        `json:"cid"`
	Signature signatureJSON `json:"signature"`
}

type verifyWithOIDCResponse struct {
	Verified   bool            `json:"verified"`
	SignerInfo json.RawMessage `json:"signer_info,omitempty"`
	Error      string          `json:"error,omitempty"`
}

// VerifyWithOIDC verifies a payload (identified by its CID) against a
// Sigstore/OIDC bundle, without requiring a client handle. The request JSON
// carries "cid" and "signature" plus the protojson fields of
// signv1.VerifyWithOIDC ("issuer", "subject", "options").
//
//export VerifyWithOIDC
func VerifyWithOIDC(reqJSON *C.char) *C.char {
	raw := C.GoString(reqJSON)

	var envelope verifyWithOIDCRequest
	if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
		return toC(verifyWithOIDCResponse{Error: "invalid request: " + err.Error()})
	}

	var oidcReq signv1.VerifyWithOIDC
	if err := unmarshalProto(raw, &oidcReq); err != nil {
		return toC(verifyWithOIDCResponse{Error: "invalid request: " + err.Error()})
	}

	signerInfo, err := cosign.VerifyWithOIDC([]byte(envelope.CID), &oidcReq, envelope.Signature.toProto())
	if err != nil {
		return toC(verifyWithOIDCResponse{Verified: false, Error: err.Error()})
	}

	raw2, err := protoToRaw(signerInfo)
	if err != nil {
		return toC(verifyWithOIDCResponse{Error: err.Error()})
	}

	return toC(verifyWithOIDCResponse{Verified: true, SignerInfo: raw2})
}

// --- Verify (client method, requires handle) ---

type verifyResponse struct {
	Response json.RawMessage `json:"response,omitempty"`
	Error    string          `json:"error,omitempty"`
}

// Verify verifies signatures for a record given a protojson-encoded
// signv1.VerifyRequest. When from_server is true, the result is the server's
// cached verification; when false, verification is performed locally.
// Returns a protojson-encoded signv1.VerifyResponse.
//
//export Verify
func Verify(handle C.longlong, reqJSON *C.char) *C.char {
	c, ok := lookupClient(int64(handle))
	if !ok {
		return toC(verifyResponse{Error: "unknown or closed client handle"})
	}

	var req signv1.VerifyRequest
	if err := unmarshalProto(C.GoString(reqJSON), &req); err != nil {
		return toC(verifyResponse{Error: "invalid request: " + err.Error()})
	}

	resp, err := c.Verify(context.Background(), &req)
	if err != nil {
		return toC(verifyResponse{Error: err.Error()})
	}

	raw, err := protoToRaw(resp)
	if err != nil {
		return toC(verifyResponse{Error: err.Error()})
	}

	return toC(verifyResponse{Response: raw})
}

// --- PullSignatures ---

type recordRefRequest struct {
	RecordRef json.RawMessage `json:"record_ref"`
}

// PullSignatures fetches all signature referrers for a record given a
// protojson-encoded corev1.RecordRef (request field "record_ref") and
// returns an arrayResponse whose Results are protojson-encoded
// signv1.Signature values.
//
//export PullSignatures
func PullSignatures(handle C.longlong, reqJSON *C.char) *C.char {
	c, ok := lookupClient(int64(handle))
	if !ok {
		return toC(arrayResponse{Error: "unknown or closed client handle"})
	}

	var req recordRefRequest
	if err := json.Unmarshal([]byte(C.GoString(reqJSON)), &req); err != nil {
		return toC(arrayResponse{Error: "invalid request: " + err.Error()})
	}

	var ref corev1.RecordRef
	if err := unmarshalProto(string(req.RecordRef), &ref); err != nil {
		return toC(arrayResponse{Error: "invalid request: " + err.Error()})
	}

	signatures, err := c.PullSignatures(context.Background(), &ref)
	if err != nil {
		return toC(arrayResponse{Error: err.Error()})
	}

	results, err := marshalProtoSlice(signatures)
	if err != nil {
		return toC(arrayResponse{Error: err.Error()})
	}

	return toC(arrayResponse{Results: results})
}

// --- PullPublicKeys ---

type publicKeysResponse struct {
	PublicKeys []string `json:"public_keys,omitempty"`
	Error      string   `json:"error,omitempty"`
}

// PullPublicKeys fetches all public key referrers for a record given a
// protojson-encoded corev1.RecordRef (request field "record_ref"). Public
// keys are returned as plain PEM strings (PublicKey is not itself a proto
// message once unwrapped from its referrer envelope).
//
//export PullPublicKeys
func PullPublicKeys(handle C.longlong, reqJSON *C.char) *C.char {
	c, ok := lookupClient(int64(handle))
	if !ok {
		return toC(publicKeysResponse{Error: "unknown or closed client handle"})
	}

	var req recordRefRequest
	if err := json.Unmarshal([]byte(C.GoString(reqJSON)), &req); err != nil {
		return toC(publicKeysResponse{Error: "invalid request: " + err.Error()})
	}

	var ref corev1.RecordRef
	if err := unmarshalProto(string(req.RecordRef), &ref); err != nil {
		return toC(publicKeysResponse{Error: "invalid request: " + err.Error()})
	}

	publicKeys, err := c.PullPublicKeys(context.Background(), &ref)
	if err != nil {
		return toC(publicKeysResponse{Error: err.Error()})
	}

	return toC(publicKeysResponse{PublicKeys: publicKeys})
}
