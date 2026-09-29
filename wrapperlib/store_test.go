// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"strings"
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	storev1 "github.com/agntcy/dir/api/store/v1"
)

func recordJSONFor(t *testing.T, name string) string {
	t.Helper()

	data := mustStruct(map[string]any{"name": name, "schema_version": "v0.7.0"})

	return mustProtoJSON(t, &corev1.Record{Data: data})
}

func recordRefJSONFor(t *testing.T, cid string) string {
	t.Helper()

	return mustProtoJSON(t, &corev1.RecordRef{Cid: cid})
}

func TestPush(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	handle := newTestClientHandle(t, addr)

	t.Run("happy path", func(t *testing.T) {
		resp := bridgeHandle(Push, handle, recordJSONFor(t, "agent-1"))

		var out recordResponse
		if err := json.Unmarshal([]byte(resp), &out); err != nil {
			t.Fatalf("response is not valid JSON: %v (%s)", err, resp)
		}

		if out.Error != "" {
			t.Fatalf("unexpected error: %s", out.Error)
		}

		var ref corev1.RecordRef
		mustProtoUnmarshal(t, out.Ref, &ref)

		if !strings.HasPrefix(ref.GetCid(), "baecid-push-") {
			t.Fatalf("got cid %q, want a baecid-push- prefixed cid", ref.GetCid())
		}
	})

	t.Run("invalid request JSON", func(t *testing.T) {
		assertErrorNonEmpty(t, bridgeHandle(Push, handle, `{not valid`))
	})

	t.Run("unknown handle", func(t *testing.T) {
		assertErrorNonEmpty(t, bridgeHandle(Push, 999999, recordJSONFor(t, "agent-1")))
	})
}

func TestPushBatch(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	handle := newTestClientHandle(t, addr)

	t.Run("happy path", func(t *testing.T) {
		req := `{"records":[` + recordJSONFor(t, "agent-1") + "," + recordJSONFor(t, "agent-2") + `]}`

		resp := bridgeHandle(PushBatch, handle, req)

		var out arrayResponse
		if err := json.Unmarshal([]byte(resp), &out); err != nil {
			t.Fatalf("response is not valid JSON: %v (%s)", err, resp)
		}

		if out.Error != "" {
			t.Fatalf("unexpected error: %s", out.Error)
		}

		if len(out.Results) != 2 {
			t.Fatalf("got %d results, want 2", len(out.Results))
		}

		seen := map[string]bool{}

		for _, raw := range out.Results {
			var ref corev1.RecordRef
			if err := json.Unmarshal(raw, &ref); err != nil {
				t.Fatalf("result is not valid JSON: %v", err)
			}

			seen[ref.GetCid()] = true
		}

		if len(seen) != 2 {
			t.Fatalf("expected 2 distinct cids, got %v", seen)
		}
	})

	t.Run("invalid request JSON", func(t *testing.T) {
		assertErrorNonEmpty(t, bridgeHandle(PushBatch, handle, `{not valid`))
	})

	t.Run("unknown handle", func(t *testing.T) {
		assertErrorNonEmpty(t, bridgeHandle(PushBatch, 999999, `{"records":[]}`))
	})
}

func TestPull(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	handle := newTestClientHandle(t, addr)

	t.Run("happy path", func(t *testing.T) {
		resp := bridgeHandle(Pull, handle, recordRefJSONFor(t, "baecid123"))

		var out recordFullResponse
		if err := json.Unmarshal([]byte(resp), &out); err != nil {
			t.Fatalf("response is not valid JSON: %v (%s)", err, resp)
		}

		if out.Error != "" {
			t.Fatalf("unexpected error: %s", out.Error)
		}

		var rec corev1.Record
		mustProtoUnmarshal(t, out.Record, &rec)

		if rec.GetData().AsMap()["name"] != "test-record" {
			t.Fatalf("got record data %v, want name=test-record", rec.GetData().AsMap())
		}
	})

	t.Run("invalid request JSON", func(t *testing.T) {
		assertErrorNonEmpty(t, bridgeHandle(Pull, handle, `{not valid`))
	})

	t.Run("unknown handle", func(t *testing.T) {
		assertErrorNonEmpty(t, bridgeHandle(Pull, 999999, recordRefJSONFor(t, "baecid123")))
	})
}

