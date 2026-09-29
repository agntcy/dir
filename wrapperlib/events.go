// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import "C"

import (
	"context"

	eventsv1 "github.com/agntcy/dir/api/events/v1"
)

// ListenStream sends a protojson-encoded eventsv1.ListenRequest and drains
// the resulting stream into an arrayResponse whose Results are
// protojson-encoded eventsv1.ListenResponse values.
//
// NOTE: the server-side Listen RPC can, depending on the request filters,
// stream indefinitely (e.g. an unfiltered "listen forever" subscription).
// Because this wrapper drains the whole stream into an in-memory array before
// returning across the C ABI, it only makes sense for requests that the
// server is expected to eventually close on its own (bounded time ranges,
// finite backlogs, etc). There is no cancellation support across the C ABI in
// this pass, so an unbounded request will block the calling thread
// indefinitely -- callers wanting a live, unbounded event feed should not use
// this wrapper as-is.
//
//export ListenStream
func ListenStream(handle C.longlong, reqJSON *C.char) *C.char {
	c, ok := lookupClient(int64(handle))
	if !ok {
		return toC(arrayResponse{Error: "unknown or closed client handle"})
	}

	var req eventsv1.ListenRequest
	if err := unmarshalProto(C.GoString(reqJSON), &req); err != nil {
		return toC(arrayResponse{Error: "invalid request: " + err.Error()})
	}

	result, err := c.ListenStream(context.Background(), &req)
	if err != nil {
		return toC(arrayResponse{Error: err.Error()})
	}

	items, err := drainStream(result)
	if err != nil {
		return toC(arrayResponse{Error: err.Error()})
	}

	results, err := marshalProtoSlice(items)
	if err != nil {
		return toC(arrayResponse{Error: err.Error()})
	}

	return toC(arrayResponse{Results: results})
}
