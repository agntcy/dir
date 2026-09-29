// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import "C"

import (
	"context"
	"encoding/json"

	corev1 "github.com/agntcy/dir/api/core/v1"
	storev1 "github.com/agntcy/dir/api/store/v1"
)

// --- Push ---

type recordResponse struct {
	Ref   json.RawMessage `json:"ref,omitempty"`
	Error string          `json:"error,omitempty"`
}

// Push sends a single protojson-encoded corev1.Record to the store and
// returns a protojson-encoded corev1.RecordRef.
//
//export Push
func Push(handle C.longlong, recordJSON *C.char) *C.char {
	c, ok := lookupClient(int64(handle))
	if !ok {
		return toC(recordResponse{Error: "unknown or closed client handle"})
	}

	var record corev1.Record
	if err := unmarshalProto(C.GoString(recordJSON), &record); err != nil {
		return toC(recordResponse{Error: "invalid request: " + err.Error()})
	}

	ref, err := c.Push(context.Background(), &record)
	if err != nil {
		return toC(recordResponse{Error: err.Error()})
	}

	raw, err := protoToRaw(ref)
	if err != nil {
		return toC(recordResponse{Error: err.Error()})
	}

	return toC(recordResponse{Ref: raw})
}

// --- PushBatch ---

type recordsRequest struct {
	Records []json.RawMessage `json:"records"`
}

// PushBatch sends multiple protojson-encoded corev1.Record values (request
// field "records") and returns an arrayResponse whose Results are
// protojson-encoded corev1.RecordRef values.
//
//export PushBatch
func PushBatch(handle C.longlong, reqJSON *C.char) *C.char {
	c, ok := lookupClient(int64(handle))
	if !ok {
		return toC(arrayResponse{Error: "unknown or closed client handle"})
	}

	var req recordsRequest
	if err := json.Unmarshal([]byte(C.GoString(reqJSON)), &req); err != nil {
		return toC(arrayResponse{Error: "invalid request: " + err.Error()})
	}

	records, err := unmarshalProtoSlice[corev1.Record, *corev1.Record](req.Records)
	if err != nil {
		return toC(arrayResponse{Error: "invalid request: " + err.Error()})
	}

	refs, err := c.PushBatch(context.Background(), records)
	if err != nil {
		return toC(arrayResponse{Error: err.Error()})
	}

	results, err := marshalProtoSlice(refs)
	if err != nil {
		return toC(arrayResponse{Error: err.Error()})
	}

	return toC(arrayResponse{Results: results})
}

// --- Pull ---

type recordFullResponse struct {
	Record json.RawMessage `json:"record,omitempty"`
	Error  string          `json:"error,omitempty"`
}

// Pull retrieves a single record given a protojson-encoded corev1.RecordRef
// and returns a protojson-encoded corev1.Record.
//
//export Pull
func Pull(handle C.longlong, recordRefJSON *C.char) *C.char {
	c, ok := lookupClient(int64(handle))
	if !ok {
		return toC(recordFullResponse{Error: "unknown or closed client handle"})
	}

	var ref corev1.RecordRef
	if err := unmarshalProto(C.GoString(recordRefJSON), &ref); err != nil {
		return toC(recordFullResponse{Error: "invalid request: " + err.Error()})
	}

	record, err := c.Pull(context.Background(), &ref)
	if err != nil {
		return toC(recordFullResponse{Error: err.Error()})
	}

	raw, err := protoToRaw(record)
	if err != nil {
		return toC(recordFullResponse{Error: err.Error()})
	}

	return toC(recordFullResponse{Record: raw})
}

// --- PullBatch ---

type recordRefsRequest struct {
	RecordRefs []json.RawMessage `json:"record_refs"`
}

