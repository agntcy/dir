// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"strings"
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
)

func TestUnmarshalProto(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		var ref corev1.RecordRef
		if err := unmarshalProto(`{"cid":"baecid123"}`, &ref); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if ref.GetCid() != "baecid123" {
			t.Fatalf("got cid %q, want baecid123", ref.GetCid())
		}
	})

	t.Run("discards unknown fields", func(t *testing.T) {
		var ref corev1.RecordRef
		if err := unmarshalProto(`{"cid":"baecid123","extra_field":"ignored"}`, &ref); err != nil {
			t.Fatalf("unexpected error for unknown field: %v", err)
		}

		if ref.GetCid() != "baecid123" {
			t.Fatalf("got cid %q, want baecid123", ref.GetCid())
		}
	})

	t.Run("malformed JSON", func(t *testing.T) {
		var ref corev1.RecordRef
		if err := unmarshalProto(`{not valid json`, &ref); err == nil {
			t.Fatal("expected an error for malformed JSON, got nil")
		}
	})
}

func TestMarshalProto(t *testing.T) {
	ref := &corev1.RecordRef{Cid: "baecid123"}

	out, err := marshalProto(ref)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "baecid123") {
		t.Fatalf("expected marshaled output to contain the cid, got %q", out)
	}
}

func TestProtoToRaw(t *testing.T) {
	ref := &corev1.RecordRef{Cid: "baecid123"}

	raw, err := protoToRaw(ref)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("raw message is not valid JSON: %v", err)
	}

	if decoded["cid"] != "baecid123" {
		t.Fatalf("got cid %v, want baecid123", decoded["cid"])
	}
}

func TestUnmarshalProtoSlice(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		raws := []json.RawMessage{
			json.RawMessage(`{"cid":"cid-1"}`),
			json.RawMessage(`{"cid":"cid-2"}`),
		}

		refs, err := unmarshalProtoSlice[corev1.RecordRef, *corev1.RecordRef](raws)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(refs) != 2 {
			t.Fatalf("got %d refs, want 2", len(refs))
		}

		if refs[0].GetCid() != "cid-1" || refs[1].GetCid() != "cid-2" {
			t.Fatalf("unexpected ref contents: %+v", refs)
		}
	})

	t.Run("malformed element", func(t *testing.T) {
		raws := []json.RawMessage{json.RawMessage(`{not valid`)}

		if _, err := unmarshalProtoSlice[corev1.RecordRef, *corev1.RecordRef](raws); err == nil {
			t.Fatal("expected an error for a malformed element, got nil")
		}
	})
}

func TestMarshalProtoSlice(t *testing.T) {
	refs := []*corev1.RecordRef{{Cid: "cid-1"}, {Cid: "cid-2"}}

	results, err := marshalProtoSlice(refs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}

	for i, want := range []string{"cid-1", "cid-2"} {
		var decoded map[string]any
		if err := json.Unmarshal(results[i], &decoded); err != nil {
			t.Fatalf("result %d is not valid JSON: %v", i, err)
		}

		if decoded["cid"] != want {
			t.Fatalf("result %d: got cid %v, want %v", i, decoded["cid"], want)
		}
	}
}

func TestToC(t *testing.T) {
	resp := toC(errorResponse{Error: "boom"})
	defer FreeCString(resp)

	if resp == nil {
		t.Fatal("toC returned nil")
	}
}
