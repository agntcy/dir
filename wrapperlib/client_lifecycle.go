// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import "C"

import (
	"context"
	"encoding/json"

	"github.com/agntcy/dir/client"
)

// --- NewClient ---

type newClientResponse struct {
	Handle int64  `json:"handle,omitempty"`
	Error  string `json:"error,omitempty"`
}

// NewClient creates a new dir client.Client from a JSON-encoded client.Config
// (server_address, tls_*, spiffe_*, auth_mode, jwt_audience, oidc_*, auth_token
// -- see client.Config for the authoritative field list, all fields are plain
// JSON, not protojson, since Config is not a proto message). It calls
// client.New(context.Background(), client.WithConfig(&cfg)) and, on success,
// registers the resulting client under an opaque handle that must later be
// released with CloseClient.
//
//export NewClient
func NewClient(configJSON *C.char) *C.char {
	var cfg client.Config
	if err := json.Unmarshal([]byte(C.GoString(configJSON)), &cfg); err != nil {
		return toC(newClientResponse{Error: "invalid config: " + err.Error()})
	}

	c, err := client.New(context.Background(), client.WithConfig(&cfg))
	if err != nil {
		return toC(newClientResponse{Error: err.Error()})
	}

	return toC(newClientResponse{Handle: registerClient(c)})
}

// --- CloseClient ---

// CloseClient closes the underlying gRPC connection (and any SPIFFE/auth
// sources) for handle and removes it from the registry. The handle becomes
// invalid for any further calls after this returns, regardless of whether an
// error occurred while closing.
//
//export CloseClient
func CloseClient(handle C.longlong) *C.char {
	c, ok := lookupClient(int64(handle))
	if !ok {
		return toC(errorResponse{Error: "unknown or closed client handle"})
	}

	err := c.Close()
	unregisterClient(int64(handle))

	if err != nil {
		return toC(errorResponse{Error: err.Error()})
	}

	return toC(errorResponse{})
}
