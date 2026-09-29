// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package main exposes the dir client SDK (github.com/agntcy/dir/client) as
// a C-shared library, so non-Go SDKs (Python, JS/TS) can bind to it directly
// instead of shelling out to the dirctl binary or a Docker image.
//
// The surface covers the full dir/client.Client API -- store (push/pull/
// lookup/delete, including referrers), routing (publish/unpublish/list/
// search), search (CIDs/records), runtime discovery (workloads), events
// (listen), naming (verification info/resolve), sync jobs, and signing/
// verification -- plus the local Sigstore/cosign signing and verification
// helpers from client/utils/cosign that do not require a live server
// connection (SignWithKey, VerifyWithKey, SignWithOIDC, VerifyWithOIDC).
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