// PullBatch retrieves multiple records given protojson-encoded
// corev1.RecordRef values (request field "record_refs") and returns an
// arrayResponse whose Results are protojson-encoded corev1.Record values.
//
//export PullBatch
func PullBatch(handle C.longlong, reqJSON *C.char) *C.char {
	c, ok := lookupClient(int64(handle))
	if !ok {
		return toC(arrayResponse{Error: "unknown or closed client handle"})
	}

	var req recordRefsRequest
	if err := json.Unmarshal([]byte(C.GoString(reqJSON)), &req); err != nil {
		return toC(arrayResponse{Error: "invalid request: " + err.Error()})
	}

	refs, err := unmarshalProtoSlice[corev1.RecordRef, *corev1.RecordRef](req.RecordRefs)
	if err != nil {
		return toC(arrayResponse{Error: "invalid request: " + err.Error()})
	}

	records, err := c.PullBatch(context.Background(), refs)
	if err != nil {
		return toC(arrayResponse{Error: err.Error()})
	}

	results, err := marshalProtoSlice(records)
	if err != nil {
		return toC(arrayResponse{Error: err.Error()})
	}

	return toC(arrayResponse{Results: results})
}

// --- PushReferrer ---

type pushReferrerResponse struct {
	Response json.RawMessage `json:"response,omitempty"`
	Error    string          `json:"error,omitempty"`
}

// PushReferrer stores a protojson-encoded storev1.PushReferrerRequest and
// returns a protojson-encoded storev1.PushReferrerResponse.
//
//export PushReferrer
func PushReferrer(handle C.longlong, reqJSON *C.char) *C.char {
	c, ok := lookupClient(int64(handle))
	if !ok {
		return toC(pushReferrerResponse{Error: "unknown or closed client handle"})
	}

	var req storev1.PushReferrerRequest
	if err := unmarshalProto(C.GoString(reqJSON), &req); err != nil {
		return toC(pushReferrerResponse{Error: "invalid request: " + err.Error()})
	}

	resp, err := c.PushReferrer(context.Background(), &req)
	if err != nil {
		return toC(pushReferrerResponse{Error: err.Error()})
	}

	raw, err := protoToRaw(resp)
	if err != nil {
		return toC(pushReferrerResponse{Error: err.Error()})
	}

	return toC(pushReferrerResponse{Response: raw})
}

// --- PullReferrer ---

// PullReferrer sends a protojson-encoded storev1.PullReferrerRequest and
// returns an arrayResponse whose Results are protojson-encoded
// storev1.PullReferrerResponse values (the RPC is server-streaming;
// responses are drained into a slice by the underlying client method).
//
//export PullReferrer
func PullReferrer(handle C.longlong, reqJSON *C.char) *C.char {
	c, ok := lookupClient(int64(handle))
	if !ok {
		return toC(arrayResponse{Error: "unknown or closed client handle"})
	}

	var req storev1.PullReferrerRequest
	if err := unmarshalProto(C.GoString(reqJSON), &req); err != nil {
		return toC(arrayResponse{Error: "invalid request: " + err.Error()})
	}

	responses, err := c.PullReferrer(context.Background(), &req)
	if err != nil {
		return toC(arrayResponse{Error: err.Error()})
	}

	results, err := marshalProtoSlice(responses)
	if err != nil {
		return toC(arrayResponse{Error: err.Error()})
	}

	return toC(arrayResponse{Results: results})
}

// --- Lookup ---

type recordMetaResponse struct {
	Meta  json.RawMessage `json:"meta,omitempty"`
	Error string          `json:"error,omitempty"`
}

// Lookup retrieves metadata for a single record given a protojson-encoded
// corev1.RecordRef and returns a protojson-encoded corev1.RecordMeta.
//
//export Lookup
func Lookup(handle C.longlong, recordRefJSON *C.char) *C.char {
	c, ok := lookupClient(int64(handle))
	if !ok {
		return toC(recordMetaResponse{Error: "unknown or closed client handle"})
	}

	var ref corev1.RecordRef
	if err := unmarshalProto(C.GoString(recordRefJSON), &ref); err != nil {
		return toC(recordMetaResponse{Error: "invalid request: " + err.Error()})
	}

	meta, err := c.Lookup(context.Background(), &ref)
	if err != nil {
		return toC(recordMetaResponse{Error: err.Error()})
	}

	raw, err := protoToRaw(meta)
	if err != nil {
		return toC(recordMetaResponse{Error: err.Error()})
	}

	return toC(recordMetaResponse{Meta: raw})
}

// --- LookupBatch ---

