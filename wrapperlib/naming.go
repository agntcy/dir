// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import "C"

import (
	"context"
	"encoding/json"
)

// --- GetVerificationInfo ---

type getVerificationInfoRequest struct {
	CID string `json:"cid"`
}

type verificationInfoResponse struct {
	Info  json.RawMessage `json:"info,omitempty"`
	Error string          `json:"error,omitempty"`
}

// GetVerificationInfo retrieves the verification info for a record by CID.
//
//export GetVerificationInfo
func GetVerificationInfo(handle C.longlong, reqJSON *C.char) *C.char {
	c, ok := lookupClient(int64(handle))
	if !ok {
		return toC(verificationInfoResponse{Error: "unknown or closed client handle"})
	}

	var req getVerificationInfoRequest
	if err := json.Unmarshal([]byte(C.GoString(reqJSON)), &req); err != nil {
		return toC(verificationInfoResponse{Error: "invalid request: " + err.Error()})
	}

	resp, err := c.GetVerificationInfo(context.Background(), req.CID)
	if err != nil {
		return toC(verificationInfoResponse{Error: err.Error()})
	}

	raw, err := protoToRaw(resp)
	if err != nil {
		return toC(verificationInfoResponse{Error: err.Error()})
	}

	return toC(verificationInfoResponse{Info: raw})
}

// --- GetVerificationInfoByName ---

type getVerificationInfoByNameRequest struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// GetVerificationInfoByName retrieves the verification info for a record by
// name. If version is empty, the latest version is used.
//
//export GetVerificationInfoByName
func GetVerificationInfoByName(handle C.longlong, reqJSON *C.char) *C.char {
	c, ok := lookupClient(int64(handle))
	if !ok {
		return toC(verificationInfoResponse{Error: "unknown or closed client handle"})
	}

	var req getVerificationInfoByNameRequest
	if err := json.Unmarshal([]byte(C.GoString(reqJSON)), &req); err != nil {
		return toC(verificationInfoResponse{Error: "invalid request: " + err.Error()})
	}

	resp, err := c.GetVerificationInfoByName(context.Background(), req.Name, req.Version)
	if err != nil {
		return toC(verificationInfoResponse{Error: err.Error()})
	}

	raw, err := protoToRaw(resp)
	if err != nil {
		return toC(verificationInfoResponse{Error: err.Error()})
	}

	return toC(verificationInfoResponse{Info: raw})
}

// --- Resolve ---

type resolveRequest struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

type resolveResponse struct {
	Result json.RawMessage `json:"result,omitempty"`
	Error  string          `json:"error,omitempty"`
}

// Resolve resolves a record reference (name with optional version) to CIDs.
// If version is empty, returns all versions; otherwise returns matches for
// the specific version.
//
//export Resolve
func Resolve(handle C.longlong, reqJSON *C.char) *C.char {
	c, ok := lookupClient(int64(handle))
	if !ok {
		return toC(resolveResponse{Error: "unknown or closed client handle"})
	}

	var req resolveRequest
	if err := json.Unmarshal([]byte(C.GoString(reqJSON)), &req); err != nil {
		return toC(resolveResponse{Error: "invalid request: " + err.Error()})
	}

	resp, err := c.Resolve(context.Background(), req.Name, req.Version)
	if err != nil {
		return toC(resolveResponse{Error: err.Error()})
	}

	raw, err := protoToRaw(resp)
	if err != nil {
		return toC(resolveResponse{Error: err.Error()})
	}

	return toC(resolveResponse{Result: raw})
}