func TestPullBatch(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	handle := newTestClientHandle(t, addr)

	t.Run("happy path", func(t *testing.T) {
		req := `{"record_refs":[` + recordRefJSONFor(t, "cid-1") + "," + recordRefJSONFor(t, "cid-2") + `]}`

		resp := bridgeHandle(PullBatch, handle, req)

		var out arrayResponse
		if err := json.Unmarshal([]byte(resp), &out); err != nil {
			t.Fatalf("response is not valid JSON: %v (%s)", err, resp)
		}

		if out.Error != "" {
			t.Fatalf("unexpected error: %s", out.Error)
		}

		if len(out.Results) != 2 {
			t.Fatalf("got %d results, want 2", len(out.Results))
		}
	})

	t.Run("invalid request JSON", func(t *testing.T) {
		assertErrorNonEmpty(t, bridgeHandle(PullBatch, handle, `{not valid`))
	})

	t.Run("unknown handle", func(t *testing.T) {
		assertErrorNonEmpty(t, bridgeHandle(PullBatch, 999999, `{"record_refs":[]}`))
	})
}

func TestLookup(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	handle := newTestClientHandle(t, addr)

	t.Run("happy path", func(t *testing.T) {
		resp := bridgeHandle(Lookup, handle, recordRefJSONFor(t, "baecid123"))

		var out recordMetaResponse
		if err := json.Unmarshal([]byte(resp), &out); err != nil {
			t.Fatalf("response is not valid JSON: %v (%s)", err, resp)
		}

		if out.Error != "" {
			t.Fatalf("unexpected error: %s", out.Error)
		}

		var meta corev1.RecordMeta
		mustProtoUnmarshal(t, out.Meta, &meta)

		if meta.GetCid() != "baecid123" {
			t.Fatalf("got cid %q, want baecid123", meta.GetCid())
		}
	})

	t.Run("invalid request JSON", func(t *testing.T) {
		assertErrorNonEmpty(t, bridgeHandle(Lookup, handle, `{not valid`))
	})

	t.Run("unknown handle", func(t *testing.T) {
		assertErrorNonEmpty(t, bridgeHandle(Lookup, 999999, recordRefJSONFor(t, "baecid123")))
	})
}

func TestLookupBatch(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	handle := newTestClientHandle(t, addr)

	t.Run("happy path", func(t *testing.T) {
		req := `{"record_refs":[` + recordRefJSONFor(t, "cid-1") + "," + recordRefJSONFor(t, "cid-2") + `]}`

		resp := bridgeHandle(LookupBatch, handle, req)

		var out arrayResponse
		if err := json.Unmarshal([]byte(resp), &out); err != nil {
			t.Fatalf("response is not valid JSON: %v (%s)", err, resp)
		}

		if out.Error != "" {
			t.Fatalf("unexpected error: %s", out.Error)
		}

		if len(out.Results) != 2 {
			t.Fatalf("got %d results, want 2", len(out.Results))
		}
	})

	t.Run("invalid request JSON", func(t *testing.T) {
		assertErrorNonEmpty(t, bridgeHandle(LookupBatch, handle, `{not valid`))
	})

	t.Run("unknown handle", func(t *testing.T) {
		assertErrorNonEmpty(t, bridgeHandle(LookupBatch, 999999, `{"record_refs":[]}`))
	})
}

func TestDelete(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	handle := newTestClientHandle(t, addr)

	t.Run("happy path", func(t *testing.T) {
		assertNoError(t, bridgeHandle(Delete, handle, recordRefJSONFor(t, "baecid123")))
	})

	t.Run("invalid request JSON", func(t *testing.T) {
		assertErrorNonEmpty(t, bridgeHandle(Delete, handle, `{not valid`))
	})

	t.Run("unknown handle", func(t *testing.T) {
		assertErrorNonEmpty(t, bridgeHandle(Delete, 999999, recordRefJSONFor(t, "baecid123")))
	})
}

func TestDeleteBatch(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	handle := newTestClientHandle(t, addr)

	t.Run("happy path", func(t *testing.T) {
		req := `{"record_refs":[` + recordRefJSONFor(t, "cid-1") + "," + recordRefJSONFor(t, "cid-2") + `]}`
		assertNoError(t, bridgeHandle(DeleteBatch, handle, req))
	})

	t.Run("invalid request JSON", func(t *testing.T) {
		assertErrorNonEmpty(t, bridgeHandle(DeleteBatch, handle, `{not valid`))
	})

	t.Run("unknown handle", func(t *testing.T) {
		assertErrorNonEmpty(t, bridgeHandle(DeleteBatch, 999999, `{"record_refs":[]}`))
	})
}

