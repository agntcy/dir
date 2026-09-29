// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import "C"

import (
	"context"

	routingv1 "github.com/agntcy/dir/api/routing/v1"
)

// --- Publish ---

// Publish announces a record to the routing layer given a protojson-encoded
// routingv1.PublishRequest.
//
//export Publish
func Publish(handle C.longlong, reqJSON *C.char) *C.char {
	c, ok := lookupClient(int64(handle))
	if !ok {
		return toC(errorResponse{Error: "unknown or closed client handle"})
	}

	var req routingv1.PublishRequest
	if err := unmarshalProto(C.GoString(reqJSON), &req); err != nil {
		return toC(errorResponse{Error: "invalid request: " + err.Error()})
	}

	if err := c.Publish(context.Background(), &req); err != nil {
		return toC(errorResponse{Error: err.Error()})
	}

	return toC(errorResponse{})
}

// --- Unpublish ---

// Unpublish removes a record announcement given a protojson-encoded
// routingv1.UnpublishRequest.
//
//export Unpublish
func Unpublish(handle C.longlong, reqJSON *C.char) *C.char {
	c, ok := lookupClient(int64(handle))
	if !ok {
		return toC(errorResponse{Error: "unknown or closed client handle"})
	}

	var req routingv1.UnpublishRequest
	if err := unmarshalProto(C.GoString(reqJSON), &req); err != nil {
		return toC(errorResponse{Error: "invalid request: " + err.Error()})
	}

	if err := c.Unpublish(context.Background(), &req); err != nil {
		return toC(errorResponse{Error: err.Error()})
	}

	return toC(errorResponse{})
}

// --- List ---

// List sends a protojson-encoded routingv1.ListRequest and drains the
// resulting channel into an arrayResponse whose Results are
// protojson-encoded routingv1.ListResponse values.
//
//export List
func List(handle C.longlong, reqJSON *C.char) *C.char {
	c, ok := lookupClient(int64(handle))
	if !ok {
		return toC(arrayResponse{Error: "unknown or closed client handle"})
	}

	var req routingv1.ListRequest
	if err := unmarshalProto(C.GoString(reqJSON), &req); err != nil {
		return toC(arrayResponse{Error: "invalid request: " + err.Error()})
	}

	ch, err := c.List(context.Background(), &req)
	if err != nil {
		return toC(arrayResponse{Error: err.Error()})
	}

	items := make([]*routingv1.ListResponse, 0)
	for item := range ch {
		items = append(items, item)
	}

	results, err := marshalProtoSlice(items)
	if err != nil {
		return toC(arrayResponse{Error: err.Error()})
	}

	return toC(arrayResponse{Results: results})
}

// --- SearchRouting ---

// SearchRouting sends a protojson-encoded routingv1.SearchRequest and drains
// the resulting channel into an arrayResponse whose Results are
// protojson-encoded routingv1.SearchResponse values.
//
//export SearchRouting
func SearchRouting(handle C.longlong, reqJSON *C.char) *C.char {
	c, ok := lookupClient(int64(handle))
	if !ok {
		return toC(arrayResponse{Error: "unknown or closed client handle"})
	}

	var req routingv1.SearchRequest
	if err := unmarshalProto(C.GoString(reqJSON), &req); err != nil {
		return toC(arrayResponse{Error: "invalid request: " + err.Error()})
	}

	ch, err := c.SearchRouting(context.Background(), &req)
	if err != nil {
		return toC(arrayResponse{Error: err.Error()})
	}

	items := make([]*routingv1.SearchResponse, 0)
	for item := range ch {
		items = append(items, item)
	}

	results, err := marshalProtoSlice(items)
	if err != nil {
		return toC(arrayResponse{Error: err.Error()})
	}

	return toC(arrayResponse{Results: results})
}
