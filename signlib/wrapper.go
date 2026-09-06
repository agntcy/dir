// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package main exposes the Sigstore/cosign signing and verification logic
// used by dirctl as a C-shared library, so non-Go SDKs (Python, JS/TS) can
// bind to it directly instead of shelling out to the dirctl binary or a
// Docker image.
//
// The C ABI is intentionally JSON-in/JSON-out: every exported function takes
// a single *C.char holding a JSON request and returns a *C.char holding a
// JSON response. Callers must free every returned string with FreeCString.
package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"context"
	"encoding/json"
	"unsafe"

	signv1 "github.com/agntcy/dir/api/sign/v1"
	"github.com/agntcy/dir/client/utils/cosign"
)

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

// --- SignWithKey ---

type signWithKeyRequest struct {
	CID        string `json:"cid"`
	PrivateKey string `json:"private_key"`
	Password   string `json:"password,omitempty"`
}

type signWithKeyResponse struct {
	Signature *signatureJSON `json:"signature,omitempty"`
	PublicKey string         `json:"public_key,omitempty"`
	Error     string         `json:"error,omitempty"`
}

//export SignWithKey
func SignWithKey(reqJSON *C.char) *C.char {
	var req signWithKeyRequest
	if err := json.Unmarshal([]byte(C.GoString(reqJSON)), &req); err != nil {
		return toC(signWithKeyResponse{Error: "invalid request: " + err.Error()})
	}

	sig, pub, err := cosign.SignBlobWithKey(context.Background(), []byte(req.CID), &signv1.SignWithKey{
		PrivateKey: req.PrivateKey,
		Password:   []byte(req.Password),
	})
	if err != nil {
		return toC(signWithKeyResponse{Error: err.Error()})
	}

	return toC(signWithKeyResponse{
		Signature: signatureFromProto(sig),
		PublicKey: pub.GetKey(),
	})
}

// --- VerifyWithKey ---

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

// FreeCString releases a string previously returned by an exported function.
// Callers MUST call this on every returned pointer to avoid leaking memory
// across the cgo boundary.
//
//export FreeCString
func FreeCString(p *C.char) {
	C.free(unsafe.Pointer(p))
}

func toC(v any) *C.char {
	b, err := json.Marshal(v)
	if err != nil {
		return C.CString(`{"error":"failed to marshal response"}`)
	}

	return C.CString(string(b))
}

func main() {}
