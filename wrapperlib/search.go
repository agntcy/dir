// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import "C"

import (
	"context"
	"errors"

	searchv1 "github.com/agntcy/dir/api/search/v1"
)

// --- SearchCIDs ---

// SearchCIDs sends a protojson-encoded searchv1.SearchCIDsRequest and drains
// the resulting stream into an arrayResponse whose Results are
// protojson-encoded searchv1.SearchCIDsResponse values.
//
//export SearchCIDs
func SearchCIDs(handle C.longlong, reqJSON *C.char) *C.char {
	c, ok := lookupClient(int64(handle))
	if !ok {
		return toC(arrayResponse{Error: "unknown or closed client handle"})
	}

	var req searchv1.SearchCIDsRequest
	if err := unmarshalProto(C.GoString(reqJSON), &req); err != nil {
		return toC(arrayResponse{Error: "invalid request: " + err.Error()})
	}

	result, err := c.SearchCIDs(context.Background(), &req)
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

// --- SearchRecords ---

// SearchRecords sends a protojson-encoded searchv1.SearchRecordsRequest and
// drains the resulting stream into an arrayResponse whose Results are
// protojson-encoded searchv1.SearchRecordsResponse values.
//
//export SearchRecords
func SearchRecords(handle C.longlong, reqJSON *C.char) *C.char {
	c, ok := lookupClient(int64(handle))
	if !ok {
		return toC(arrayResponse{Error: "unknown or closed client handle"})
	}

	var req searchv1.SearchRecordsRequest
	if err := unmarshalProto(C.GoString(reqJSON), &req); err != nil {
		return toC(arrayResponse{Error: "invalid request: " + err.Error()})
	}

	result, err := c.SearchRecords(context.Background(), &req)
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

// streamResult is the subset of streaming.StreamResult[T] this file needs,
// declared locally to avoid importing the streaming package's generic type
// parameter machinery into every call site.
type streamResult[T any] interface {
	ResCh() <-chan *T
	ErrCh() <-chan error
	DoneCh() <-chan struct{}
}

// drainStream reads a streaming.StreamResult to completion, following the
// same select-over-channels pattern used by client/store.go's PullBatch and
// LookupBatch, and returns every received item along with any joined errors.
func drainStream[T any](result streamResult[T]) ([]*T, error) {
	var (
		items []*T
		errs  error
	)

	for {
		select {
		case err := <-result.ErrCh():
			errs = errors.Join(errs, err)
		case item := <-result.ResCh():
			items = append(items, item)
		case <-result.DoneCh():
			return items, errs
		}
	}
}