func TestPushReferrer(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	handle := newTestClientHandle(t, addr)

	t.Run("happy path", func(t *testing.T) {
		req := mustProtoJSON(t, &storev1.PushReferrerRequest{
			RecordRef: &corev1.RecordRef{Cid: "baecid123"},
			Type:      "agntcy.dir.sign.v1.Signature",
		})

		resp := bridgeHandle(PushReferrer, handle, req)

		var out pushReferrerResponse
		if err := json.Unmarshal([]byte(resp), &out); err != nil {
			t.Fatalf("response is not valid JSON: %v (%s)", err, resp)
		}

		if out.Error != "" {
			t.Fatalf("unexpected error: %s", out.Error)
		}

		var pushResp storev1.PushReferrerResponse
		mustProtoUnmarshal(t, out.Response, &pushResp)

		if !pushResp.GetSuccess() {
			t.Fatal("expected success=true")
		}

		if pushResp.GetReferrerRef().GetCid() != "baecid123" {
			t.Fatalf("got referrer ref cid %q, want baecid123", pushResp.GetReferrerRef().GetCid())
		}
	})

	t.Run("invalid request JSON", func(t *testing.T) {
		assertErrorNonEmpty(t, bridgeHandle(PushReferrer, handle, `{not valid`))
	})

	t.Run("unknown handle", func(t *testing.T) {
		req := mustProtoJSON(t, &storev1.PushReferrerRequest{RecordRef: &corev1.RecordRef{Cid: "baecid123"}})
		assertErrorNonEmpty(t, bridgeHandle(PushReferrer, 999999, req))
	})
}

func TestPullReferrer(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	handle := newTestClientHandle(t, addr)

	t.Run("happy path", func(t *testing.T) {
		req := mustProtoJSON(t, &storev1.PullReferrerRequest{RecordRef: &corev1.RecordRef{Cid: "baecid123"}})

		resp := bridgeHandle(PullReferrer, handle, req)

		var out arrayResponse
		if err := json.Unmarshal([]byte(resp), &out); err != nil {
			t.Fatalf("response is not valid JSON: %v (%s)", err, resp)
		}

		if out.Error != "" {
			t.Fatalf("unexpected error: %s", out.Error)
		}

		if len(out.Results) != 2 {
			t.Fatalf("got %d results, want 2 (signature + public key referrers)", len(out.Results))
		}
	})

	t.Run("invalid request JSON", func(t *testing.T) {
		assertErrorNonEmpty(t, bridgeHandle(PullReferrer, handle, `{not valid`))
	})

	t.Run("unknown handle", func(t *testing.T) {
		req := mustProtoJSON(t, &storev1.PullReferrerRequest{RecordRef: &corev1.RecordRef{Cid: "baecid123"}})
		assertErrorNonEmpty(t, bridgeHandle(PullReferrer, 999999, req))
	})
}

func TestDeleteReferrer(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	handle := newTestClientHandle(t, addr)

	t.Run("happy path", func(t *testing.T) {
		req := mustProtoJSON(t, &storev1.DeleteReferrerRequest{
			Record:      &corev1.RecordRef{Cid: "baecid123"},
			ReferrerRef: &corev1.ReferrerRef{Cid: "referrer-cid"},
		})

		resp := bridgeHandle(DeleteReferrer, handle, req)

		var out deleteReferrerResponse
		if err := json.Unmarshal([]byte(resp), &out); err != nil {
			t.Fatalf("response is not valid JSON: %v (%s)", err, resp)
		}

		if out.Error != "" {
			t.Fatalf("unexpected error: %s", out.Error)
		}

		var delResp storev1.DeleteReferrerResponse
		mustProtoUnmarshal(t, out.Response, &delResp)

		if len(delResp.GetReferrerRefs()) != 1 || delResp.GetReferrerRefs()[0].GetCid() != "referrer-cid" {
			t.Fatalf("unexpected referrer refs: %+v", delResp.GetReferrerRefs())
		}
	})

	t.Run("invalid request JSON", func(t *testing.T) {
		assertErrorNonEmpty(t, bridgeHandle(DeleteReferrer, handle, `{not valid`))
	})

	t.Run("unknown handle", func(t *testing.T) {
		req := mustProtoJSON(t, &storev1.DeleteReferrerRequest{Record: &corev1.RecordRef{Cid: "baecid123"}})
		assertErrorNonEmpty(t, bridgeHandle(DeleteReferrer, 999999, req))
	})
}