// LookupBatch retrieves metadata for multiple records given protojson-encoded
// corev1.RecordRef values (request field "record_refs") and returns an
// arrayResponse whose Results are protojson-encoded corev1.RecordMeta values.
//
//export LookupBatch
func LookupBatch(handle C.longlong, reqJSON *C.char) *C.char {
	c, ok := lookupClient(int64(handle))
	if !ok {
		return toC(arrayResponse{Error: "unknown or closed client handle"})
	}

	var req recordRefsRequest
	if err := json.Unmarshal([]byte(C.GoString(reqJSON)), &req); err != nil {
		return toC(arrayResponse{Error: "invalid request: " + err.Error()})
	}

	refs, err := unmarshalProtoSlice[corev1.RecordRef, *corev1.RecordRef](req.RecordRefs)
	if err != nil {
		return toC(arrayResponse{Error: "invalid request: " + err.Error()})
	}

	metas, err := c.LookupBatch(context.Background(), refs)
	if err != nil {
		return toC(arrayResponse{Error: err.Error()})
	}

	results, err := marshalProtoSlice(metas)
	if err != nil {
		return toC(arrayResponse{Error: err.Error()})
	}

	return toC(arrayResponse{Results: results})
}

// --- Delete ---

// Delete removes a single record given a protojson-encoded corev1.RecordRef.
//
//export Delete
func Delete(handle C.longlong, recordRefJSON *C.char) *C.char {
	c, ok := lookupClient(int64(handle))
	if !ok {
		return toC(errorResponse{Error: "unknown or closed client handle"})
	}

	var ref corev1.RecordRef
	if err := unmarshalProto(C.GoString(recordRefJSON), &ref); err != nil {
		return toC(errorResponse{Error: "invalid request: " + err.Error()})
	}

	if err := c.Delete(context.Background(), &ref); err != nil {
		return toC(errorResponse{Error: err.Error()})
	}

	return toC(errorResponse{})
}

// --- DeleteBatch ---

// DeleteBatch removes multiple records given protojson-encoded
// corev1.RecordRef values (request field "record_refs").
//
//export DeleteBatch
func DeleteBatch(handle C.longlong, reqJSON *C.char) *C.char {
	c, ok := lookupClient(int64(handle))
	if !ok {
		return toC(errorResponse{Error: "unknown or closed client handle"})
	}

	var req recordRefsRequest
	if err := json.Unmarshal([]byte(C.GoString(reqJSON)), &req); err != nil {
		return toC(errorResponse{Error: "invalid request: " + err.Error()})
	}

	refs, err := unmarshalProtoSlice[corev1.RecordRef, *corev1.RecordRef](req.RecordRefs)
	if err != nil {
		return toC(errorResponse{Error: "invalid request: " + err.Error()})
	}

	if err := c.DeleteBatch(context.Background(), refs); err != nil {
		return toC(errorResponse{Error: err.Error()})
	}

	return toC(errorResponse{})
}

// --- DeleteReferrer ---

type deleteReferrerResponse struct {
	Response json.RawMessage `json:"response,omitempty"`
	Error    string          `json:"error,omitempty"`
}

// DeleteReferrer sends a protojson-encoded storev1.DeleteReferrerRequest and
// returns a protojson-encoded storev1.DeleteReferrerResponse.
//
//export DeleteReferrer
func DeleteReferrer(handle C.longlong, reqJSON *C.char) *C.char {
	c, ok := lookupClient(int64(handle))
	if !ok {
		return toC(deleteReferrerResponse{Error: "unknown or closed client handle"})
	}

	var req storev1.DeleteReferrerRequest
	if err := unmarshalProto(C.GoString(reqJSON), &req); err != nil {
		return toC(deleteReferrerResponse{Error: "invalid request: " + err.Error()})
	}

	resp, err := c.DeleteReferrer(context.Background(), &req)
	if err != nil {
		return toC(deleteReferrerResponse{Error: err.Error()})
	}

	raw, err := protoToRaw(resp)
	if err != nil {
		return toC(deleteReferrerResponse{Error: err.Error()})
	}

	return toC(deleteReferrerResponse{Response: raw})
}
