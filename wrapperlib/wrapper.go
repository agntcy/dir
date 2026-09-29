// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package main exposes the dir client SDK's signing and verification logic
// (github.com/agntcy/dir/client, github.com/agntcy/dir/client/utils/cosign)
// as a C-shared library, so non-Go SDKs (Python, JS/TS) can bind to it
// directly instead of shelling out to the dirctl binary or a Docker image.
//
// Every dir gRPC service (store, routing, search, runtime discovery, events,
// naming, sync) already has a .proto definition that any language can
// generate a native client stub from, so this library does not wrap those
// 1:1 passthrough RPCs -- doing so would just be a second, redundant client
// for the same wire call. What this library DOES provide is the logic that
// is not just a single RPC:
//
//   - SignWithKey, SignWithOIDC, VerifyWithKey, VerifyWithOIDC: local
//     Sigstore/cosign cryptography (key loading, ephemeral keypairs, bundle
//     signing/verification) that only exists as Go code today and cannot be
//     reproduced by calling a dir server RPC.
//   - Sign, Verify: orchestrate local signing/verification with the store
//     (pushing/looking up referrers, or dispatching to the server's cached
//     verification) -- more than a single RPC.
//   - PullSignatures, PullPublicKeys: fetch referrers and decode them by
//     referrer type (signature vs. public key) out of the OCI-referrer
//     envelope -- logic beyond a raw PullReferrer call.
//
// The C ABI is intentionally JSON-in/JSON-out: every exported function takes
// one or two *C.char arguments (an optional client handle argument comes
// first as a C.longlong, followed by a *C.char holding a JSON request) and
// returns a *C.char holding a JSON response. Callers must free every
// returned string with FreeCString.
//
// Requests/responses that correspond to a protobuf message are encoded with
// protojson (see helpers.go); requests/responses built from plain scalar
// arguments use a small local JSON struct instead. See handle.go and
// client_lifecycle.go for the client handle registry that backs every
// exported function requiring a live *client.Client (created via NewClient,
// released via CloseClient).
package main

/*
#include <stdlib.h>
*/
import "C"

import "unsafe"

// FreeCString releases a string previously returned by an exported function.
// Callers MUST call this on every returned pointer to avoid leaking memory
// across the cgo boundary.
//
//export FreeCString
func FreeCString(p *C.char) {
	C.free(unsafe.Pointer(p))
}

func main() {}
