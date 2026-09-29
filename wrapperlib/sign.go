// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import "C"

import (
	"context"
	"encoding/json"

	signv1 "github.com/agntcy/dir/api/sign/v1"
	"github.com/agntcy/dir/client/utils/cosign"
)

// --- SignWithKey (local cosign, no handle) ---

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

// SignWithKey signs a payload (identified by its CID) using a local private
// key, without requiring a client handle.
//
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

// --- SignWithOIDC (local cosign, no handle) ---

// signWithOIDCRequest carries the CID (payload) as a plain field alongside
// the fields of signv1.SignWithOIDC (id_token, options), which are parsed
// separately via protojson since SignWithOIDC is a proto message.
type signWithOIDCRequest struct {
	CID string `json:"cid"`
}

type signWithOIDCResponse struct {
	Signature *signatureJSON `json:"signature,omitempty"`
	PublicKey string         `json:"public_key,omitempty"`
	Error     string         `json:"error,omitempty"`
}

// SignWithOIDC signs a payload (identified by its CID) using OIDC-based
// keyless signing, without requiring a client handle. The request JSON
// carries "cid" plus the protojson fields of signv1.SignWithOIDC
// ("id_token", "options").
//
//export SignWithOIDC
func SignWithOIDC(reqJSON *C.char) *C.char {
	raw := C.GoString(reqJSON)

	var envelope signWithOIDCRequest
	if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
		return toC(signWithOIDCResponse{Error: "invalid request: " + err.Error()})
	}

	var oidcReq signv1.SignWithOIDC
	if err := unmarshalProto(raw, &oidcReq); err != nil {
		return toC(signWithOIDCResponse{Error: "invalid request: " + err.Error()})
	}

	sig, pub, err := cosign.SignBlobWithOIDC(context.Background(), []byte(envelope.CID), &oidcReq)
	if err != nil {
		return toC(signWithOIDCResponse{Error: err.Error()})
	}

	return toC(signWithOIDCResponse{
		Signature: signatureFromProto(sig),
		PublicKey: pub.GetKey(),
	})
}

// --- Sign (client method, requires handle) ---

type signResponse struct {
	Response json.RawMessage `json:"response,omitempty"`
	Error    string          `json:"error,omitempty"`
}

// Sign routes to the appropriate signing method (Key or OIDC) based on the
// provider set in the protojson-encoded signv1.SignRequest, then pushes the
// resulting signature and public key to the store as referrers. Returns a
// protojson-encoded signv1.SignResponse.
//
//export Sign
func Sign(handle C.longlong, reqJSON *C.char) *C.char {
	c, ok := lookupClient(int64(handle))
	if !ok {
		return toC(signResponse{Error: "unknown or closed client handle"})
	}

	var req signv1.SignRequest
	if err := unmarshalProto(C.GoString(reqJSON), &req); err != nil {
		return toC(signResponse{Error: "invalid request: " + err.Error()})
	}

	resp, err := c.Sign(context.Background(), &req)
	if err != nil {
		return toC(signResponse{Error: err.Error()})
	}

	raw, err := protoToRaw(resp)
	if err != nil {
		return toC(signResponse{Error: err.Error()})
	}

	return toC(signResponse{Response: raw})
}
