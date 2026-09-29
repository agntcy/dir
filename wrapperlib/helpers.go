// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import "C"

import (
	"encoding/json"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// unmarshalProto parses jsonStr into msg using protojson. Unknown fields are
// discarded so request envelopes can carry extra plain-JSON fields (e.g. a
// CID) alongside a proto sub-message without failing to parse.
func unmarshalProto(jsonStr string, msg proto.Message) error {
	return protojson.UnmarshalOptions{DiscardUnknown: true}.Unmarshal([]byte(jsonStr), msg)
}

// marshalProto renders msg as a JSON string using protojson.
func marshalProto(msg proto.Message) (string, error) {
	b, err := protojson.Marshal(msg)
	if err != nil {
		return "", err //nolint:wrapcheck
	}

	return string(b), nil
}

// protoToRaw marshals msg with protojson into a json.RawMessage, suitable
// for splicing into a response envelope struct.
func protoToRaw(msg proto.Message) (json.RawMessage, error) {
	b, err := protojson.Marshal(msg)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	return json.RawMessage(b), nil
}

// unmarshalProtoSlice parses a slice of protojson-encoded messages into a
// slice of newly allocated *T values. T is the proto message struct type
// (e.g. corev1.RecordRef) and PT is its pointer type, which must implement
// proto.Message -- this indirection is required because Go generics have no
// "new T that implements interface I" shorthand.
func unmarshalProtoSlice[T any, PT interface {
	*T
	proto.Message
}](raws []json.RawMessage,
) ([]PT, error) {
	out := make([]PT, 0, len(raws))

	for _, raw := range raws {
		var v T

		p := PT(&v)
		if err := unmarshalProto(string(raw), p); err != nil {
			return nil, err
		}

		out = append(out, p)
	}

	return out, nil
}

// marshalProtoSlice renders a slice of proto messages as a slice of
// protojson-encoded json.RawMessage values, for splicing into an
// arrayResponse-shaped envelope.
func marshalProtoSlice[T proto.Message](items []T) ([]json.RawMessage, error) {
	out := make([]json.RawMessage, 0, len(items))

	for _, item := range items {
		b, err := protojson.Marshal(item)
		if err != nil {
			return nil, err //nolint:wrapcheck
		}

		out = append(out, json.RawMessage(b))
	}

	return out, nil
}

// arrayResponse is the common envelope for "drain to array" streaming calls:
// each element of Results is a protojson-encoded message.
type arrayResponse struct {
	Results []json.RawMessage `json:"results,omitempty"`
	Error   string            `json:"error,omitempty"`
}

// errorResponse is the common envelope for calls that only report success or
// failure (no payload on success).
type errorResponse struct {
	Error string `json:"error,omitempty"`
}

// toC marshals v to JSON and returns it as a newly allocated C string.
// Callers must release the result with FreeCString.
func toC(v any) *C.char {
	b, err := json.Marshal(v)
	if err != nil {
		return C.CString(`{"error":"failed to marshal response"}`)
	}

	return C.CString(string(b))
}
